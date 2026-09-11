// link.go — per-link analytics and recent clicks (PRD 9.6.2, 9.6.3).
package analytics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"linkpulse/internal/httpx"
)

// LinkAnalyticsResult mirrors PRD 9.6.2.
type LinkAnalyticsResult struct {
	LinkID               uuid.UUID   `json:"link_id"`
	ShortCode            string      `json:"short_code"`
	TotalClicks          int64       `json:"total_clicks"`
	UniqueClicksEstimate int64       `json:"unique_clicks_estimate"`
	ClicksOverTime       []TimePoint `json:"clicks_over_time"`
	TopReferrers         []NameCount `json:"top_referrers"`
	TopBrowsers          []NameCount `json:"top_browsers"`
	TopDevices           []NameCount `json:"top_devices"`
	TopOS                []NameCount `json:"top_os"`
	TopSources           []NameCount `json:"top_sources"`
	TopCampaigns         []NameCount `json:"top_campaigns"`
}

// RecentClick is one row of the recent clicks feed (PRD 9.6.3). Raw IPs
// are never included — only their salted hash.
type RecentClick struct {
	ID         uuid.UUID `json:"id"`
	ClickedAt  time.Time `json:"clicked_at"`
	IPHash     string    `json:"ip_hash"`
	Browser    string    `json:"browser"`
	OS         string    `json:"os"`
	DeviceType string    `json:"device_type"`
	Referrer   string    `json:"referrer"`
	Country    *string   `json:"country_code"`
}

// RecentClicksResult is a paginated page of recent clicks.
type RecentClicksResult struct {
	Clicks   []RecentClick `json:"clicks"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int64         `json:"total"`
}

// LinkNotFound mirrors the link package's sentinel without importing it
// (analytics must not depend on link — link depends on nothing of ours).
var linkNotFound = &httpx.UserError{
	Status:  http.StatusNotFound,
	Code:    httpx.CodeNotFound,
	Message: "link not found",
}

// LinkAnalytics aggregates one link's clicks (PRD 9.6.2). Tenant scoping
// is applied inside every query: link_id AND tenant_id must both match.
func (s *Service) LinkAnalytics(ctx context.Context, tenantID, linkID uuid.UUID, p Params) (LinkAnalyticsResult, error) {
	res := LinkAnalyticsResult{
		LinkID:         linkID,
		ClicksOverTime: []TimePoint{},
		TopReferrers:   []NameCount{},
		TopBrowsers:    []NameCount{},
		TopDevices:     []NameCount{},
		TopOS:          []NameCount{},
		TopSources:     []NameCount{},
		TopCampaigns:   []NameCount{},
	}

	// Existence + ownership check first — 404 must not leak analytics.
	var shortCode string
	err := s.db.QueryRow(ctx, `
		SELECT short_code FROM links
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		linkID, tenantID,
	).Scan(&shortCode)
	if err != nil {
		return res, linkNotFound
	}
	res.ShortCode = shortCode

	// Totals + unique estimate (same formula as Overview, PRD 9.6.4).
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(DISTINCT (ip_hash, (clicked_at AT TIME ZONE 'UTC')::date))
		FROM click_events
		WHERE link_id = $1 AND tenant_id = $2
		  AND clicked_at >= $3 AND clicked_at < $4`,
		linkID, tenantID, p.Start, p.End,
	).Scan(&res.TotalClicks, &res.UniqueClicksEstimate)
	if err != nil {
		return res, fmt.Errorf("link totals: %w", err)
	}

	// Time series with daily gap fill.
	rows, err := s.db.Query(ctx, fmt.Sprintf(`
		SELECT %s AS bucket, COUNT(*) AS clicks
		FROM click_events
		WHERE link_id = $1 AND tenant_id = $2
		  AND clicked_at >= $3 AND clicked_at < $4
		GROUP BY 1 ORDER BY 1`, bucketExpr(p.Granularity)),
		linkID, tenantID, p.Start, p.End)
	if err != nil {
		return res, fmt.Errorf("link time series: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	series := []TimePoint{}
	for rows.Next() {
		var bucket time.Time
		var clicks int64
		if err := rows.Scan(&bucket, &clicks); err != nil {
			return res, fmt.Errorf("scan link time series: %w", err)
		}
		key := bucket.UTC().Format("2006-01-02")
		counts[key] = clicks
		series = append(series, TimePoint{Date: key, Clicks: clicks})
	}
	if err := rows.Err(); err != nil {
		return res, fmt.Errorf("iterate link time series: %w", err)
	}
	if p.Granularity == GranularityDay {
		res.ClicksOverTime = fillDaily(p.Start, p.End, counts)
	} else {
		res.ClicksOverTime = series
	}

	// Top rankings scoped to this link.
	scoped := func(column, emptyLabel string, skipEmpty bool) ([]NameCount, error) {
		return s.topNamesForLink(ctx, linkID, tenantID, p, column, emptyLabel, skipEmpty)
	}
	if res.TopReferrers, err = scoped("referrer", "(direct)", false); err != nil {
		return res, fmt.Errorf("link top referrers: %w", err)
	}
	if res.TopBrowsers, err = scoped("browser", "(unknown)", false); err != nil {
		return res, fmt.Errorf("link top browsers: %w", err)
	}
	if res.TopDevices, err = scoped("device_type", "(unknown)", false); err != nil {
		return res, fmt.Errorf("link top devices: %w", err)
	}
	if res.TopOS, err = scoped("os", "(unknown)", false); err != nil {
		return res, fmt.Errorf("link top os: %w", err)
	}
	if res.TopSources, err = scoped("source", "(direct)", true); err != nil {
		return res, fmt.Errorf("link top sources: %w", err)
	}
	if res.TopCampaigns, err = scoped("campaign", "", true); err != nil {
		return res, fmt.Errorf("link top campaigns: %w", err)
	}

	return res, nil
}

// topNamesForLink is topNames constrained to a single link. Positional
// params: $1 link, $2 tenant, $3 start, $4 end, $5 empty label.
func (s *Service) topNamesForLink(ctx context.Context, linkID, tenantID uuid.UUID, p Params, column, emptyLabel string, skipEmpty bool) ([]NameCount, error) {
	query := fmt.Sprintf(`
		SELECT COALESCE(NULLIF(%s, ''), $5), COUNT(*)
		FROM click_events
		WHERE link_id = $1 AND tenant_id = $2
		  AND clicked_at >= $3 AND clicked_at < $4`, column)
	if skipEmpty {
		query += fmt.Sprintf(" AND %s IS NOT NULL AND %s <> ''", column, column)
	}
	query += " GROUP BY 1 ORDER BY 2 DESC LIMIT 10"
	return s.runNameCount(ctx, query, linkID, tenantID, p.Start, p.End, emptyLabel)
}

// runNameCount executes a top-X query with positional params shared by
// the link-scoped variants ($1 link, $2 tenant, $3 start, $4 end/label).
func (s *Service) runNameCount(ctx context.Context, query string, args ...any) ([]NameCount, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NameCount{}
	for rows.Next() {
		var nc NameCount
		if err := rows.Scan(&nc.Name, &nc.Clicks); err != nil {
			return nil, err
		}
		out = append(out, nc)
	}
	return out, rows.Err()
}

// RecentClicks returns the newest click events for a link (PRD 9.6.3),
// newest first, paginated. Raw IPs never leave the database.
func (s *Service) RecentClicks(ctx context.Context, tenantID, linkID uuid.UUID, page, pageSize int) (RecentClicksResult, error) {
	res := RecentClicksResult{Page: page, PageSize: pageSize}

	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM click_events
		WHERE link_id = $1 AND tenant_id = $2`,
		linkID, tenantID,
	).Scan(&res.Total)
	if err != nil {
		return res, fmt.Errorf("click count: %w", err)
	}

	rows, err := s.db.Query(ctx, `
		SELECT id, clicked_at, ip_hash,
		       COALESCE(browser, ''), COALESCE(os, ''),
		       COALESCE(device_type, ''), COALESCE(referrer, ''),
		       country_code
		FROM click_events
		WHERE link_id = $1 AND tenant_id = $2
		ORDER BY clicked_at DESC
		LIMIT $3 OFFSET $4`,
		linkID, tenantID, pageSize, (page-1)*pageSize)
	if err != nil {
		return res, fmt.Errorf("recent clicks query: %w", err)
	}
	defer rows.Close()

	res.Clicks = []RecentClick{}
	for rows.Next() {
		var c RecentClick
		if err := rows.Scan(&c.ID, &c.ClickedAt, &c.IPHash, &c.Browser, &c.OS,
			&c.DeviceType, &c.Referrer, &c.Country); err != nil {
			return res, fmt.Errorf("scan recent click: %w", err)
		}
		res.Clicks = append(res.Clicks, c)
	}
	return res, rows.Err()
}
