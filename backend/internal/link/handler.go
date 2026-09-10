package link

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"linkpulse/internal/httpx"
	"linkpulse/internal/tenant"
)

// canManageLinks: viewer is read-only (PRD 9.3 permission matrix).
func canManageLinks(role string) bool {
    return role == tenant.RoleOwner || role == tenant.RoleAdmin || role == tenant.RoleMember
}

// Handler exposes link endpoints over HTTP.
type Handler struct {
    svc *Service
    log *slog.Logger
}

// NewHandler builds the link HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
    return &Handler{svc: svc, log: log}
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

func rejectNonManager(w http.ResponseWriter, r *http.Request) bool {
    role, ok := tenant.RoleFrom(r.Context())
    if !ok || !canManageLinks(role) {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "requires member role or above")
        return true
    }
    return false
}

// Create handles POST /api/v1/tenants/{tenantId}/links.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    if rejectNonManager(w, r) {
        return
    }
    userID, okU := httpx.UserIDFrom(r.Context())
    tenantID, okT := tenant.TenantIDFrom(r.Context())
    if !okU || !okT {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var in CreateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.Create(r.Context(), tenantID, userID, in)
    if err != nil {
        h.handleErr(w, err, "link create")
        return
    }
    httpx.Success(w, http.StatusCreated, res)
}

// List handles GET /api/v1/tenants/{tenantId}/links.
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
        h.handleErr(w, err, "link list")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// Get handles GET /api/v1/tenants/{tenantId}/links/{linkId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
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

    res, err := h.svc.Get(r.Context(), tenantID, linkID)
    if err != nil {
        h.handleErr(w, err, "link get")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// Update handles PATCH /api/v1/tenants/{tenantId}/links/{linkId}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
    if rejectNonManager(w, r) {
        return
    }
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

    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var in UpdateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.Update(r.Context(), tenantID, linkID, in)
    if err != nil {
        h.handleErr(w, err, "link update")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// Delete handles DELETE /api/v1/tenants/{tenantId}/links/{linkId}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
    if rejectNonManager(w, r) {
        return
    }
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

    if err := h.svc.Delete(r.Context(), tenantID, linkID); err != nil {
        h.handleErr(w, err, "link delete")
        return
    }
    httpx.Success(w, http.StatusOK, map[string]string{"message": "link deleted"})
}

// parseListQuery reads and validates the list filters (PRD 9.4.2).
func parseListQuery(r *http.Request) (ListFilter, *httpx.UserError) {
    q := r.URL.Query()
    f := ListFilter{Page: 1, PageSize: 20, SortBy: "created_at", SortOrder: "desc"}

    if v := q.Get("page"); v != "" {
        p, err := strconv.Atoi(v)
        if err != nil || p < 1 {
            return f, &httpx.UserError{422, httpx.CodeValidationError, "page must be a positive integer"}
        }
        f.Page = p
    }
    if v := q.Get("page_size"); v != "" {
        ps, err := strconv.Atoi(v)
        if err != nil || ps < 1 || ps > 100 {
            return f, &httpx.UserError{422, httpx.CodeValidationError, "page_size must be 1-100"}
        }
        f.PageSize = ps
    }

    f.Search = strings.TrimSpace(q.Get("search"))
    f.Tag = strings.TrimSpace(q.Get("tag"))

    f.Status = q.Get("status")
    if f.Status != "" {
        if _, ok := statusFilters[f.Status]; !ok {
            return f, &httpx.UserError{422, httpx.CodeValidationError,
                "status must be one of: active, expired, disabled, max_clicks_reached, password_protected"}
        }
    }

    f.SortBy = q.Get("sort_by")
    if f.SortBy == "" {
        f.SortBy = "created_at"
    }
    if _, ok := sortColumns[f.SortBy]; !ok {
        return f, &httpx.UserError{422, httpx.CodeValidationError,
            "sort_by must be one of: created_at, click_count, title"}
    }

    f.SortOrder = q.Get("sort_order")
    if f.SortOrder == "" {
        f.SortOrder = "desc"
    }
    if f.SortOrder != "asc" && f.SortOrder != "desc" {
        return f, &httpx.UserError{422, httpx.CodeValidationError, "sort_order must be asc or desc"}
    }

    if v := q.Get("start_date"); v != "" {
        d, err := time.Parse("2006-01-02", v)
        if err != nil {
            return f, &httpx.UserError{422, httpx.CodeValidationError, "start_date must be YYYY-MM-DD"}
        }
        f.StartDate = d
    }
    if v := q.Get("end_date"); v != "" {
        d, err := time.Parse("2006-01-02", v)
        if err != nil {
            return f, &httpx.UserError{422, httpx.CodeValidationError, "end_date must be YYYY-MM-DD"}
        }
        f.EndDate = d
    }

    return f, nil
}
