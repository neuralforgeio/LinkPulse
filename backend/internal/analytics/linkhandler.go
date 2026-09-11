// linkhandler.go — per-link analytics endpoints (PRD 9.6.2, 9.6.3).
package analytics

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"linkpulse/internal/httpx"
	"linkpulse/internal/tenant"
)

// LinkAnalytics handles GET /api/v1/tenants/{tenantId}/links/{linkId}/analytics.
func (h *Handler) LinkAnalytics(w http.ResponseWriter, r *http.Request) {
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid link id")
		return
	}
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

	res, err := h.svc.LinkAnalytics(r.Context(), tenantID, linkID, p)
	if err != nil {
		var uerr *httpx.UserError
		if errors.As(err, &uerr) {
			httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
			return
		}
		h.log.Error("link analytics failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
		return
	}
	httpx.Success(w, http.StatusOK, res)
}

// RecentClicks handles GET /api/v1/tenants/{tenantId}/links/{linkId}/clicks.
func (h *Handler) RecentClicks(w http.ResponseWriter, r *http.Request) {
	linkID, err := uuid.Parse(chi.URLParam(r, "linkId"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid link id")
		return
	}
	tenantID, ok := tenant.TenantIDFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
		return
	}

	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p >= 1 {
			page = p
		}
	}
	pageSize := 20
	if v := r.URL.Query().Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps >= 1 && ps <= 100 {
			pageSize = ps
		}
	}

	res, err := h.svc.RecentClicks(r.Context(), tenantID, linkID, page, pageSize)
	if err != nil {
		h.log.Error("recent clicks failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
		return
	}
	httpx.Success(w, http.StatusOK, res)
}
