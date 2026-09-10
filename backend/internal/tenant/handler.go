package tenant

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"linkpulse/internal/httpx"
)

// Handler exposes tenant endpoints over HTTP.
type Handler struct {
    svc *Service
    log *slog.Logger
}

// NewHandler builds the tenant HTTP handler.
func NewHandler(svc *Service, log *slog.Logger) *Handler {
    return &Handler{svc: svc, log: log}
}

// handleErr maps service errors to responses.
func (h *Handler) handleErr(w http.ResponseWriter, err error, action string) {
    var uerr *httpx.UserError
    if errors.As(err, &uerr) {
        httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
        return
    }
    h.log.Error(action + " failed", "error", err)
    httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
}

// Create handles POST /api/v1/tenants.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    userID, ok := httpx.UserIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 4096)
    var in CreateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.Create(r.Context(), userID, in)
    if err != nil {
        h.handleErr(w, err, "tenant create")
        return
    }
    httpx.Success(w, http.StatusCreated, res)
}

// List handles GET /api/v1/tenants.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    userID, ok := httpx.UserIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    res, err := h.svc.List(r.Context(), userID)
    if err != nil {
        h.handleErr(w, err, "tenant list")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// Get handles GET /api/v1/tenants/{tenantId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
    userID, ok := httpx.UserIDFrom(r.Context())
    tenantID, okT := TenantIDFrom(r.Context())
    if !ok || !okT {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    res, err := h.svc.Get(r.Context(), userID, tenantID)
    if err != nil {
        h.handleErr(w, err, "tenant get")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// Update handles PATCH /api/v1/tenants/{tenantId} (owner/admin only).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
    role, ok := RoleFrom(r.Context())
    if !ok || (role != RoleOwner && role != RoleAdmin) {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "requires owner or admin role")
        return
    }

    userID, okU := httpx.UserIDFrom(r.Context())
    tenantID, okT := TenantIDFrom(r.Context())
    if !okU || !okT {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 8192)
    var in UpdateInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.Update(r.Context(), userID, tenantID, in)
    if err != nil {
        h.handleErr(w, err, "tenant update")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}
