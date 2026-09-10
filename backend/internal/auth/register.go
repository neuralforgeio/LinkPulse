package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"linkpulse/internal/httpx"
	"linkpulse/internal/tenant"
)

// uniqueViolation is PostgreSQL's error code for a UNIQUE constraint hit.
const uniqueViolation = "23505"

// ServiceConfig carries auth-specific settings from the environment.
type ServiceConfig struct {
    JWTSecret    string
    AccessTTL    time.Duration
    RefreshTTL   time.Duration
    CookieSecure bool
}

// Service holds dependencies for auth flows.
type Service struct {
    db           *pgxpool.Pool
    jwtSecret    string
    accessTTL    time.Duration
    refreshTTL   time.Duration
    cookieSecure bool
}

// NewService builds an auth Service.
func NewService(db *pgxpool.Pool, cfg ServiceConfig) *Service {
    return &Service{
        db:           db,
        jwtSecret:    cfg.JWTSecret,
        accessTTL:    cfg.AccessTTL,
        refreshTTL:   cfg.RefreshTTL,
        cookieSecure: cfg.CookieSecure,
    }
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

// ErrEmailTaken maps to 409 CONFLICT (PRD section 15).
var ErrEmailTaken = &httpx.UserError{
    Status:  http.StatusConflict,
    Code:    httpx.CodeConflict,
    Message: "email is already registered",
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

    // Default workspace: "{name}'s Workspace" (PRD 9.1.1). Slug generation
    // lives in the tenant module, its rightful home.
    tenantName := name + "'s Workspace"
    slug, err := tenant.UniqueSlug(ctx, tx, tenant.Slugify(tenantName))
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
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    if problems := in.validate(); len(problems) > 0 {
        httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError, "invalid input", problems...)
        return
    }

    res, err := h.svc.Register(r.Context(), in)
    if err != nil {
        var uerr *httpx.UserError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
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
