package link

import (
	"time"

	"github.com/google/uuid"
)

// LinkOut is the API shape of a link. The password hash is never
// exposed — only the fact that a password exists.
type LinkOut struct {
    ID                uuid.UUID  `json:"id"`
    ShortCode         string     `json:"short_code"`
    ShortURL          string     `json:"short_url"`
    DestinationURL    string     `json:"destination_url"`
    Title             string     `json:"title"`
    Status            string     `json:"status"`
    PasswordProtected bool       `json:"password_protected"`
    ExpiresAt         *time.Time `json:"expires_at"`
    MaxClicks         *int64     `json:"max_clicks"`
    ClickCount        int64      `json:"click_count"`
    Tags              []string   `json:"tags"`
    UTMSource         string     `json:"utm_source"`
    UTMMedium         string     `json:"utm_medium"`
    UTMCampaign       string     `json:"utm_campaign"`
    UTMTerm           string     `json:"utm_term"`
    UTMContent        string     `json:"utm_content"`
    CreatedAt         time.Time  `json:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at"`
}

// row mirrors the links table columns for scanning.
type row struct {
    ID             uuid.UUID
    ShortCode      string
    DestinationURL string
    Title          string
    Status         string
    PasswordHash   *string
    ExpiresAt      *time.Time
    MaxClicks      *int64
    ClickCount     int64
    Tags           []string
    UTMSource      *string
    UTMMedium      *string
    UTMCampaign    *string
    UTMTerm        *string
    UTMContent     *string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// columns is the shared SELECT list for every link query.
const columns = `id, short_code, destination_url, title, status, password_hash,
    expires_at, max_clicks, click_count, tags,
    utm_source, utm_medium, utm_campaign, utm_term, utm_content,
    created_at, updated_at`

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface{ Scan(dest ...any) error }

// scanLink scans the shared column list, plus optional extra columns
// (e.g. the window-function total used by List).
func scanLink(sc scanner, extra ...any) (row, error) {
    var r row
    dest := []any{
        &r.ID, &r.ShortCode, &r.DestinationURL, &r.Title, &r.Status, &r.PasswordHash,
        &r.ExpiresAt, &r.MaxClicks, &r.ClickCount, &r.Tags,
        &r.UTMSource, &r.UTMMedium, &r.UTMCampaign, &r.UTMTerm, &r.UTMContent,
        &r.CreatedAt, &r.UpdatedAt,
    }
    dest = append(dest, extra...)
    err := sc.Scan(dest...)
    return r, err
}

// effectiveStatus computes the link status per PRD 9.13: user-set
// disabled wins, then expiry, then click limits. Password protection
// is reported separately as a flag.
func effectiveStatus(r row) string {
    switch {
    case r.Status == "disabled":
        return "disabled"
    case r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()):
        return "expired"
    case r.MaxClicks != nil && r.ClickCount >= *r.MaxClicks:
        return "max_clicks_reached"
    default:
        return "active"
    }
}
