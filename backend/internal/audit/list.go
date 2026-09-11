package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"linkpulse/internal/httpx"
	"linkpulse/internal/tenant"
)

// LogEntry is one row of the audit history. The IP hash and user agent
// stay internal — need-to-know only.
type LogEntry struct {
	ID           uuid.UUID       `json:"id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	UserID       *uuid.UUID      `json:"user_id"`
	UserName     string          `json:"user_name"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ListResult is the paginated history response.
type ListResult struct {
	Entries  []LogEntry `json:"entries"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

// ListFilter is the parsed query for GET .../audit-logs.
type ListFilter struct {
	Page         int
	PageSize     int
	Action       string
	ResourceType string
	UserID       *uuid.UUID
	StartDate    time.Time
	EndDate      time.Time
}

// List returns the workspace's audit history.
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, f ListFilter) (ListResult, error) {
	where := "a.tenant_id = $1"
	args := []any{tenantID}
	n := 2

	if f.Action != "" {
		where += fmt.Sprintf(" AND a.action = $%d", n)
		args = append(args, f.Action)
		n++
	}
	if f.ResourceType != "" {
		where += fmt.Sprintf(" AND a.resource_type = $%d", n)
		args = append(args, f.ResourceType)
		n++
	}
	if f.UserID != nil {
		where += fmt.Sprintf(" AND a.user_id = $%d", n)
		args = append(args, *f.UserID)
		n++
	}
	if !f.StartDate.IsZero() {
		where += fmt.Sprintf(" AND a.created_at >= $%d", n)
		args = append(args, f.StartDate)
		n++
	}
	if !f.EndDate.IsZero() {
		where += fmt.Sprintf(" AND a.created_at < $%d", n)
		args = append(args, f.EndDate.Add(24*time.Hour)) // inclusive day
		n++
	}

	offset := (f.Page - 1) * f.PageSize
	sql := fmt.Sprintf(`
        SELECT a.id, a.action, a.resource_type, a.resource_id, a.user_id,
               COALESCE(u.name, ''), a.metadata, a.created_at,
               COUNT(*) OVER() AS total
        FROM audit_logs a
        LEFT JOIN users u ON u.id = a.user_id
        WHERE %s
        ORDER BY a.created_at DESC
        LIMIT $%d OFFSET $%d`, where, n, n+1)
	args = append(args, f.PageSize, offset)

	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	result := ListResult{Entries: []LogEntry{}, Page: f.Page, PageSize: f.PageSize}
	for rows.Next() {
		var e LogEntry
		var total int
		if err := rows.Scan(&e.ID, &e.Action, &e.ResourceType, &e.ResourceID,
			&e.UserID, &e.UserName, &e.Metadata, &e.CreatedAt, &total); err != nil {
			return ListResult{}, fmt.Errorf("scan audit log: %w", err)
		}
		result.Total = total
		result.Entries = append(result.Entries, e)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("iterate audit logs: %w", err)
	}
	return result, nil
}

// Handler exposes the audit log read endpoint.
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler builds the audit HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// List handles GET /api/v1/tenants/{tenantId}/audit-logs.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.TenantIDFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
		return
	}

	f, uerr := parseListQuery(r)
	if uerr != nil {
		httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
		return
	}

	res, err := h.svc.List(r.Context(), tenantID, f)
	if err != nil {
		h.log.Error("audit list failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
		return
	}
	httpx.Success(w, http.StatusOK, res)
}

// parseListQuery reads and validates the history filters (PRD 9.10).
func parseListQuery(r *http.Request) (ListFilter, *httpx.UserError) {
	q := r.URL.Query()
	f := ListFilter{Page: 1, PageSize: 20}

	if v := q.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			return f, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "page must be a positive integer"}
		}
		f.Page = p
	}
	if v := q.Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > 100 {
			return f, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "page_size must be 1-100"}
		}
		f.PageSize = ps
	}

	f.Action = strings.TrimSpace(q.Get("action"))
	f.ResourceType = strings.TrimSpace(q.Get("resource_type"))

	if v := q.Get("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return f, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "user_id must be a UUID"}
		}
		f.UserID = &id
	}
	if v := q.Get("start_date"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "start_date must be YYYY-MM-DD"}
		}
		f.StartDate = d
	}
	if v := q.Get("end_date"); v != "" {
		d, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "end_date must be YYYY-MM-DD"}
		}
		f.EndDate = d
	}
	return f, nil
}
