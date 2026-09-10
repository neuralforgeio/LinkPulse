package link

// mutate.go — partial updates and soft deletes (PRD 9.4.4, 9.4.5).

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    "linkpulse/internal/auth"
    "linkpulse/internal/httpx"
)

// UpdateInput is the request body for PATCH .../links/{linkId}. Nil
// fields stay unchanged. short_code is immutable (PRD 9.4.4). Password
// semantics: absent = unchanged, "" = remove, value = set new.
type UpdateInput struct {
    DestinationURL *string    `json:"destination_url"`
    Title          *string    `json:"title"`
    Tags           *[]string  `json:"tags"`
    ExpiresAt      *time.Time `json:"expires_at"`
    MaxClicks      *int64     `json:"max_clicks"`
    Password       *string    `json:"password"`
    Status         *string    `json:"status"`
}

// Update modifies link fields. Setting expires_at to a future time
// re-activates an expired link automatically (PRD 9.4.4) — status is
// computed, never stored as expired.
func (s *Service) Update(ctx context.Context, tenantID, linkID uuid.UUID, in UpdateInput) (LinkOut, error) {
    var urlArg, titleArg, statusArg *string
    var tagsArg *[]string
    var expiresArg *time.Time
    var clicksArg *int64

    if in.DestinationURL != nil {
        destination := strings.TrimSpace(*in.DestinationURL)
        if uerr := ValidateDestinationURL(destination); uerr != nil {
            return LinkOut{}, uerr
        }
        urlArg = &destination
    }
    if in.Title != nil {
        title := strings.TrimSpace(*in.Title)
        if len(title) > maxTitleLen {
            return LinkOut{}, &httpx.UserError{
                Status:  422,
                Code:    httpx.CodeValidationError,
                Message: "title is too long (max 200 characters)",
            }
        }
        titleArg = &title
    }
    if in.Tags != nil {
        tags := NormalizeTags(*in.Tags)
        tagsArg = &tags
    }
    if in.ExpiresAt != nil {
        if in.ExpiresAt.Before(time.Now()) {
            return LinkOut{}, &httpx.UserError{
                Status:  422,
                Code:    httpx.CodeValidationError,
                Message: "expires_at must be in the future",
            }
        }
        expiresArg = in.ExpiresAt
    }
    if in.MaxClicks != nil {
        if *in.MaxClicks <= 0 {
            return LinkOut{}, &httpx.UserError{
                Status:  422,
                Code:    httpx.CodeValidationError,
                Message: "max_clicks must be greater than 0",
            }
        }
        clicksArg = in.MaxClicks
    }
    if in.Status != nil {
        st := strings.TrimSpace(*in.Status)
        if st != "active" && st != "disabled" {
            return LinkOut{}, &httpx.UserError{
                Status:  422,
                Code:    httpx.CodeValidationError,
                Message: "status must be active or disabled",
            }
        }
        statusArg = &st
    }
    if in.Password != nil && *in.Password != "" && len(*in.Password) < minLinkPassword {
        return LinkOut{}, &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "link password must be at least 4 characters",
        }
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return LinkOut{}, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // Main update — every field except the password.
    sql := fmt.Sprintf(`
        UPDATE links SET
            destination_url = COALESCE($1, destination_url),
            title = COALESCE($2, title),
            tags = COALESCE($3, tags),
            expires_at = COALESCE($4, expires_at),
            max_clicks = COALESCE($5, max_clicks),
            status = COALESCE($6, status),
            updated_at = NOW()
        WHERE id = $7 AND tenant_id = $8 AND deleted_at IS NULL
        RETURNING %s`, columns)
    r, err := scanLink(tx.QueryRow(ctx, sql,
        urlArg, titleArg, tagsArg, expiresArg, clicksArg, statusArg, linkID, tenantID))
    if errors.Is(err, pgx.ErrNoRows) {
        return LinkOut{}, ErrNotFound
    }
    if err != nil {
        return LinkOut{}, fmt.Errorf("update link: %w", err)
    }

    // Password changes need their own statement: COALESCE cannot tell
    // "remove" (empty string) apart from "unchanged" (null).
    if in.Password != nil {
        if *in.Password == "" {
            _, err = tx.Exec(ctx,
                `UPDATE links SET password_hash = NULL, updated_at = NOW() WHERE id = $1`, linkID)
        } else {
            h, herr := auth.HashPassword(*in.Password)
            if herr != nil {
                return LinkOut{}, fmt.Errorf("hash password: %w", herr)
            }
            _, err = tx.Exec(ctx,
                `UPDATE links SET password_hash = $1, updated_at = NOW() WHERE id = $2`, h, linkID)
        }
        if err != nil {
            return LinkOut{}, fmt.Errorf("update password: %w", err)
        }
        r, err = scanLink(tx.QueryRow(ctx,
            fmt.Sprintf(`SELECT %s FROM links WHERE id = $1`, columns), linkID))
        if err != nil {
            return LinkOut{}, fmt.Errorf("reload link: %w", err)
        }
    }

    if err := tx.Commit(ctx); err != nil {
        return LinkOut{}, fmt.Errorf("commit: %w", err)
    }
    return s.toOut(r), nil
}

// Delete soft-deletes a link (PRD 9.4.5): deleted_at is set, analytics
// history is preserved, and the link stops resolving.
func (s *Service) Delete(ctx context.Context, tenantID, linkID uuid.UUID) error {
    tag, err := s.db.Exec(ctx,
        `UPDATE links SET deleted_at = NOW(), updated_at = NOW()
         WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
        linkID, tenantID)
    if err != nil {
        return fmt.Errorf("delete link: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return ErrNotFound
    }
    return nil
}
