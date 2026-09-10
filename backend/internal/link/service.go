// Package link implements short link management: creation, listing,
// updates, and soft deletes (PRD 9.4).
package link

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"linkpulse/internal/auth"
	"linkpulse/internal/httpx"
	"linkpulse/internal/shortid"
)

// uniqueViolation is PostgreSQL's error code for a UNIQUE constraint hit.
const uniqueViolation = "23505"

var (
    // ErrCodeTaken: the custom alias is already in use.
    ErrCodeTaken = &httpx.UserError{
        Status:  http.StatusConflict,
        Code:    httpx.CodeConflict,
        Message: "short code is already taken",
    }
    // ErrNotFound: no such link in this workspace.
    ErrNotFound = &httpx.UserError{
        Status:  http.StatusNotFound,
        Code:    httpx.CodeNotFound,
        Message: "link not found",
    }
)

// Service holds dependencies for link flows.
type Service struct {
    db      *pgxpool.Pool
    log     *slog.Logger
    baseURL string
}

// NewService builds a link Service. baseURL builds short_url values
// (PRD 9.4.1 response shape).
func NewService(db *pgxpool.Pool, log *slog.Logger, baseURL string) *Service {
    return &Service{db: db, log: log, baseURL: baseURL}
}

// CreateInput is the request body for POST /api/v1/tenants/{tenantId}/links.
type CreateInput struct {
    DestinationURL string     `json:"destination_url"`
    Title          string     `json:"title"`
    CustomCode     string     `json:"custom_code"`
    ExpiresAt      *time.Time `json:"expires_at"`
    MaxClicks      *int64     `json:"max_clicks"`
    Password       *string    `json:"password"`
    Tags           []string   `json:"tags"`
    UTM            *UTMInput  `json:"utm"`
}

// Create makes a short link (PRD 9.4.1).
func (s *Service) Create(ctx context.Context, tenantID, userID uuid.UUID, in CreateInput) (LinkOut, error) {
    destination := strings.TrimSpace(in.DestinationURL)
    title := strings.TrimSpace(in.Title)
    customCode := strings.TrimSpace(in.CustomCode)

    if uerr := ValidateDestinationURL(destination); uerr != nil {
        return LinkOut{}, uerr
    }
    if len(title) > maxTitleLen {
        return LinkOut{}, &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "title is too long (max 200 characters)",
        }
    }
    if uerr := ValidateCustomCode(customCode); uerr != nil {
        return LinkOut{}, uerr
    }
    if in.ExpiresAt != nil && in.ExpiresAt.Before(time.Now()) {
        return LinkOut{}, &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "expires_at must be in the future",
        }
    }
    if in.MaxClicks != nil && *in.MaxClicks <= 0 {
        return LinkOut{}, &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "max_clicks must be greater than 0",
        }
    }

    var passwordHash *string
    if in.Password != nil && *in.Password != "" {
        if len(*in.Password) < minLinkPassword {
            return LinkOut{}, &httpx.UserError{
                Status:  422,
                Code:    httpx.CodeValidationError,
                Message: "link password must be at least 4 characters",
            }
        }
        h, err := auth.HashPassword(*in.Password)
        if err != nil {
            return LinkOut{}, fmt.Errorf("hash password: %w", err)
        }
        passwordHash = &h
    }
    tags := NormalizeTags(in.Tags)

    // Merge UTM into the destination (PRD 9.12: explicit values win).
    utm := UTMInput{}
    if in.UTM != nil {
        utm = *in.UTM
    }
    destination = ApplyUTM(destination, utm)

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return LinkOut{}, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // Resolve the short code: custom (conflict-checked) or generated.
    code := customCode
    if code == "" {
        code, err = uniqueCode(ctx, tx)
        if err != nil {
            return LinkOut{}, fmt.Errorf("generate code: %w", err)
        }
    } else {
        var taken bool
        err = tx.QueryRow(ctx,
            `SELECT EXISTS (SELECT 1 FROM links WHERE short_code = $1)`, code,
        ).Scan(&taken)
        if err != nil {
            return LinkOut{}, fmt.Errorf("check code: %w", err)
        }
        if taken {
            return LinkOut{}, ErrCodeTaken
        }
    }

    linkID := uuid.New()
    _, err = tx.Exec(ctx, `
        INSERT INTO links (id, tenant_id, created_by, short_code, destination_url,
            title, password_hash, expires_at, max_clicks, tags,
            utm_source, utm_medium, utm_campaign, utm_term, utm_content)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
        linkID, tenantID, userID, code, destination,
        title, passwordHash, in.ExpiresAt, in.MaxClicks, tags,
        utmOrNil(utm.Source), utmOrNil(utm.Medium), utmOrNil(utm.Campaign),
        utmOrNil(utm.Term), utmOrNil(utm.Content),
    )
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
            // Race backstop: two concurrent creates with the same code.
            return LinkOut{}, ErrCodeTaken
        }
        return LinkOut{}, fmt.Errorf("insert link: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return LinkOut{}, fmt.Errorf("commit: %w", err)
    }

    r := row{
        ID: linkID, ShortCode: code, DestinationURL: destination, Title: title,
        Status: "active", PasswordHash: passwordHash, ExpiresAt: in.ExpiresAt,
        MaxClicks: in.MaxClicks, ClickCount: 0, Tags: tags,
        UTMSource:  utmOrNil(utm.Source),
        UTMMedium:  utmOrNil(utm.Medium),
        UTMCampaign: utmOrNil(utm.Campaign),
        UTMTerm:    utmOrNil(utm.Term),
        UTMContent: utmOrNil(utm.Content),
        CreatedAt:  time.Now(), UpdatedAt: time.Now(),
    }
    return s.toOut(r), nil
}

// toOut converts a scanned row into the API shape.
func (s *Service) toOut(r row) LinkOut {
    return LinkOut{
        ID:                r.ID,
        ShortCode:         r.ShortCode,
        ShortURL:          s.baseURL + "/" + r.ShortCode,
        DestinationURL:    r.DestinationURL,
        Title:             r.Title,
        Status:            effectiveStatus(r),
        PasswordProtected: r.PasswordHash != nil,
        ExpiresAt:         r.ExpiresAt,
        MaxClicks:         r.MaxClicks,
        ClickCount:        r.ClickCount,
        Tags:              r.Tags,
        UTMSource:         derefStr(r.UTMSource),
        UTMMedium:         derefStr(r.UTMMedium),
        UTMCampaign:       derefStr(r.UTMCampaign),
        UTMTerm:           derefStr(r.UTMTerm),
        UTMContent:        derefStr(r.UTMContent),
        CreatedAt:         r.CreatedAt,
        UpdatedAt:         r.UpdatedAt,
    }
}

// uniqueCode draws random base62 codes until one is unused.
func uniqueCode(ctx context.Context, tx pgx.Tx) (string, error) {
    for attempt := 0; attempt < 5; attempt++ {
        code, err := shortid.New(7)
        if err != nil {
            return "", err
        }
        var taken bool
        err = tx.QueryRow(ctx,
            `SELECT EXISTS (SELECT 1 FROM links WHERE short_code = $1)`, code,
        ).Scan(&taken)
        if err != nil {
            return "", fmt.Errorf("check code: %w", err)
        }
        if !taken {
            return code, nil
        }
    }
    return "", errors.New("could not generate a unique short code")
}

func utmOrNil(v string) *string {
    if v == "" {
        return nil
    }
    s := v
    return &s
}

func derefStr(p *string) string {
    if p == nil {
        return ""
    }
    return *p
}
