package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"linkpulse/internal/httpx"
)

// ErrInvalidSession is the generic refresh failure.
var ErrInvalidSession = &httpx.UserError{
	Status:  http.StatusUnauthorized,
	Code:    httpx.CodeInvalidToken,
	Message: "invalid session",
}

// reuseError signals that an already-revoked refresh token was replayed.
type reuseError struct {
	userID uuid.UUID
}

func (e *reuseError) Error() string { return "refresh token reuse detected" }

// RefreshResult is what the service layer produces on successful refresh.
type RefreshResult struct {
	AccessToken      string
	TokenType        string
	AccessExpiresAt  time.Time
	RefreshRaw       string
	RefreshExpiresAt time.Time
}

// refreshResponse is the JSON body for a successful refresh.
type refreshResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Refresh rotates the session (PRD 9.1.4).
func (s *Service) Refresh(ctx context.Context, rawToken string) (RefreshResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		tokenID   uuid.UUID
		userID    uuid.UUID
		expiresAt time.Time
		revokedAt *time.Time
	)
	err = tx.QueryRow(ctx, `
        SELECT id, user_id, expires_at, revoked_at
        FROM refresh_tokens
        WHERE token_hash = $1
        FOR UPDATE`, HashRefreshToken(rawToken),
	).Scan(&tokenID, &userID, &expiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshResult{}, ErrInvalidSession
	}
	if err != nil {
		return RefreshResult{}, fmt.Errorf("load refresh token: %w", err)
	}

	if revokedAt != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE refresh_tokens SET revoked_at = NOW()
             WHERE user_id = $1 AND revoked_at IS NULL`, userID,
		); err != nil {
			return RefreshResult{}, fmt.Errorf("revoke sessions: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return RefreshResult{}, fmt.Errorf("commit reuse revocation: %w", err)
		}
		return RefreshResult{}, &reuseError{userID: userID}
	}

	if time.Now().After(expiresAt) {
		return RefreshResult{}, ErrInvalidSession
	}

	if _, err := tx.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`, tokenID,
	); err != nil {
		return RefreshResult{}, fmt.Errorf("revoke old token: %w", err)
	}

	newRaw, newHash, err := NewRefreshToken()
	if err != nil {
		return RefreshResult{}, fmt.Errorf("create refresh token: %w", err)
	}
	refreshExpiresAt := time.Now().Add(s.refreshTTL)
	if _, err := tx.Exec(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		uuid.New(), userID, newHash, refreshExpiresAt,
	); err != nil {
		return RefreshResult{}, fmt.Errorf("store new refresh token: %w", err)
	}

	access, accessExpiresAt, err := NewAccessToken(s.jwtSecret, userID, s.accessTTL)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("create access token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return RefreshResult{}, fmt.Errorf("commit: %w", err)
	}

	return RefreshResult{
		AccessToken:      access,
		TokenType:        "Bearer",
		AccessExpiresAt:  accessExpiresAt,
		RefreshRaw:       newRaw,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

// Logout revokes the session tied to the presented refresh token (PRD
// 9.1.3). Unknown tokens are not an error — logout is idempotent.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	_, err := s.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW()
         WHERE token_hash = $1 AND revoked_at IS NULL`, HashRefreshToken(rawToken),
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// Refresh handles POST /api/v1/auth/refresh (cookie-authenticated).
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidToken, "missing refresh token")
		return
	}

	res, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		var reuse *reuseError
		if errors.As(err, &reuse) {
			h.log.Warn("refresh token reuse: all user sessions revoked",
				"user_id", reuse.userID)
			httpx.Error(w, http.StatusUnauthorized, httpx.CodeInvalidToken, "invalid session")
			return
		}
		var uerr *httpx.UserError
		if errors.As(err, &uerr) {
			httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
			return
		}
		h.log.Error("refresh failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
		return
	}

	h.setRefreshCookie(w, res.RefreshRaw)
	httpx.Success(w, http.StatusOK, refreshResponse{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
		ExpiresAt:   res.AccessExpiresAt,
	})
}

// Logout handles POST /api/v1/auth/logout (cookie-authenticated).
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		if err := h.svc.Logout(r.Context(), cookie.Value); err != nil {
			h.log.Error("logout failed", "error", err)
			httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
			return
		}
	}

	h.clearRefreshCookie(w)
	httpx.Success(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// cookieSameSite resolves the SameSite attribute for the deployment mode.
func (h *Handler) cookieSameSite() http.SameSite {
	if h.svc.cookieSameSiteNone {
		// Cross-domain deployment: the cookie must be None+Secure to
		// travel between the frontend and API domains.
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

// setRefreshCookie writes the refresh token cookie.
func (h *Handler) setRefreshCookie(w http.ResponseWriter, raw string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    raw,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.svc.cookieSecure,
		SameSite: h.cookieSameSite(),
		MaxAge:   int(h.svc.refreshTTL.Seconds()),
	})
}

// clearRefreshCookie deletes the refresh token cookie in the browser.
func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   h.svc.cookieSecure,
		SameSite: h.cookieSameSite(),
		MaxAge:   -1, // instructs the browser to delete the cookie
	})
}
