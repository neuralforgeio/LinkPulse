package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"linkpulse/internal/httpx"
)

// refreshCookieName carries the refresh token. Path is scoped to the auth
// endpoints so the cookie is not sent on unrelated requests.
const refreshCookieName = "lp_refresh_token"

// dummyHash is a syntactically-valid Argon2id string used only to equalize
// timing: when the email is unknown, we still run a full verification so
// the response takes the same time as a wrong-password attempt. The
// comparison result is always discarded.
const dummyHash = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

// ErrInvalidCredentials is the generic login failure (PRD 9.1.2): the same
// message for unknown email and wrong password — no hints for attackers.
var ErrInvalidCredentials = &userError{
    status:  http.StatusUnauthorized,
    code:    httpx.CodeUnauthorized,
    message: "invalid credentials",
}

// LoginInput is the request body for POST /api/v1/auth/login.
type LoginInput struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// LoginResult is what the service layer produces on success.
type LoginResult struct {
    User             UserOut
    AccessToken      string
    TokenType        string
    AccessExpiresAt  time.Time
    RefreshRaw       string
    RefreshExpiresAt time.Time
}

// loginResponse is the JSON body. The refresh token is deliberately NOT
// here — it lives only in the HTTP-only cookie, unreachable from JavaScript.
type loginResponse struct {
    User        UserOut   `json:"user"`
    AccessToken string    `json:"access_token"`
    TokenType   string    `json:"token_type"`
    ExpiresAt   time.Time `json:"expires_at"`
}

// TenantMembership is one workspace the user belongs to.
type TenantMembership struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name"`
    Slug string    `json:"slug"`
    Role string    `json:"role"`
}

// MeResult is the response for GET /api/v1/auth/me (PRD 9.1.5).
type MeResult struct {
    User            UserOut            `json:"user"`
    Tenants         []TenantMembership `json:"tenants"`
    DefaultTenantID uuid.UUID          `json:"default_tenant_id"`
}

// Login authenticates the user and opens a session: one short-lived JWT
// access token (stateless) plus one long-lived refresh token (stateful,
// stored hashed, revocable — PRD 9.1.2).
func (s *Service) Login(ctx context.Context, in LoginInput) (LoginResult, error) {
    email := strings.ToLower(strings.TrimSpace(in.Email))

    var (
        userID uuid.UUID
        name   string
        hash   string
    )
    err := s.db.QueryRow(ctx,
        `SELECT id, name, password_hash FROM users WHERE email = $1`, email,
    ).Scan(&userID, &name, &hash)
    if errors.Is(err, pgx.ErrNoRows) {
        // Burn the same Argon2id cost as a real check (timing equalization).
        _, _ = VerifyPassword(in.Password, dummyHash)
        return LoginResult{}, ErrInvalidCredentials
    }
    if err != nil {
        return LoginResult{}, fmt.Errorf("load user: %w", err)
    }

    ok, err := VerifyPassword(in.Password, hash)
    if err != nil {
        return LoginResult{}, fmt.Errorf("verify password: %w", err)
    }
    if !ok {
        return LoginResult{}, ErrInvalidCredentials
    }

    // Session record: only the hash is stored. One row = one device/session.
    refreshRaw, refreshHash, err := NewRefreshToken()
    if err != nil {
        return LoginResult{}, fmt.Errorf("create refresh token: %w", err)
    }
    refreshExpiresAt := time.Now().Add(s.refreshTTL)
    _, err = s.db.Exec(ctx,
        `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
        uuid.New(), userID, refreshHash, refreshExpiresAt,
    )
    if err != nil {
        return LoginResult{}, fmt.Errorf("store refresh token: %w", err)
    }

    access, accessExpiresAt, err := NewAccessToken(s.jwtSecret, userID, s.accessTTL)
    if err != nil {
        return LoginResult{}, fmt.Errorf("create access token: %w", err)
    }

    return LoginResult{
        User:             UserOut{ID: userID, Name: name, Email: email},
        AccessToken:      access,
        TokenType:        "Bearer",
        AccessExpiresAt:  accessExpiresAt,
        RefreshRaw:       refreshRaw,
        RefreshExpiresAt: refreshExpiresAt,
    }, nil
}

// Me loads the user and every workspace they belong to, with their role
// (PRD 9.1.5). The earliest membership is treated as the default tenant.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (MeResult, error) {
    var user UserOut
    err := s.db.QueryRow(ctx,
        `SELECT id, name, email FROM users WHERE id = $1`, userID,
    ).Scan(&user.ID, &user.Name, &user.Email)
    if errors.Is(err, pgx.ErrNoRows) {
        return MeResult{}, &userError{
            status:  http.StatusUnauthorized,
            code:    httpx.CodeUnauthorized,
            message: "user no longer exists",
        }
    }
    if err != nil {
        return MeResult{}, fmt.Errorf("load user: %w", err)
    }

    rows, err := s.db.Query(ctx, `
        SELECT t.id, t.name, t.slug, m.role
        FROM memberships m
        JOIN tenants t ON t.id = m.tenant_id
        WHERE m.user_id = $1
        ORDER BY m.created_at ASC`, userID)
    if err != nil {
        return MeResult{}, fmt.Errorf("load tenants: %w", err)
    }
    defer rows.Close()

    result := MeResult{User: user, Tenants: []TenantMembership{}}
    for rows.Next() {
        var tm TenantMembership
        if err := rows.Scan(&tm.ID, &tm.Name, &tm.Slug, &tm.Role); err != nil {
            return MeResult{}, fmt.Errorf("scan tenant: %w", err)
        }
        if len(result.Tenants) == 0 {
            result.DefaultTenantID = tm.ID
        }
        result.Tenants = append(result.Tenants, tm)
    }
    if err := rows.Err(); err != nil {
        return MeResult{}, fmt.Errorf("iterate tenants: %w", err)
    }

    return result, nil
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 4096)

    var in LoginInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBaqRequest, "invalid JSON body")
        return
    }
    if strings.TrimSpace(in.Email) == "" || in.Password == "" {
        httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid credentials")
        return
    }

    res, err := h.svc.Login(r.Context(), in)
    if err != nil {
        var uerr *userError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.status, uerr.code, uerr.message)
            return
        }
        h.log.Error("login failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    // Refresh token → HTTP-only cookie (PRD 9.1.2, 16.4–16.6): JavaScript
    // cannot read it, so an XSS bug cannot steal the session.
    h.setRefreshCookie(w, res.RefreshRaw)

    httpx.Success(w, http.StatusOK, loginResponse{
        User:        res.User,
        AccessToken: res.AccessToken,
        TokenType:   res.TokenType,
        ExpiresAt:   res.AccessExpiresAt,
    })
}

// Me handles GET /api/v1/auth/me (requires a valid access token).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
    userID, ok := UserIDFrom(r.Context())
    if !ok {
        // Should never happen: RequireAuth runs before this handler.
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    res, err := h.svc.Me(r.Context(), userID)
    if err != nil {
        var uerr *userError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.status, uerr.code, uerr.message)
            return
        }
        h.log.Error("me failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    httpx.Success(w, http.StatusOK, res)
}
