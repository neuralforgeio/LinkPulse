package redirect

// service.go — short code lookup (PRD 9.5.1).

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound: no live link with this code.
var ErrNotFound = errors.New("link not found")

// ResolvedLink is everything the redirect needs from the links row.
type ResolvedLink struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    ShortCode      string
    DestinationURL string
    PasswordHash   *string
    ExpiresAt      *time.Time
    MaxClicks      *int64
    ClickCount     int64
    Status         string
    UTMSource      string
    UTMMedium      string
    UTMCampaign    string
}

// Service resolves short codes.
type Service struct {
    db *pgxpool.Pool
}

// NewService builds a redirect Service.
func NewService(db *pgxpool.Pool) *Service {
    return &Service{db: db}
}

// Find loads a live link by short code. Soft-deleted links are
// invisible (PRD 9.4.5).
func (s *Service) Find(ctx context.Context, code string) (ResolvedLink, error) {
    var l ResolvedLink
    var utmSource, utmMedium, utmCampaign *string

    err := s.db.QueryRow(ctx, `
        SELECT id, tenant_id, short_code, destination_url, password_hash,
               expires_at, max_clicks, click_count, status,
               utm_source, utm_medium, utm_campaign
        FROM links
        WHERE short_code = $1 AND deleted_at IS NULL`, code,
    ).Scan(
        &l.ID, &l.TenantID, &l.ShortCode, &l.DestinationURL, &l.PasswordHash,
        &l.ExpiresAt, &l.MaxClicks, &l.ClickCount, &l.Status,
        &utmSource, &utmMedium, &utmCampaign,
    )
    if errors.Is(err, pgx.ErrNoRows) {
        return l, ErrNotFound
    }
    if err != nil {
        return l, fmt.Errorf("load link: %w", err)
    }

    l.UTMSource = derefStr(utmSource)
    l.UTMMedium = derefStr(utmMedium)
    l.UTMCampaign = derefStr(utmCampaign)
    return l, nil
}

func derefStr(p *string) string {
    if p == nil {
        return ""
    }
    return *p
}
