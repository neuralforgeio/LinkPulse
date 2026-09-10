package auth

import (
    "context"
    "crypto/rand"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "regexp"
    "strings"
    "unicode"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/httpx"
    "linkpulse/internal/tenant"
)

// uniqueViolation is PostgreSQL's error code for a UNIQUE constraint hit.
const uniqueViolation = "23505"

// Service holds dependencies for auth flows.
type Service struct {
    db *pgxpool.Pool
}

// NewService builds an auth Service.
func NewService(db *pgxpool.Pool) *Service {
    return &Service{db: db}
}

// RegisterInput is the request body for POST /api/v1/auth/register.
type RegisterInput struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

// UserOut is the public shape of a user. It never includes secrets.
type UserOut struct {
    ID    uuid.UUID `json:"id"`
    Name  string    `json:"name"`
    Email string    `json:"email"`
}

// TenantOut is the public shape of a workspace.
type TenantOut struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name"`
    Slug string    `json:"slug"`
}

// RegisterResult is the data returned on successful registration.
type RegisterResult struct {
    User   UserOut   `json:"user"`
    Tenant TenantOut `json:"tenant"`
}

// userError marks errors that map to a 4xx response instead of a 500.
type userError struct {
    status  int
    code    string
    message string
}

func (e *userError) Error() string { return e.message }

// ErrEmailTaken maps to 409 CONFLICT (PRD section 15).
var ErrEmailTaken = &userError{
    status:  http.StatusConflict,
    code:    httpx.CodeConflict,
    message: "email is already registered",
}

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Register creates a user, their default workspace, and an owner
// membership — all in ONE transaction, so a failure anywhere rolls
// everything back (PRD 9.1.1).
func (s *Service) Register(ctx context.Context, in RegisterInput) (RegisterResult, error) {
    name := strings.TrimSpace(in.Name)
    email := strings.ToLower(strings.TrimSpace(in.Email))

    hash, err := HashPassword(in.Password)
    if err != nil {
        return RegisterResult{}, fmt.Errorf("hash password: %w", err)
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return RegisterResult{}, fmt.Errorf("begin transaction: %w", err)
    }
    // No-op after a successful commit; rolls back on any early return.
    defer tx.Rollback(ctx)

    // Friendly pre-check. The UNIQUE constraint below is the real guard
    // against races (two registers with the same email at the same time).
    var exists bool
    err = tx.QueryRow(ctx,
        `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email,
    ).Scan(&exists)
    if err != nil {
        return RegisterResult{}, fmt.Errorf("check email: %w", err)
    }
    if exists {
        return RegisterResult{}, ErrEmailTaken
    }

    userID := uuid.New()
    _, err = tx.Exec(ctx,
        `INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)`,
        userID, name, email, hash,
    )
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
            return RegisterResult{}, ErrEmailTaken
        }
        return RegisterResult{}, fmt.Errorf("insert user: %w", err)
    }

    // Default workspace: "{name}'s Workspace" (PRD 9.1.1).
    tenantName := name + "'s Workspace"
    slug, err := uniqueSlug(ctx, tx, tenantName)
    if err != nil {
        return RegisterResult{}, fmt.Errorf("generate slug: %w", err)
    }

    tenantID := uuid.New()
    _, err = tx.Exec(ctx,
        `INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
        tenantID, tenantName, slug,
    )
    if err != nil {
        return RegisterResult{}, fmt.Errorf("insert tenant: %w", err)
    }

    _, err = tx.Exec(ctx,
        `INSERT INTO memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, 'owner')`,
        uuid.New(), tenantID, userID,
    )
    if err != nil {
        return RegisterResult{}, fmt.Errorf("insert membership: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return RegisterResult{}, fmt.Errorf("commit: %w", err)
    }

    return RegisterResult{
        User:   UserOut{ID: userID, Name: name, Email: email},
        Tenant: TenantOut{ID: tenantID, Name: tenantName, Slug: slug},
    }, nil
}

// uniqueSlug returns a slug for the tenant name, appending a short random
// suffix when the clean version is already taken (slugs are globally
// unique). Retries a few times before giving up.
func uniqueSlug(ctx context.Context, tx pgx.Tx, name string) (string, error) {
    base := tenant.Slugify(name)
    slug := base
    for attempt := 0; attempt < 3; attempt++ {
        var taken bool
        err := tx.QueryRow(ctx,
            `SELECT EXISTS (SELECT 1 FROM tenants WHERE slug = $1)`, slug,
        ).Scan(&taken)
        if err != nil {
            return "", fmt.Errorf("check slug: %w", err)
        }
        if !taken {
            return slug, nil
        }
        suffix, err := randomSuffix(4)
        if err != nil {
            return "", err
        }
        slug = base + "-" + suffix
    }
    return "", errors.New("could not build a unique slug")
}

// randomSuffix returns n random lowercase alphanumeric characters.
func randomSuffix(n int) (string, error) {
    const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
    raw := make([]byte, n)
    if _, err := rand.Read(raw); err != nil {
        return "", fmt.Errorf("read random bytes: %w", err)
    }
    out := make([]byte, n)
    for i, v := range raw {
        out[i] = alphabet[int(v)%len(alphabet)]
    }
    return string(out), nil
}

// validate enforces the register rules (PRD 9.1.1) and returns a list of
// human-readable problems. An empty list means valid.
func (in RegisterInput) validate() []string {
    var problems []string

    name := strings.TrimSpace(in.Name)
    email := strings.ToLower(strings.TrimSpace(in.Email))

    switch {
    case len(name) < 2:
        problems = append(problems, "name must be at least 2 characters")
    case len(name) > 100:
        problems = append(problems, "name must be at most 100 characters")
    }

    if !emailRegex.MatchString(email) {
        problems = append(problems, "email is not a valid address")
    }

    if len(in.Password) < 8 {
        problems = append(problems, "password must be at least 8 characters")
    }
    if !hasLetter(in.Password) || !hasDigit(in.Password) {
        problems = append(problems, "password must contain letters and numbers")
    }

    return problems
}

func hasLetter(s string) bool {
    for _, r := range s {
        if unicode.IsLetter(r) {
            return true
        }
    }
    return false
}

func hasDigit(s string) bool {
    for _, r := range s {
        if unicode.IsDigit(r) {
            return true
        }
    }
    return false
}

// Handler exposes auth flows over HTTP.
type Handler struct {
    svc *Service
    log *slog.Logger
}

// NewHandler builds the auth HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
    return &Handler{svc: svc, log: log}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
    // Reject oversized bodies up front: a register payload is tiny, so
    // anything bigger is suspicious (PRD 9.9 abuse prevention mindset).
    r.Body = http.MaxBytesReader(w, r.Body, 4096)

    var in RegisterInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBaqRequest, "invalid JSON body")
        return
    }

    if problems := in.validate(); len(problems) > 0 {
        httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError, "invalid input", problems...)
        return
    }

    res, err := h.svc.Register(r.Context(), in)
    if err != nil {
        var uerr *userError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.status, uerr.code, uerr.message)
            return
        }
        // Unexpected failure: log the details, return a generic message.
        // Never leak internals to the client.
        h.log.Error("register failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    httpx.Success(w, http.StatusCreated, res)
}
