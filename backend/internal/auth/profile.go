package auth

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "strings"

    "github.com/google/uuid"

    "linkpulse/internal/httpx"
)

// ProfileUpdateInput is the request body for PATCH /api/v1/auth/me.
type ProfileUpdateInput struct {
    Name        string  `json:"name"`
    OldPassword *string `json:"old_password"`
    NewPassword *string `json:"new_password"`
}

// UpdateProfile renames the user and optionally rotates their password.
// currentToken is the caller's refresh token — its session survives the
// password change, all others are revoked.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, currentToken string, in ProfileUpdateInput) (UserOut, error) {
    name := strings.TrimSpace(in.Name)
    if len(name) < 2 || len(name) > 100 {
        return UserOut{}, &httpx.UserError{
            Status:  http.StatusUnprocessableEntity,
            Code:    httpx.CodeValidationError,
            Message: "name must be 2-100 characters",
        }
    }

    // Password change is optional — but both fields must come together.
    if in.OldPassword != nil || in.NewPassword != nil {
        if in.OldPassword == nil || in.NewPassword == nil ||
            *in.OldPassword == "" || *in.NewPassword == "" {
            return UserOut{}, &httpx.UserError{
                Status:  http.StatusUnprocessableEntity,
                Code:    httpx.CodeValidationError,
                Message: "changing the password requires both old_password and new_password",
            }
        }
        if len(*in.NewPassword) < 8 || !hasLetter(*in.NewPassword) || !hasDigit(*in.NewPassword) {
            return UserOut{}, &httpx.UserError{
                Status:  http.StatusUnprocessableEntity,
                Code:    httpx.CodeValidationError,
                Message: "new password must be at least 8 characters with letters and numbers",
            }
        }

        // The current password must be verified before any change.
        var hash string
        err := s.db.QueryRow(ctx,
            `SELECT password_hash FROM users WHERE id = $1`, userID,
        ).Scan(&hash)
        if err != nil {
            return UserOut{}, fmt.Errorf("load user: %w", err)
        }
        ok, err := VerifyPassword(*in.OldPassword, hash)
        if err != nil {
            return UserOut{}, fmt.Errorf("verify current password: %w", err)
        }
        if !ok {
            return UserOut{}, &httpx.UserError{
                Status:  http.StatusUnauthorized,
                Code:    httpx.CodeUnauthorized,
                Message: "current password is incorrect",
            }
        }

        newHash, err := HashPassword(*in.NewPassword)
        if err != nil {
            return UserOut{}, fmt.Errorf("hash new password: %w", err)
        }

        _, err = s.db.Exec(ctx, `
            UPDATE users SET name = $1, password_hash = $2, updated_at = NOW()
            WHERE id = $3`, name, newHash, userID)
        if err != nil {
            return UserOut{}, fmt.Errorf("update user: %w", err)
        }

        // Revoke every session except the one making this request.
        _, err = s.db.Exec(ctx, `
            UPDATE refresh_tokens SET revoked_at = NOW()
            WHERE user_id = $1 AND revoked_at IS NULL AND token_hash <> $2`,
            userID, HashRefreshToken(currentToken))
        if err != nil {
            return UserOut{}, fmt.Errorf("revoke other sessions: %w", err)
        }
    } else {
        _, err := s.db.Exec(ctx, `
            UPDATE users SET name = $1, updated_at = NOW()
            WHERE id = $2`, name, userID)
        if err != nil {
            return UserOut{}, fmt.Errorf("update user: %w", err)
        }
    }

    var out UserOut
    err := s.db.QueryRow(ctx,
        `SELECT id, name, email FROM users WHERE id = $1`, userID,
    ).Scan(&out.ID, &out.Name, &out.Email)
    if err != nil {
        return UserOut{}, fmt.Errorf("reload user: %w", err)
    }
    return out, nil
}

// UpdateProfile handles PATCH /api/v1/auth/me.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
    userID, ok := httpx.UserIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 4096)
    var in ProfileUpdateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    // The current session's refresh token — kept alive across a
    // password change.
    currentToken := ""
    if cookie, err := r.Cookie(refreshCookieName); err == nil {
        currentToken = cookie.Value
    }

    res, err := h.svc.UpdateProfile(r.Context(), userID, currentToken, in)
    if err != nil {
        var uerr *httpx.UserError
        if errors.As(err, &uerr) {
            httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
            return
        }
        h.log.Error("profile update failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}
