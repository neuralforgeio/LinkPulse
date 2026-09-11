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
    "github.com/jackc/pgx/v5"

    "linkpulse/internal/httpx"
)

const (
    otpTTL         = 10 * time.Minute
    otpMaxAttempts = 5
)

var ErrOtpInvalid = &httpx.UserError{
    Status:  http.StatusUnauthorized,
    Code:    httpx.CodeInvalidToken,
    Message: "invalid or expired code",
}

// hashOtp peppers the 6-digit code with the JWT secret so a leaked
// database cannot brute-force the 1M possible codes offline.
func (s *Service) hashOtp(code string) string {
    sum := sha256.Sum256([]byte("otp:" + s.jwtSecret + ":" + code))
    return hex.EncodeToString(sum[:])
}

// generateOtpCode returns a uniformly random 6-digit code.
func generateOtpCode() (string, error) {
    n, err := rand.Int(rand.Reader, big.NewInt(1000000))
    if err != nil {
        return "", fmt.Errorf("rand: %w", err)
    }
    return fmt.Sprintf("%06d", n.Int64()), nil
}

// CreateOtp issues a fresh code for a user, invalidating any pending
// codes for the same email. Returns the RAW code (for delivery) and
// the expiry — only the hash is stored.
func (s *Service) CreateOtp(ctx context.Context, userID uuid.UUID, email string) (string, time.Time, error) {
    code, err := generateOtpCode()
    if err != nil {
        return "", time.Time{}, err
    }
    expiresAt := time.Now().Add(otpTTL)

    // One pending code per email: retire older ones.
    _, err = s.db.Exec(ctx,
        `UPDATE otp_codes SET verified_at = NOW() WHERE email = $1 AND verified_at IS NULL`,
        email)
    if err != nil {
        return "", time.Time{}, fmt.Errorf("retire old otp codes: %w", err)
    }

    _, err = s.db.Exec(ctx, `
        INSERT INTO otp_codes (id, user_id, email, code_hash, expires_at)
        VALUES ($1, $2, $3, $4, $5)`,
        uuid.New(), userID, email, s.hashOtp(code), expiresAt)
    if err != nil {
        return "", time.Time{}, fmt.Errorf("insert otp: %w", err)
    }

    return code, expiresAt, nil
}

// VerifyLoginOtp completes the login: validates the code and issues
// the session (refresh token + access JWT).
func (s *Service) VerifyLoginOtp(ctx context.Context, email, code string) (LoginResult, error) {
    email = strings.ToLower(strings.TrimSpace(email))
    code = strings.TrimSpace(code)

    var (
        otpID     uuid.UUID
        userID    uuid.UUID
        codeHash  string
        expiresAt time.Time
        attempts  int
    )
    err := s.db.QueryRow(ctx, `
        SELECT id, user_id, code_hash, expires_at, attempts
        FROM otp_codes
        WHERE email = $1 AND verified_at IS NULL
        ORDER BY created_at DESC
        LIMIT 1`, email,
    ).Scan(&otpID, &userID, &codeHash, &expiresAt, &attempts)
    if errors.Is(err, pgx.ErrNoRows) {
        return LoginResult{}, ErrOtpInvalid
    }
    if err != nil {
        return LoginResult{}, fmt.Errorf("load otp: %w", err)
    }

    if time.Now().After(expiresAt) || attempts >= otpMaxAttempts {
        return LoginResult{}, ErrOtpInvalid
    }

    if s.hashOtp(code) != codeHash {
        // Count the failed attempt; lock the code after too many.
        _, _ = s.db.Exec(ctx,
            `UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1`, otpID)
        return LoginResult{}, ErrOtpInvalid
    }

    // The code is single-use.
    _, err = s.db.Exec(ctx,
        `UPDATE otp_codes SET verified_at = NOW() WHERE id = $1`, otpID)
    if err != nil {
        return LoginResult{}, fmt.Errorf("consume otp: %w", err)
    }

    return s.IssueSession(ctx, userID)
}

// IssueSession creates a full login session: one refresh token row
// (hashed) plus one access JWT.
func (s *Service) IssueSession(ctx context.Context, userID uuid.UUID) (LoginResult, error) {
    var user UserOut
    err := s.db.QueryRow(ctx,
        `SELECT id, name, email FROM users WHERE id = $1`, userID,
    ).Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return LoginResult{}, fmt.Errorf("load user: %w", err)
    }

    refreshRaw, refreshHash, err := NewRefreshToken()
    if err != nil {
        return LoginResult{}, fmt.Errorf("create refresh token: %w", err)
    }
    refreshExpiresAt := time.Now().Add(s.refreshTTL)
    _, err = s.db.Exec(ctx,
        `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
        uuid.New(), userID, refreshHash, refreshExpiresAt)
    if err != nil {
        return LoginResult{}, fmt.Errorf("store refresh token: %w", err)
    }

    access, accessExpiresAt, err := NewAccessToken(s.jwtSecret, userID, s.accessTTL)
    if err != nil {
        return LoginResult{}, fmt.Errorf("create access token: %w", err)
    }

    return LoginResult{
        User:             user,
        AccessToken:      access,
        TokenType:        "Bearer",
        AccessExpiresAt:  accessExpiresAt,
        RefreshRaw:       refreshRaw,
        RefreshExpiresAt: refreshExpiresAt,
    }, nil
}

// VerifyLoginOtp handles POST /api/v1/auth/login/verify.
func (h *Handler) VerifyLoginOtp(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in struct {
        Email string `json:"email"`
        Code  string `json:"code"`
    }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }
    if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Code) == "" {
        httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError, "email and code are required")
        return
    }

    res, err := h.svc.VerifyLoginOtp(r.Context(), in.Email, in.Code)
    if err != nil {
        var uerr *httpx.UserError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
            return
        }
        h.log.Error("otp verify failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    h.setRefreshCookie(w, res.RefreshRaw)
    httpx.Success(w, http.StatusOK, loginResponse{
        User:        res.User,
        AccessToken: res.AccessToken,
        TokenType:   res.TokenType,
        ExpiresAt:   res.AccessExpiresAt,
    })
}

// otpEmailHTML renders the branded OTP email. Table-based with inline
// styles — the only HTML email clients reliably render.
func otpEmailHTML(code string) string {
    return `<!DOCTYPE html>
<html>
<body style="margin:0;padding:0;background:#09090b;">
  <div style="display:none;max-height:0;overflow:hidden;">Your LinkPulse code: ` + code + `</div>
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#09090b;padding:40px 16px;">
    <tr><td align="center">
      <table role="presentation" width="480" cellpadding="0" cellspacing="0" style="max-width:480px;background:#18181b;border-radius:12px;">
        <tr><td style="padding:32px 32px 8px;">
          <p style="margin:0;font-size:20px;font-weight:bold;color:#fafafa;font-family:Arial,Helvetica,sans-serif;">Link<span style="color:#3b82f6;">Pulse</span></p>
        </td></tr>
        <tr><td style="padding:8px 32px;">
          <p style="margin:0;font-size:14px;color:#a1a1aa;font-family:Arial,Helvetica,sans-serif;">Your login verification code:</p>
        </td></tr>
        <tr><td style="padding:16px 32px;">
          <div style="background:#09090b;border-radius:8px;padding:20px;text-align:center;">
            <span style="font-size:32px;letter-spacing:10px;color:#3b82f6;font-weight:bold;font-family:'Courier New',monospace;">` + code + `</span>
          </div>
        </td></tr>
        <tr><td style="padding:8px 32px 32px;">
          <p style="margin:0;font-size:12px;color:#71717a;font-family:Arial,Helvetica,sans-serif;">This code expires in 10 minutes. If you didn't try to sign in, you can safely ignore this email.</p>
        </td></tr>
      </table>
      <p style="margin-top:24px;font-size:11px;color:#52525b;font-family:Arial,Helvetica,sans-serif;">LinkPulse — self-hosted URL shortener</p>
    </td></tr>
  </table>
</body>
</html>`
}
