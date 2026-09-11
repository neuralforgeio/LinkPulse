package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"linkpulse/internal/httpx"
)

const (
	resetOtpTTL         = 10 * time.Minute
	resetOtpMaxAttempts = 5
)

var errResetOtpInvalid = &httpx.UserError{
	Status:  http.StatusUnauthorized,
	Code:    httpx.CodeInvalidToken,
	Message: "invalid or expired code",
}

// hashResetOtp peppers the code with the JWT secret — a leaked DB
// alone cannot brute-force it.
func (s *Service) hashResetOtp(code string) string {
	sum := sha256.Sum256([]byte("reset:" + s.jwtSecret + ":" + code))
	return hex.EncodeToString(sum[:])
}

// generateResetOtpCode returns a uniformly random 6-digit code.
func generateResetOtpCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// RequestResetInput is the body for POST /auth/password/reset-request.
type RequestResetInput struct {
	Email string `json:"email"`
}

// RequestReset issues a reset code. The response is ALWAYS identical
// whether the email exists (PRD 9.1.7 anti-enumeration) — delivery
// happens here, not at verify time.
func (s *Service) RequestReset(ctx context.Context, email string) (rawCode string, ok bool) {
	var userID uuid.UUID
	err := s.db.QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1`, email,
	).Scan(&userID)
	if err != nil {
		return "", false
	}

	rawCode, err = generateResetOtpCode()
	if err != nil {
		return "", false
	}

	// One pending code per user.
	_, _ = s.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, userID)
	_, _ = s.db.Exec(ctx, `
        INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
        VALUES ($1, $2, $3, $4)`,
		uuid.New(), userID, s.hashResetOtp(rawCode), time.Now().Add(resetOtpTTL))

	return rawCode, true
}

// ResetConfirmInput is the body for POST /auth/password/reset-confirm.
type ResetConfirmInput struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

// ConfirmReset validates the code and sets the new password.
func (s *Service) ConfirmReset(ctx context.Context, in ResetConfirmInput) error {
	var (
		resetID   uuid.UUID
		userID    uuid.UUID
		codeHash  string
		expiresAt time.Time
		usedAt    *time.Time
	)
	err := s.db.QueryRow(ctx, `
        SELECT id, user_id, token_hash, expires_at, used_at
        FROM password_reset_tokens
        WHERE user_id = (SELECT id FROM users WHERE email = $1)
        ORDER BY created_at DESC
        LIMIT 1`, strings.ToLower(strings.TrimSpace(in.Email)),
	).Scan(&resetID, &userID, &codeHash, &expiresAt, &usedAt)
	if err != nil {
		return errResetOtpInvalid
	}

	if time.Now().After(expiresAt) || usedAt != nil {
		return errResetOtpInvalid
	}

	if s.hashResetOtp(strings.TrimSpace(in.Code)) != codeHash {
		return errResetOtpInvalid
	}

	if len(in.Password) < 8 || !hasLetter(in.Password) || !hasDigit(in.Password) {
		return &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: "password must be at least 8 characters with letters and numbers",
		}
	}

	newHash, err := HashPassword(in.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if _, err := s.db.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
		newHash, userID); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if _, err := s.db.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, resetID); err != nil {
		return fmt.Errorf("consume token: %w", err)
	}

	// PRD 9.1.8: revoke every session on reset.
	_, _ = s.db.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)

	return nil
}

// RequestResetHandler handles POST /api/v1/auth/password/reset-request.
// Always the same generic response (PRD 9.1.7).
func (h *Handler) RequestReset(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	var in RequestResetInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !emailRegex.MatchString(email) {
		// Generic response even for invalid emails.
		httpx.Success(w, http.StatusOK, map[string]string{
			"message": "If an account exists, a reset code has been sent.",
		})
		return
	}

	rawCode, ok := h.svc.RequestReset(r.Context(), email)
	if ok {
		if err := h.mailer.Send(r.Context(), email,
			"Your LinkPulse password reset code", otpEmailHTML(rawCode)); err != nil {
			h.log.Error("reset email send failed", "error", err)
			httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "could not send the code, try again")
			return
		}
		if h.mailer.InDevMode() {
			h.log.Info("reset otp code (dev mode)", "email", email, "code", rawCode)
		}
	}

	httpx.Success(w, http.StatusOK, map[string]string{
		"message": "If an account exists, a reset code has been sent.",
	})
}

// ConfirmResetHandler handles POST /api/v1/auth/password/reset-confirm.
func (h *Handler) ConfirmReset(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	var in ResetConfirmInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
		return
	}

	if err := h.svc.ConfirmReset(r.Context(), in); err != nil {
		var uerr *httpx.UserError
		if errors.As(err, &uerr) {
			httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
			return
		}
		h.log.Error("reset confirm failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
		return
	}

	httpx.Success(w, http.StatusOK, map[string]string{
		"message": "Password updated. Please sign in with your new password.",
	})
}
