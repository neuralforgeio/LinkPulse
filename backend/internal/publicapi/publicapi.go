package publicapi

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/apikey"
    "linkpulse/internal/httpx"
    "linkpulse/internal/link"
)

// Handler serves /api/v1/public/*.
type Handler struct {
    db    *pgxpool.Pool
    links *link.Service
    log   *slog.Logger
}

// NewHandler builds the public API handler.
func NewHandler(db *pgxpool.Pool, links *link.Service, log *slog.Logger) *Handler {
    return &Handler{db: db, links: links, log: log}
}

func (h *Handler) handleErr(w http.ResponseWriter, err error, action string) {
    var uerr *httpx.UserError
    if errors.As(err, &uerr) {
        httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
        return
    }
    h.log.Error(action + " failed", "error", err)
    httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
}

func (h *Handler) auth(r *http.Request) (*apikey.Auth, bool) {
    return apikey.FromContext(r.Context())
}

// findIDByCode resolves a short code within the key's tenant.
func (h *Handler) findIDByCode(ctx context.Context, tenantID uuid.UUID, code string) (uuid.UUID, error) {
    var id uuid.UUID
    err := h.db.QueryRow(ctx, `
        SELECT id FROM links
        WHERE short_code = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
        code, tenantID).Scan(&id)
    if errors.Is(err, pgx.ErrNoRows) {
        return id, &httpx.UserError{
            Status:  http.StatusNotFound,
            Code:    httpx.CodeNotFound,
            Message: "link not found",
        }
    }
    if err != nil {
        return id, fmt.Errorf("resolve link: %w", err)
    }
    return id, nil
}

// CreateLink handles POST /api/v1/public/links (links:write).
func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
    auth, ok := h.auth(r)
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var in link.CreateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.links.Create(r.Context(), auth.TenantID, auth.CreatedBy, in)
    if err != nil {
        h.handleErr(w, err, "public link create")
        return
    }
    httpx.Success(w, http.StatusCreated, res)
}

// ListLinks handles GET /api/v1/public/links (links:read).
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
    auth, ok := h.auth(r)
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    q := r.URL.Query()
    f := link.ListFilter{Page: 1, PageSize: 20, SortBy: "created_at", SortOrder: "desc"}
    if v := q.Get("page"); v != "" {
        if p, err := strconv.Atoi(v); err == nil && p >= 1 {
            f.Page = p
        }
    }
    if v := q.Get("page_size"); v != "" {
        if ps, err := strconv.Atoi(v); err == nil && ps >= 1 && ps <= 100 {
            f.PageSize = ps
        }
    }
    f.Search = strings.TrimSpace(q.Get("search"))
    f.Status = q.Get("status")

    res, err := h.links.List(r.Context(), auth.TenantID, f)
    if err != nil {
        h.handleErr(w, err, "public link list")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// GetLink handles GET /api/v1/public/links/{shortCode} (links:read).
func (h *Handler) GetLink(w http.ResponseWriter, r *http.Request) {
    auth, ok := h.auth(r)
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    linkID, err := h.findIDByCode(r.Context(), auth.TenantID, chi.URLParam(r, "shortCode"))
    if err != nil {
        h.handleErr(w, err, "public link resolve")
        return
    }

    res, err := h.links.Get(r.Context(), auth.TenantID, linkID)
    if err != nil {
        h.handleErr(w, err, "public link get")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// UpdateLink handles PATCH /api/v1/public/links/{shortCode} (links:write).
func (h *Handler) UpdateLink(w http.ResponseWriter, r *http.Request) {
    auth, ok := h.auth(r)
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    linkID, err := h.findIDByCode(r.Context(), auth.TenantID, chi.URLParam(r, "shortCode"))
    if err != nil {
        h.handleErr(w, err, "public link resolve")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var in link.UpdateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.links.Update(r.Context(), auth.TenantID, linkID, in)
    if err != nil {
        h.handleErr(w, err, "public link update")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// DeleteLink handles DELETE /api/v1/public/links/{shortCode} (links:write).
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
    auth, ok := h.auth(r)
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    linkID, err := h.findIDByCode(r.Context(), auth.TenantID, chi.URLParam(r, "shortCode"))
    if err != nil {
        h.handleErr(w, err, "public link resolve")
        return
    }

    if err := h.links.Delete(r.Context(), auth.TenantID, linkID); err != nil {
        h.handleErr(w, err, "public link delete")
        return
    }
    httpx.Success(w, http.StatusOK, map[string]string{"message": "link deleted"})
}

// LinkAnalytics handles GET /api/v1/public/links/{shortCode}/analytics
// (analytics:read) — click_count plus the last 30 days as a daily series.
func (h *Handler) LinkAnalytics(w http.ResponseWriter, r *http.Request) {
    auth, ok := h.auth(r)
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    code := chi.URLParam(r, "shortCode")
    linkID, err := h.findIDByCode(r.Context(), auth.TenantID, code)
    if err != nil {
        h.handleErr(w, err, "public link resolve")
        return
    }

    // Current state (includes click_count).
    current, err := h.links.Get(r.Context(), auth.TenantID, linkID)
    if err != nil {
        h.handleErr(w, err, "public link get")
        return
    }

    // Daily series, last 30 days, UTC buckets, zero-filled.
    now := time.Now().UTC()
    start := now.AddDate(0, 0, -30)
    rows, err := h.db.Query(r.Context(), `
        SELECT (clicked_at AT TIME ZONE 'UTC')::date, COUNT(*)
        FROM click_events
        WHERE link_id = $1 AND clicked_at >= $2
        GROUP BY 1`, linkID, start)
    if err != nil {
        h.handleErr(w, err, "public link analytics")
        return
    }
    defer rows.Close()

    counts := map[string]int64{}
    for rows.Next() {
        var day time.Time
        var clicks int64
        if err := rows.Scan(&day, &clicks); err != nil {
            h.handleErr(w, err, "public link analytics scan")
            return
        }
        counts[day.Format("2006-01-02")] = clicks
    }
    if err := rows.Err(); err != nil {
        h.handleErr(w, err, "public link analytics iterate")
        return
    }

    type point struct {
        Date   string `json:"date"`
        Clicks int64  `json:"clicks"`
    }
    points := []point{}
    for d := start; d.Before(now); d = d.AddDate(0, 0, 1) {
        key := d.Format("2006-01-02")
        points = append(points, point{Date: key, Clicks: counts[key]})
    }

    httpx.Success(w, http.StatusOK, map[string]any{
        "link_id":          linkID,
        "short_code":       code,
        "click_count":      current.ClickCount,
        "clicks_over_time": points,
    })
}
