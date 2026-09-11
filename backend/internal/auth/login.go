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

// refreshCookieName carries the refresh token.
const refreshCookieName = "lp_refresh_token"

// dummyHash equalizes login timing (PRD 9.1.2).
const dummyHash = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"

// ErrInvalidCredentials is the generic login failure (PRD 9.1.2).
var ErrInvalidCredentials = &httpx.UserError{
    Status:  http.StatusUnauthorized,
    Code:    httpx.CodeUnauthorized,
    Message: "invalid credentials",
}

// LoginInput is the request body for POST /api/v1/auth/login.
type LoginInput struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

// LoginResult is what the service layer produces.
// With OTP enabled, Login returns only the OTP step; the session fields
// are filled by VerifyLoginOtp (see otp.go).
type LoginResult struct {
    OtpRequired bool
    OtpRawCode  string
    OtpExpires  time.Time

    User             UserOut
    AccessToken      string
    TokenType        string
    AccessExpiresAt  time.Time
    RefreshRaw       string
    RefreshExpiresAt time.Time
}

// loginResponse is the JSON body for a COMPLETED login (token issued).
type loginResponse struct {
    User        UserOut   `json:"user"`
    AccessToken string    `json:"access_token"`
    TokenType   string    `json:"token_type"`
    ExpiresAt   time.Time `json:"expires_at"`
}

// otpResponse is the JSON body for the OTP-pending login step.
type otpResponse struct {
    OtpRequired bool      `json:"otp_required"`
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

// Login verifies the password and issues an OTP (PRD 9.1.2, extended
// with email OTP as a second factor). The session itself is only
// issued after the code is verified.
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

    // Password correct → issue the OTP step.
    code, expiresAt, err := s.CreateOtp(ctx, userID, email)
    if err != nil {
        return LoginResult{}, fmt.Errorf("create otp: %w", err)
    }
    return LoginResult{
        OtpRequired: true,
        OtpRawCode:  code,
        OtpExpires:  expiresAt,
    }, nil
}

// Me loads the user and every workspace they belong to (PRD 9.1.5).
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (MeResult, error) {
    var user UserOut
    err := s.db.QueryRow(ctx,
        `SELECT id, name, email FROM users WHERE id = $1`, userID,
    ).Scan(&user.ID, &user.Name, &user.Email)
    if errors.Is(err, pgx.ErrNoRows) {
        return MeResult{}, &httpx.UserError{
            Status:  http.StatusUnauthorized,
            Code:    httpx.CodeUnauthorized,
            Message: "user no longer exists",
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
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }
    if strings.TrimSpace(in.Email) == "" || in.Password == "" {
        httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid credentials")
        return
    }

    res, err := h.svc.Login(r.Context(), in)
    if err != nil {
        var uerr *httpx.UserError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
            return
        }
        h.log.Error("login failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    if res.OtpRequired {
        // Deliver the code — email when Resend is configured.
        if err := h.mailer.Send(r.Context(), res.User.Email,
            "Your LinkPulse login code", otpEmailHTML(res.OtpRawCode)); err != nil {
            h.log.Error("otp email send failed", "error", err)
            httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "could not send the code, try again")
            return
        }
        if h.mailer.InDevMode() {
            h.log.Info("otp code (dev mode — check this to log in)",
                "email", res.User.Email, "code", res.OtpRawCode)
        }

        httpx.Success(w, http.StatusOK, otpResponse{
            OtpRequired: true,
            ExpiresAt:   res.OtpExpires,
        })
        return
    }

    // Non-OTP path (kept for future OTP-disabled deployments).
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
    userID, ok := httpx.UserIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    res, err := h.svc.Me(r.Context(), userID)
    if err != nil {
        var uerr *httpx.UserError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
            return
        }
        h.log.Error("me failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    httpx.Success(w, http.StatusOK, res)
}
