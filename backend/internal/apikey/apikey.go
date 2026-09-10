package apikey

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/httpx"
    "linkpulse/internal/shortid"
    "linkpulse/internal/tenant"
)

// Scopes (PRD 9.7).
const (
    ScopeLinksRead     = "links:read"
    ScopeLinksWrite    = "links:write"
    ScopeAnalyticsRead = "analytics:read"
)

var allScopes = map[string]bool{
    ScopeLinksRead:     true,
    ScopeLinksWrite:    true,
    ScopeAnalyticsRead: true,
}

var (
    // ErrInvalidKey: unknown key (401).
    ErrInvalidKey = &httpx.UserError{
        Status:  http.StatusUnauthorized,
        Code:    httpx.CodeUnauthorized,
        Message: "invalid API key",
    }
    // ErrRevoked: the key was revoked (401).
    ErrRevoked = &httpx.UserError{
        Status:  http.StatusUnauthorized,
        Code:    httpx.CodeUnauthorized,
        Message: "this API key has been revoked",
    }
    // ErrNotFound: no such key in this workspace.
    ErrNotFound = &httpx.UserError{
        Status:  http.StatusNotFound,
        Code:    httpx.CodeNotFound,
        Message: "API key not found",
    }
)

// Service holds dependencies for API key flows.
type Service struct {
    db  *pgxpool.Pool
    log *slog.Logger
}

// NewService builds an apikey Service.
func NewService(db *pgxpool.Pool, log *slog.Logger) *Service {
    return &Service{db: db, log: log}
}

// CreateInput is the request body for POST .../api-keys.
type CreateInput struct {
    Name   string   `json:"name"`
    Scopes []string `json:"scopes"`
}

// CreateResult shows the full key exactly once (PRD 9.7.1).
type CreateResult struct {
    ID        uuid.UUID `json:"id"`
    Name      string    `json:"name"`
    Key       string    `json:"key"`
    KeyPrefix string    `json:"key_prefix"`
    Scopes    []string  `json:"scopes"`
    CreatedAt time.Time `json:"created_at"`
}

// KeyOut is the list shape — the raw key never appears here.
type KeyOut struct {
    ID         uuid.UUID  `json:"id"`
    Name       string     `json:"name"`
    KeyPrefix  string     `json:"key_prefix"`
    Scopes     []string   `json:"scopes"`
    LastUsedAt *time.Time `json:"last_used_at"`
    RevokedAt  *time.Time `json:"revoked_at"`
    CreatedAt  time.Time  `json:"created_at"`
}

// Create generates a new key: lp_live_<32 base62 chars>. Only the
// SHA-256 hash and a 12-char prefix are stored.
func (s *Service) Create(ctx context.Context, tenantID, userID uuid.UUID, in CreateInput) (CreateResult, error) {
    name := strings.TrimSpace(in.Name)
    if len(name) < 2 || len(name) > 100 {
        return CreateResult{}, &httpx.UserError{
            Status:  422, Code: httpx.CodeValidationError,
            Message: "name must be 2-100 characters",
        }
    }

    scopes, uerr := normalizeScopes(in.Scopes)
    if uerr != nil {
        return CreateResult{}, uerr
    }

    secret, err := shortid.New(32)
    if err != nil {
        return CreateResult{}, fmt.Errorf("generate key: %w", err)
    }
    key := "lp_live_" + secret

    keyID := uuid.New()
    _, err = s.db.Exec(ctx, `
        INSERT INTO api_keys (id, tenant_id, created_by, name, key_prefix, key_hash, scopes)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
        keyID, tenantID, userID, name, key[:12], hashKey(key), scopes)
    if err != nil {
        return CreateResult{}, fmt.Errorf("insert api key: %w", err)
    }

    return CreateResult{
        ID:        keyID,
        Name:      name,
        Key:       key,
        KeyPrefix: key[:12],
        Scopes:    scopes,
        CreatedAt: time.Now(),
    }, nil
}

// List returns the workspace's keys (no raw keys).
func (s *Service) List(ctx context.Context, tenantID uuid.UUID) ([]KeyOut, error) {
    rows, err := s.db.Query(ctx, `
        SELECT id, name, key_prefix, scopes, last_used_at, revoked_at, created_at
        FROM api_keys
        WHERE tenant_id = $1
        ORDER BY created_at DESC`, tenantID)
    if err != nil {
        return nil, fmt.Errorf("load api keys: %w", err)
    }
    defer rows.Close()

    keys := []KeyOut{}
    for rows.Next() {
        var k KeyOut
        if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.Scopes,
            &k.LastUsedAt, &k.RevokedAt, &k.CreatedAt); err != nil {
            return nil, fmt.Errorf("scan api key: %w", err)
        }
        keys = append(keys, k)
    }
    return keys, rows.Err()
}

// Revoke marks a key unusable. Revoked keys keep their history.
func (s *Service) Revoke(ctx context.Context, tenantID, keyID uuid.UUID) error {
    tag, err := s.db.Exec(ctx, `
        UPDATE api_keys SET revoked_at = NOW()
        WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`,
        keyID, tenantID)
    if err != nil {
        return fmt.Errorf("revoke api key: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return ErrNotFound
    }
    return nil
}

// Auth is the resolved identity for a public API request.
type Auth struct {
    KeyID     uuid.UUID
    TenantID  uuid.UUID
    CreatedBy uuid.UUID
    Scopes    []string
}

// HasScope reports whether the key carries a scope.
func (a *Auth) HasScope(scope string) bool {
    for _, s := range a.Scopes {
        if s == scope {
            return true
        }
    }
    return false
}

// Validate resolves a raw key into its tenant and scopes. Unknown and
// revoked keys are rejected. last_used_at refreshes at most once a
// minute — the WHERE clause self-throttles (PRD 9.7.4).
func (s *Service) Validate(ctx context.Context, rawKey string) (*Auth, error) {
    var (
        a         Auth
        revokedAt *time.Time
    )
    err := s.db.QueryRow(ctx, `
        SELECT id, tenant_id, created_by, scopes, revoked_at
        FROM api_keys
        WHERE key_hash = $1`, hashKey(rawKey),
    ).Scan(&a.KeyID, &a.TenantID, &a.CreatedBy, &a.Scopes, &revokedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrInvalidKey
    }
    if err != nil {
        return nil, fmt.Errorf("load api key: %w", err)
    }
    if revokedAt != nil {
        return nil, ErrRevoked
    }

    _, err = s.db.Exec(ctx, `
        UPDATE api_keys SET last_used_at = NOW()
        WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < NOW() - INTERVAL '1 minute')`,
        a.KeyID)
    if err != nil {
        // Usage tracking must never break the request itself.
        s.log.Error("api key last_used update failed", "error", err)
    }

    return &a, nil
}

type authKeyCtx struct{}

// RequireKey authenticates public API requests via the Bearer API key.
func (s *Service) RequireKey(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        const prefix = "Bearer "
        header := r.Header.Get("Authorization")
        if !strings.HasPrefix(header, prefix) {
            httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing API key")
            return
        }

        auth, err := s.Validate(r.Context(), strings.TrimPrefix(header, prefix))
        if err != nil {
            var uerr *httpx.UserError
            if errors.As(err, &uerr) {
                httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
            } else {
                s.log.Error("api key validate failed", "error", err)
                httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
            }
            return
        }

        ctx := context.WithValue(r.Context(), authKeyCtx{}, auth)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// RequireScope gates a route on a specific scope.
func (s *Service) RequireScope(scope string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth, ok := FromContext(r.Context())
            if !ok || !auth.HasScope(scope) {
                httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden,
                    "this API key lacks the required scope: "+scope)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// FromContext extracts the API key auth set by RequireKey.
func FromContext(ctx context.Context) (*Auth, bool) {
    a, ok := ctx.Value(authKeyCtx{}).(*Auth)
    return a, ok
}

func hashKey(key string) string {
    sum := sha256.Sum256([]byte(key))
    return hex.EncodeToString(sum[:])
}

func normalizeScopes(in []string) ([]string, *httpx.UserError) {
    if len(in) == 0 {
        return nil, &httpx.UserError{
            Status:  422, Code: httpx.CodeValidationError,
            Message: "at least one scope is required",
        }
    }
    seen := map[string]bool{}
    out := []string{}
    for _, sc := range in {
        sc = strings.TrimSpace(sc)
        if !allScopes[sc] {
            return nil, &httpx.UserError{
                Status:  422, Code: httpx.CodeValidationError,
                Message: "unknown scope: " + sc,
            }
        }
        if !seen[sc] {
            seen[sc] = true
            out = append(out, sc)
        }
    }
    return out, nil
}

// ---- Dashboard handlers (PRD 9.7.1–9.7.3, owner/admin only) ----

type Handler struct {
    svc *Service
    log *slog.Logger
}

// NewHandler builds the apikey HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
    return &Handler{svc: svc, log: log}
}

func (h *Handler) handleErr(w http.ResponseWriter, err error, action string) {
    var uerr *httpx.UserError
    if errors.As(err, &uerr) {
        httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
        return
    }
    h.log.Error(action + " failed", "error", err)
    httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
}

func (h *Handler) requireManager(w http.ResponseWriter, r *http.Request) bool {
    role, ok := tenant.RoleFrom(r.Context())
    if !ok || (role != tenant.RoleOwner && role != tenant.RoleAdmin) {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden,
            "requires owner or admin role")
        return true
    }
    return false
}

// Create handles POST /api/v1/tenants/{tenantId}/api-keys.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    if h.requireManager(w, r) {
        return
    }
    tenantID, okT := tenant.TenantIDFrom(r.Context())
    userID, okU := httpx.UserIDFrom(r.Context())
    if !okT || !okU {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 4096)
    var in CreateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.Create(r.Context(), tenantID, userID, in)
    if err != nil {
        h.handleErr(w, err, "api key create")
        return
    }
    httpx.Success(w, http.StatusCreated, res)
}

// List handles GET /api/v1/tenants/{tenantId}/api-keys.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    if h.requireManager(w, r) {
        return
    }
    tenantID, ok := tenant.TenantIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    res, err := h.svc.List(r.Context(), tenantID)
    if err != nil {
        h.handleErr(w, err, "api key list")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// Revoke handles DELETE /api/v1/tenants/{tenantId}/api-keys/{keyId}.
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
    if h.requireManager(w, r) {
        return
    }
    keyID, err := uuid.Parse(chi.URLParam(r, "keyId"))
    if err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid key id")
        return
    }
    tenantID, ok := tenant.TenantIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    if err := h.svc.Revoke(r.Context(), tenantID, keyID); err != nil {
        h.handleErr(w, err, "api key revoke")
        return
    }
    httpx.Success(w, http.StatusOK, map[string]string{"message": "API key revoked"})
}
