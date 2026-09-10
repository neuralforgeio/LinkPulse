// Package analytics aggregates click data into dashboard summaries
// (PRD 9.6). All aggregation runs in PostgreSQL; time buckets are UTC
// days for deterministic charts.
package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Granularity options for the time series.
const (
    GranularityDay   = "day"
    GranularityWeek  = "week"
    GranularityMonth = "month"
)

// Params is the validated query for the overview endpoint.
type Params struct {
    Start       time.Time
    End         time.Time
    Granularity string
}

// TimePoint is one bucket of the clicks-over-time series.
type TimePoint struct {
    Date   string `json:"date"`
    Clicks int64  `json:"clicks"`
}

// TopLink is one row of the top-links ranking.
type TopLink struct {
    LinkID    uuid.UUID `json:"link_id"`
    ShortCode string    `json:"short_code"`
    Title     string    `json:"title"`
    Clicks    int64     `json:"clicks"`
}

// NameCount is one row of a "top X" ranking.
type NameCount struct {
    Name   string `json:"name"`
    Clicks int64  `json:"clicks"`
}

// OverviewResult mirrors PRD 9.6.1.
type OverviewResult struct {
    TotalClicks            int64       `json:"total_clicks"`
    UniqueClicksEstimate   int64       `json:"unique_clicks_estimate"`
    ActiveLinks            int64       `json:"active_links"`
    ExpiredLinks           int64       `json:"expired_links"`
    PasswordProtectedLinks int64       `json:"password_protected_links"`
    DisabledLinks          int64       `json:"disabled_links"`
    ClicksOverTime         []TimePoint `json:"clicks_over_time"`
    TopLinks               []TopLink   `json:"top_links"`
    TopReferrers           []NameCount `json:"top_referrers"`
    TopBrowsers            []NameCount `json:"top_browsers"`
    TopDevices             []NameCount `json:"top_devices"`
    TopOS                  []NameCount `json:"top_os"`
    TopCampaigns           []NameCount `json:"top_campaigns"`
}

// Service holds dependencies for analytics flows.
type Service struct {
    db  *pgxpool.Pool
    log *slog.Logger
}

// NewService builds an analytics Service.
func NewService(db *pgxpool.Pool, log *slog.Logger) *Service {
    return &Service{db: db, log: log}
}

// Overview aggregates the workspace's analytics (PRD 9.6.1).
//
// Unique-visitor estimate: COUNT(DISTINCT (ip_hash, day)) — one visitor
// per salted IP hash per day. The chosen formula per PRD 9.6.4.
func (s *Service) Overview(ctx context.Context, tenantID uuid.UUID, p Params) (OverviewResult, error) {
    res := OverviewResult{
        ClicksOverTime: []TimePoint{},
        TopLinks:       []TopLink{},
        TopReferrers:   []NameCount{},
        TopBrowsers:    []NameCount{},
        TopDevices:     []NameCount{},
        TopOS:          []NameCount{},
        TopCampaigns:   []NameCount{},
    }

    // Totals + unique visitors.
    err := s.db.QueryRow(ctx, `
        SELECT COUNT(*),
               COUNT(DISTINCT (ip_hash, (clicked_at AT TIME ZONE 'UTC')::date))
        FROM click_events
        WHERE tenant_id = $1 AND clicked_at >= $2 AND clicked_at < $3`,
        tenantID, p.Start, p.End,
    ).Scan(&res.TotalClicks, &res.UniqueClicksEstimate)
    if err != nil {
        return res, fmt.Errorf("totals: %w", err)
    }

    // Clicks over time.
    rows, err := s.db.Query(ctx, fmt.Sprintf(`
        SELECT %s AS bucket, COUNT(*) AS clicks
        FROM click_events
        WHERE tenant_id = $1 AND clicked_at >= $2 AND clicked_at < $3
        GROUP BY 1
        ORDER BY 1`, bucketExpr(p.Granularity)),
        tenantID, p.Start, p.End)
    if err != nil {
        return res, fmt.Errorf("time series: %w", err)
    }
    defer rows.Close()

    counts := make(map[string]int64)
    series := []TimePoint{}
    for rows.Next() {
        var bucket time.Time
        var clicks int64
        if err := rows.Scan(&bucket, &clicks); err != nil {
            return res, fmt.Errorf("scan time series: %w", err)
        }
        key := bucket.UTC().Format("2006-01-02")
        counts[key] = clicks
        series = append(series, TimePoint{Date: key, Clicks: clicks})
    }
    if err := rows.Err(); err != nil {
        return res, fmt.Errorf("iterate time series: %w", err)
    }

    // Daily granularity gets gap-filled so zero-click days still appear.
    if p.Granularity == GranularityDay {
        res.ClicksOverTime = fillDaily(p.Start, p.End, counts)
    } else {
        res.ClicksOverTime = series
    }

    // Current link states (not range-bound — these are states, not events).
    err = s.db.QueryRow(ctx, `
        SELECT
            COUNT(*) FILTER (WHERE status = 'active'
                AND (expires_at IS NULL OR expires_at > NOW())
                AND (max_clicks IS NULL OR click_count < max_clicks)
                AND deleted_at IS NULL),
            COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at <= NOW()
                AND deleted_at IS NULL),
            COUNT(*) FILTER (WHERE password_hash IS NOT NULL AND deleted_at IS NULL),
            COUNT(*) FILTER (WHERE status = 'disabled' AND deleted_at IS NULL)
        FROM links
        WHERE tenant_id = $1`, tenantID,
    ).Scan(&res.ActiveLinks, &res.ExpiredLinks, &res.PasswordProtectedLinks, &res.DisabledLinks)
    if err != nil {
        return res, fmt.Errorf("link counts: %w", err)
    }

    // Top links — deleted links keep their analytics (PRD 9.4.5).
    rows, err = s.db.Query(ctx, `
        SELECT l.id, l.short_code, COALESCE(NULLIF(l.title, ''), l.short_code), COUNT(c.id)
        FROM click_events c
        JOIN links l ON l.id = c.link_id
        WHERE c.tenant_id = $1 AND c.clicked_at >= $2 AND c.clicked_at < $3
        GROUP BY l.id, l.short_code, l.title
        ORDER BY COUNT(c.id) DESC
        LIMIT 10`,
        tenantID, p.Start, p.End)
    if err != nil {
        return res, fmt.Errorf("top links: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var tl TopLink
        if err := rows.Scan(&tl.LinkID, &tl.ShortCode, &tl.Title, &tl.Clicks); err != nil {
            return res, fmt.Errorf("scan top link: %w", err)
        }
        res.TopLinks = append(res.TopLinks, tl)
    }
    if err := rows.Err(); err != nil {
        return res, fmt.Errorf("iterate top links: %w", err)
    }

    // Top rankings. `column` is always a literal from this file — never
    // user input — so the Sprintf is injection-safe.
    res.TopReferrers, err = s.topNames(ctx, tenantID, p, "referrer", "(direct)", false)
    if err != nil {
        return res, fmt.Errorf("top referrers: %w", err)
    }
    res.TopBrowsers, err = s.topNames(ctx, tenantID, p, "browser", "(unknown)", false)
    if err != nil {
        return res, fmt.Errorf("top browsers: %w", err)
    }
    res.TopDevices, err = s.topNames(ctx, tenantID, p, "device_type", "(unknown)", false)
    if err != nil {
        return res, fmt.Errorf("top devices: %w", err)
    }
    res.TopOS, err = s.topNames(ctx, tenantID, p, "os", "(unknown)", false)
    if err != nil {
        return res, fmt.Errorf("top os: %w", err)
    }
    res.TopCampaigns, err = s.topNames(ctx, tenantID, p, "campaign", "", true)
    if err != nil {
        return res, fmt.Errorf("top campaigns: %w", err)
    }

    return res, nil
}

// topNames ranks a click_events column by click count. Empty values map
// to emptyLabel; skipEmpty drops rows without a value (used for campaigns).
func (s *Service) topNames(ctx context.Context, tenantID uuid.UUID, p Params, column, emptyLabel string, skipEmpty bool) ([]NameCount, error) {
    query := fmt.Sprintf(`
        SELECT COALESCE(NULLIF(%s, ''), $4), COUNT(*)
        FROM click_events
        WHERE tenant_id = $1 AND clicked_at >= $2 AND clicked_at < $3`, column)
    if skipEmpty {
        query += fmt.Sprintf(" AND %s IS NOT NULL AND %s <> ''", column, column)
    }
    query += " GROUP BY 1 ORDER BY 2 DESC LIMIT 10"

    rows, err := s.db.Query(ctx, query, tenantID, p.Start, p.End, emptyLabel)
    if err != nil {
        return nil, fmt.Errorf("query %s: %w", column, err)
    }
    defer rows.Close()

    out := []NameCount{}
    for rows.Next() {
        var nc NameCount
        if err := rows.Scan(&nc.Name, &nc.Clicks); err != nil {
            return nil, fmt.Errorf("scan %s: %w", column, err)
        }
        out = append(out, nc)
    }
    return out, rows.Err()
}

// bucketExpr returns the UTC time bucket for the granularity.
func bucketExpr(granularity string) string {
    switch granularity {
    case GranularityWeek:
        return `date_trunc('week', clicked_at AT TIME ZONE 'UTC')`
    case GranularityMonth:
        return `date_trunc('month', clicked_at AT TIME ZONE 'UTC')`
    default:
        return `(clicked_at AT TIME ZONE 'UTC')::date`
    }
}

// fillDaily produces one point per UTC day, zeroes included, so the
// chart never hides empty days.
func fillDaily(start, end time.Time, counts map[string]int64) []TimePoint {
    out := []TimePoint{}
    for d := start.UTC().Truncate(24 * time.Hour); d.Before(end); d = d.AddDate(0, 0, 1) {
        key := d.Format("2006-01-02")
        out = append(out, TimePoint{Date: key, Clicks: counts[key]})
    }
    return out
}
