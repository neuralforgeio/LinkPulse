package auth

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/httpx"
    "linkpulse/internal/shortid"
)

// resetTokenTTL is how long a reset link stays valid (PRD 9.1.7).
const resetTokenTTL = 30 * time.Minute

// ResetHandler serves the password reset endpoints. It deliberately
// does NOT reuse the session machinery — reset works without a session.
type ResetHandler struct {
    db             *pgxpool.Pool
    log            *slog.Logger
    resetLinkBase  string // frontend origin used to build reset links
}

// NewResetHandler builds the reset handler. resetLinkBase is the
// frontend origin the reset link points at.
func NewResetHandler(db *pgxpool.Pool, log *slog.Logger, resetLinkBase string) *ResetHandler {
    return &ResetHandler{db: db, log: log, resetLinkBase: resetLinkBase}
}

// ResetRequestInput is the body for POST /api/v1/auth/password/reset-request.
type ResetRequestInput struct {
    Email string `json:"email"`
}

// ResetConfirmInput is the body for POST /api/v1/auth/password/reset-confirm.
type ResetConfirmInput struct {
    Token    string `json:"token"`
    Password string `json:"password"`
}

// RequestReset handles POST /api/v1/auth/password/reset-request.
// The response is ALWAYS the same, whether or not the email exists —
// never leak account existence (PRD 9.1.7).
func (h *ResetHandler) RequestReset(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in ResetRequestInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    email := strings.ToLower(strings.TrimSpace(in.Email))
    if emailRegex.MatchString(email) {
        var userID uuid.UUID
        err := h.db.QueryRow(r.Context(),
            `SELECT id FROM users WHERE email = $1`, email,
        ).Scan(&userID)
        if err == nil {
            raw, hash, terr := newResetToken()
            if terr == nil {
                _, ierr := h.db.Exec(r.Context(), `
                    INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
                    VALUES ($1, $2, $3, $4)`,
                    uuid.New(), userID, hash, time.Now().Add(resetTokenTTL))
                if ierr == nil {
                    // Dev mode delivery: the link goes to the log (PRD 9.1.7).
                    h.log.Info("password reset link generated",
                        "link", h.resetLinkBase+"/reset-password?token="+raw)
                } else {
                    h.log.Error("reset token insert failed", "error", ierr)
                }
            } else {
                h.log.Error("reset token generation failed", "error", terr)
            }
        }
        // Lookup failures are swallowed on purpose: identical responses
        // regardless of the email's existence.
    }

    httpx.Success(w, http.StatusOK, map[string]string{
        "message": "If an account exists for that email, a reset link has been sent.",
    })
}

// ConfirmReset handles POST /api/v1/auth/password/reset-confirm.
func (h *ResetHandler) ConfirmReset(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in ResetConfirmInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    token := strings.TrimSpace(in.Token)
    if token == "" {
        httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError, "token is required")
        return
    }
    if len(in.Password) < 8 || !hasLetter(in.Password) || !hasDigit(in.Password) {
        httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError,
            "password must be at least 8 characters with letters and numbers")
        return
    }

    var (
        resetID   uuid.UUID
        userID    uuid.UUID
        expiresAt time.Time
        usedAt    *time.Time
    )
    err := h.db.QueryRow(r.Context(), `
        SELECT id, user_id, expires_at, used_at
        FROM password_reset_tokens
        WHERE token_hash = $1`, hashResetToken(token),
    ).Scan(&resetID, &userID, &expiresAt, &usedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidToken, "invalid or expired reset token")
        return
    }
    if err != nil {
        h.log.Error("reset token lookup failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }
    if usedAt != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidToken, "this reset token has already been used")
        return
    }
    if time.Now().After(expiresAt) {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeInvalidToken, "this reset token has expired")
        return
    }

    newHash, err := HashPassword(in.Password)
    if err != nil {
        h.log.Error("reset password hash failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    _, err = h.db.Exec(r.Context(),
        `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`,
        newHash, userID)
    if err != nil {
        h.log.Error("reset password update failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    _, err = h.db.Exec(r.Context(),
        `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, resetID)
    if err != nil {
        h.log.Error("reset token consume failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    // PRD 9.1.8: a successful reset revokes every session.
    _, err = h.db.Exec(r.Context(),
        `UPDATE refresh_tokens SET revoked_at = NOW()
         WHERE user_id = $1 AND revoked_at IS NULL`, userID)
    if err != nil {
        h.log.Error("reset session revocation failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    httpx.Success(w, http.StatusOK, map[string]string{
        "message": "Password updated. All sessions have been revoked — please sign in again.",
    })
}

// newResetToken returns (raw, hash): raw goes into the reset link,
// only the SHA-256 hash is stored. Like refresh tokens, the raw value
// is 256 random bits — a fast hash is sufficient.
func newResetToken() (raw, hash string, err error) {
    secret, err := shortid.New(32)
    if err != nil {
        return "", "", err
    }
    raw = "lp_reset_" + secret
    return raw, hashResetToken(raw), nil
}

func hashResetToken(raw string) string {
    sum := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(sum[:])
}

// unused import guard: context is used by future query helpers.
var _ = context.Background
