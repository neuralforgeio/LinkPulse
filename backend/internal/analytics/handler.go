package analytics

// handler.go — GET /api/v1/tenants/{tenantId}/analytics/overview
// (PRD 9.6.1).

import (
    "log/slog"
    "net/http"
    "time"

    "linkpulse/internal/httpx"
    "linkpulse/internal/tenant"
)

// Handler exposes analytics endpoints over HTTP.
type Handler struct {
    svc *Service
    log *slog.Logger
}

// NewHandler builds the analytics HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
    return &Handler{svc: svc, log: log}
}

// Overview handles GET /api/v1/tenants/{tenantId}/analytics/overview.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
    tenantID, ok := tenant.TenantIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    p, uerr := parseParams(r)
    if uerr != nil {
        httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
        return
    }

    res, err := h.svc.Overview(r.Context(), tenantID, p)
    if err != nil {
        h.log.Error("analytics overview failed", "error", err)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// parseParams reads and validates start_date, end_date, granularity.
// Defaults: last 30 days, daily buckets. Dates are UTC; the range is
// capped at 366 days.
func parseParams(r *http.Request) (Params, *httpx.UserError) {
    q := r.URL.Query()
    p := Params{Granularity: GranularityDay}

    end := time.Now().UTC()
    start := end.AddDate(0, 0, -30)

    if v := q.Get("start_date"); v != "" {
        d, err := time.Parse("2006-01-02", v)
        if err != nil {
            return p, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "start_date must be YYYY-MM-DD"}
        }
        start = d
    }
    if v := q.Get("end_date"); v != "" {
        d, err := time.Parse("2006-01-02", v)
        if err != nil {
            return p, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "end_date must be YYYY-MM-DD"}
        }
        end = d.Add(24 * time.Hour) // end_date is inclusive
    }
    if !end.After(start) {
        return p, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "end_date must be after start_date"}
    }
    if end.Sub(start) > 366*24*time.Hour {
        return p, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "date range too large (max 366 days)"}
    }

    if v := q.Get("granularity"); v != "" {
        switch v {
        case GranularityDay, GranularityWeek, GranularityMonth:
            p.Granularity = v
        default:
            return p, &httpx.UserError{http.StatusUnprocessableEntity, httpx.CodeValidationError, "granularity must be day, week, or month"}
        }
    }

    p.Start = start
    p.End = end
    return p, nil
}
