package link

// query.go — list with search/filter/sort/pagination (PRD 9.4.2) and
// single-link detail (PRD 9.4.3).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"linkpulse/internal/httpx"
)

// ListFilter is the parsed query string of GET .../links.
type ListFilter struct {
    Page      int
    PageSize  int
    Search    string
    Status    string
    Tag       string
    SortBy    string
    SortOrder string
    StartDate time.Time
    EndDate   time.Time
}

// ListResult is the paginated list response.
type ListResult struct {
    Links    []LinkOut `json:"links"`
    Page     int       `json:"page"`
    PageSize int       `json:"page_size"`
    Total    int       `json:"total"`
}

// Whitelists: only these fixed fragments ever reach the SQL string.
// Everything the user types travels as a bound parameter.
var statusFilters = map[string]string{
    "active":             "status = 'active' AND (expires_at IS NULL OR expires_at > NOW()) AND (max_clicks IS NULL OR click_count < max_clicks)",
    "expired":            "expires_at IS NOT NULL AND expires_at <= NOW()",
    "max_clicks_reached": "max_clicks IS NOT NULL AND click_count >= max_clicks",
    "disabled":           "status = 'disabled'",
    "password_protected": "password_hash IS NOT NULL",
}

var sortColumns = map[string]string{
    "created_at": "created_at",
    "click_count": "click_count",
    "title":       "title",
}

// List returns the workspace's links, filtered and paginated.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, f ListFilter) (ListResult, error) {
    where := "tenant_id = $1 AND deleted_at IS NULL"
    args := []any{tenantID}
    n := 2

    if f.Search != "" {
        pattern := "%" + f.Search + "%"
        where += fmt.Sprintf(
            " AND (title ILIKE $%d OR short_code ILIKE $%d OR destination_url ILIKE $%d)", n, n, n)
        args = append(args, pattern)
        n++
    }
    if f.Tag != "" {
        where += fmt.Sprintf(" AND tags @> ARRAY[$%d]", n)
        args = append(args, f.Tag)
        n++
    }
    if !f.StartDate.IsZero() {
        where += fmt.Sprintf(" AND created_at >= $%d", n)
        args = append(args, f.StartDate)
        n++
    }
    if !f.EndDate.IsZero() {
        // end_date is inclusive: covers the whole day.
        where += fmt.Sprintf(" AND created_at < $%d", n)
        args = append(args, f.EndDate.Add(24*time.Hour))
        n++
    }
    if f.Status != "" {
        fragment, ok := statusFilters[f.Status]
        if !ok {
            return ListResult{}, &httpx.UserError{
                Status:  422,
                Code:    httpx.CodeValidationError,
                Message: "unknown status filter",
            }
        }
        where += " AND " + fragment
    }

    sortCol := sortColumns[f.SortBy]
    order := "DESC"
    if f.SortOrder == "asc" {
        order = "ASC"
    }

    offset := (f.Page - 1) * f.PageSize
    sql := fmt.Sprintf(`
        SELECT %s, COUNT(*) OVER() AS total
        FROM links
        WHERE %s
        ORDER BY %s %s
        LIMIT $%d OFFSET $%d`,
        columns, where, sortCol, order, n, n+1)
    args = append(args, f.PageSize, offset)

    rows, err := s.db.Query(ctx, sql, args...)
    if err != nil {
        return ListResult{}, fmt.Errorf("query links: %w", err)
    }
    defer rows.Close()

    result := ListResult{Links: []LinkOut{}, Page: f.Page, PageSize: f.PageSize}
    for rows.Next() {
        var total int
        r, err := scanLink(rows, &total)
        if err != nil {
            return ListResult{}, fmt.Errorf("scan link: %w", err)
        }
        result.Total = total
        result.Links = append(result.Links, s.toOut(r))
    }
    if err := rows.Err(); err != nil {
        return ListResult{}, fmt.Errorf("iterate links: %w", err)
    }
    return result, nil
}

// Get returns one link's detail (PRD 9.4.3). The full analytics summary
// joins this response in Milestone 5 — today click_count tells the story.
func (s *Service) Get(ctx context.Context, tenantID, linkID uuid.UUID) (LinkOut, error) {
    sql := fmt.Sprintf(
        `SELECT %s FROM links WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
        columns)
    r, err := scanLink(s.db.QueryRow(ctx, sql, linkID, tenantID))
    if errors.Is(err, pgx.ErrNoRows) {
        return LinkOut{}, ErrNotFound
    }
    if err != nil {
        return LinkOut{}, fmt.Errorf("load link: %w", err)
    }
    return s.toOut(r), nil
}
