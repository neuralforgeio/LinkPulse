package tenant

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
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/httpx"
)

// Roles (PRD 9.3 permission matrix).
const (
    RoleOwner  = "owner"
    RoleAdmin  = "admin"
    RoleMember = "member"
    RoleViewer = "viewer"
)

// uniqueViolation is PostgreSQL's error code for a UNIQUE constraint hit.
const uniqueViolation = "23505"

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var (
    // ErrNotFound: the workspace does not exist or is invisible to you.
    ErrNotFound = &httpx.UserError{
        Status:  http.StatusNotFound,
        Code:    httpx.CodeNotFound,
        Message: "workspace not found",
    }
    // ErrSlugTaken: the slug is already in use.
    ErrSlugTaken = &httpx.UserError{
        Status:  http.StatusConflict,
        Code:    httpx.CodeConflict,
        Message: "slug is already taken",
    }
)

// Service holds dependencies for tenant flows.
type Service struct {
    db  *pgxpool.Pool
    log *slog.Logger
}

// NewService builds a tenant Service.
func NewService(db *pgxpool.Pool, log *slog.Logger) *Service {
    return &Service{db: db, log: log}
}

// TenantSummary is a workspace as seen in lists, with the caller's role.
type TenantSummary struct {
    ID        uuid.UUID `json:"id"`
    Name      string    `json:"name"`
    Slug      string    `json:"slug"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
}

// TenantDetail is the full workspace record for a member.
type TenantDetail struct {
    ID          uuid.UUID       `json:"id"`
    Name        string          `json:"name"`
    Slug        string          `json:"slug"`
    Settings    json.RawMessage `json:"settings"`
    Role        string          `json:"role"`
    MemberCount int             `json:"member_count"`
    CreatedAt   time.Time       `json:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at"`
}

// CreateInput is the request body for POST /api/v1/tenants.
type CreateInput struct {
    Name string `json:"name"`
    Slug string `json:"slug"`
}

// UpdateInput is the request body for PATCH /api/v1/tenants/{tenantId}.
// Nil fields mean "leave unchanged" (partial update, PRD 9.2.4).
type UpdateInput struct {
    Name     *string          `json:"name"`
    Slug     *string          `json:"slug"`
    Settings *json.RawMessage `json:"settings"`
}

// Create makes a new workspace; the creator becomes its owner (PRD 9.2.1).
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in CreateInput) (TenantSummary, error) {
    name := strings.TrimSpace(in.Name)
    slugInput := strings.ToLower(strings.TrimSpace(in.Slug))

    if uerr := validateName(name); uerr != nil {
        return TenantSummary{}, uerr
    }
    if uerr := validateSlug(slugInput); uerr != nil {
        return TenantSummary{}, uerr
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return TenantSummary{}, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    base := slugInput
    if base == "" {
        base = Slugify(name)
    }
    slug, err := UniqueSlug(ctx, tx, base)
    if err != nil {
        return TenantSummary{}, fmt.Errorf("generate slug: %w", err)
    }

    tenantID := uuid.New()
    _, err = tx.Exec(ctx,
        `INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3)`,
        tenantID, name, slug)
    if err != nil {
        return TenantSummary{}, fmt.Errorf("insert tenant: %w", err)
    }

    _, err = tx.Exec(ctx,
        `INSERT INTO memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, $4)`,
        uuid.New(), tenantID, userID, RoleOwner)
    if err != nil {
        return TenantSummary{}, fmt.Errorf("insert membership: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return TenantSummary{}, fmt.Errorf("commit: %w", err)
    }

    return TenantSummary{
        ID:        tenantID,
        Name:      name,
        Slug:      slug,
        Role:      RoleOwner,
        CreatedAt: time.Now(),
    }, nil
}

// List returns every workspace the user belongs to, with their role
// (PRD 9.2.2).
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]TenantSummary, error) {
    rows, err := s.db.Query(ctx, `
        SELECT t.id, t.name, t.slug, m.role, t.created_at
        FROM memberships m
        JOIN tenants t ON t.id = m.tenant_id
        WHERE m.user_id = $1
        ORDER BY m.created_at ASC`, userID)
    if err != nil {
        return nil, fmt.Errorf("load tenants: %w", err)
    }
    defer rows.Close()

    tenants := []TenantSummary{}
    for rows.Next() {
        var t TenantSummary
        if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Role, &t.CreatedAt); err != nil {
            return nil, fmt.Errorf("scan tenant: %w", err)
        }
        tenants = append(tenants, t)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate tenants: %w", err)
    }
    return tenants, nil
}

// Get returns one workspace, visible only to its members (PRD 9.2.3).
func (s *Service) Get(ctx context.Context, userID, tenantID uuid.UUID) (TenantDetail, error) {
    var d TenantDetail
    err := s.db.QueryRow(ctx, `
        SELECT t.id, t.name, t.slug, t.settings, m.role,
               (SELECT COUNT(*) FROM memberships mc WHERE mc.tenant_id = t.id),
               t.created_at, t.updated_at
        FROM tenants t
        JOIN memberships m ON m.tenant_id = t.id AND m.user_id = $1
        WHERE t.id = $2`, userID, tenantID,
    ).Scan(&d.ID, &d.Name, &d.Slug, &d.Settings, &d.Role, &d.MemberCount, &d.CreatedAt, &d.UpdatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return TenantDetail{}, ErrNotFound
    }
    if err != nil {
        return TenantDetail{}, fmt.Errorf("load tenant: %w", err)
    }
    return d, nil
}

// Update modifies workspace fields (owner/admin only — enforced by the
// handler after the membership gate; PRD 9.2.4).
func (s *Service) Update(ctx context.Context, actorID, tenantID uuid.UUID, in UpdateInput) (TenantDetail, error) {
    name := strings.TrimSpace(deref(in.Name))
    slug := strings.ToLower(strings.TrimSpace(deref(in.Slug)))

    var nameArg, slugArg *string

    if in.Name != nil {
        if uerr := validateName(name); uerr != nil {
            return TenantDetail{}, uerr
        }
        nameArg = &name
    }

    if in.Slug != nil {
        if slug == "" {
            return TenantDetail{}, &httpx.UserError{
                Status:  http.StatusUnprocessableEntity,
                Code:    httpx.CodeValidationError,
                Message: "slug cannot be empty",
            }
        }
        if uerr := validateSlug(slug); uerr != nil {
            return TenantDetail{}, uerr
        }
        slugArg = &slug
    }

    if in.Settings != nil {
        if len(*in.Settings) > 4096 {
            return TenantDetail{}, &httpx.UserError{
                Status:  http.StatusUnprocessableEntity,
                Code:    httpx.CodeValidationError,
                Message: "settings too large (max 4 KB)",
            }
        }
        var obj map[string]any
        if err := json.Unmarshal(*in.Settings, &obj); err != nil {
            return TenantDetail{}, &httpx.UserError{
                Status:  http.StatusUnprocessableEntity,
                Code:    httpx.CodeValidationError,
                Message: "settings must be a JSON object",
            }
        }
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return TenantDetail{}, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    if slugArg != nil {
        var taken bool
        err := tx.QueryRow(ctx,
            `SELECT EXISTS (SELECT 1 FROM tenants WHERE slug = $1 AND id <> $2)`,
            slug, tenantID).Scan(&taken)
        if err != nil {
            return TenantDetail{}, fmt.Errorf("check slug: %w", err)
        }
        if taken {
            return TenantDetail{}, ErrSlugTaken
        }
    }

    // COALESCE keeps unset fields unchanged.
    tag, err := tx.Exec(ctx, `
        UPDATE tenants SET
            name = COALESCE($1, name),
            slug = COALESCE($2, slug),
            settings = COALESCE($3, settings),
            updated_at = NOW()
        WHERE id = $4`,
        nameArg, slugArg, in.Settings, tenantID)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
            return TenantDetail{}, ErrSlugTaken
        }
        return TenantDetail{}, fmt.Errorf("update tenant: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return TenantDetail{}, ErrNotFound
    }

    if err := tx.Commit(ctx); err != nil {
        return TenantDetail{}, fmt.Errorf("commit: %w", err)
    }

    return s.Get(ctx, actorID, tenantID)
}

// UniqueSlug returns a slug based on the given (already slugified) base,
// appending a short random suffix when the clean version is taken
// (slugs are globally unique). Register uses this too, for the default
// workspace created alongside a new account.
func UniqueSlug(ctx context.Context, tx pgx.Tx, base string) (string, error) {
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

func validateName(name string) *httpx.UserError {
    if len(name) < 2 || len(name) > 100 {
        return &httpx.UserError{
            Status:  http.StatusUnprocessableEntity,
            Code:    httpx.CodeValidationError,
            Message: "name must be 2-100 characters",
        }
    }
    return nil
}

func validateSlug(slug string) *httpx.UserError {
    if slug == "" {
        return nil // empty means "generate one from the name"
    }
    if len(slug) < 3 || len(slug) > 48 || !slugRegex.MatchString(slug) {
        return &httpx.UserError{
            Status:  http.StatusUnprocessableEntity,
            Code:    httpx.CodeValidationError,
            Message: "slug must be 3-48 chars: lowercase letters, digits, dashes",
        }
    }
    return nil
}

func deref(p *string) string {
    if p == nil {
        return ""
    }
    return *p
}
