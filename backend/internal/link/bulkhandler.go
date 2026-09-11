// bulkhandler.go — HTTP handlers for bulk create, CSV import and CSV
// export (PRD 9.4.6–9.4.8).
package link

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"linkpulse/internal/httpx"
	"linkpulse/internal/tenant"
)

// BulkCreate handles POST /api/v1/tenants/{tenantId}/links/bulk.
// Body: {"links": [CreateInput...]}. Manager roles only.
func (h *Handler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	if rejectNonManager(w, r) {
		return
	}
	userID, okU := httpx.UserIDFrom(r.Context())
	tenantID, okT := tenant.TenantIDFrom(r.Context())
	if !okU || !okT {
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
		return
	}

	// 100 links * up to ~2KB each is a generous but bounded body.
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	var body struct {
		Links []CreateInput `json:"links"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
		return
	}

	res, err := h.svc.BulkCreate(r.Context(), tenantID, userID, body.Links)
	if err != nil {
		h.handleErr(w, err, "bulk create")
		return
	}
	httpx.Success(w, http.StatusCreated, res)
}

// Import handles POST /api/v1/tenants/{tenantId}/links/import.
// Accepts multipart/form-data with a "file" field (CSV), or a JSON body
// {"csv": "destination_url,title,..."} for API-driven imports.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if rejectNonManager(w, r) {
		return
	}
	userID, okU := httpx.UserIDFrom(r.Context())
	tenantID, okT := tenant.TenantIDFrom(r.Context())
	if !okU || !okT {
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
		return
	}

	var csvData []byte
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20) // 2 MiB cap
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid multipart form")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError, "form field 'file' with a CSV is required")
			return
		}
		defer file.Close()
		csvData, err = io.ReadAll(io.LimitReader(file, 2<<20))
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "could not read uploaded file")
			return
		}
	} else {
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		var body struct {
			CSV string `json:"csv"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CSV == "" {
			httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError,
				"send multipart/form-data with a 'file' field, or JSON {\"csv\": \"...\"}")
			return
		}
		csvData = []byte(body.CSV)
	}

	res, err := h.svc.ImportCSV(r.Context(), tenantID, userID, csvData)
	if err != nil {
		h.handleErr(w, err, "csv import")
		return
	}
	httpx.Success(w, http.StatusCreated, res)
}

// Export handles GET /api/v1/tenants/{tenantId}/links/export?format=csv|json.
// All links (deleted excluded), respecting the same filters as List minus
// pagination. Viewer roles can export (read operation per PRD 9.4.7).
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenant.TenantIDFrom(r.Context())
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		httpx.Error(w, http.StatusUnprocessableEntity, httpx.CodeValidationError, "format must be csv or json")
		return
	}

	// Reuse list filters (search/status/tag/dates) but force a single
	// large page — export is capped at 1000 rows, orders of magnitude
	// above a portfolio workspace's size.
	f, uerr := parseListQuery(r)
	if uerr != nil {
		httpx.Error(w, uerr.Status, uerr.Code, uerr.Message)
		return
	}
	f.Page = 1
	f.PageSize = parseExportLimit(r)

	res, err := h.svc.List(r.Context(), tenantID, f)
	if err != nil {
		h.handleErr(w, err, "link export")
		return
	}

	// The audit middleware records mutations only; reads that export
	// data are worth auditing too (PRD 9.10 "export links"), so the
	// route wraps this handler with an explicit track action.

	if format == "json" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=links-%s.json", time.Now().UTC().Format("20060102")))
		httpx.Success(w, http.StatusOK, res.Links)
		return
	}

	csvBytes, err := h.svc.ExportCSV(res.Links)
	if err != nil {
		h.log.Error("csv export failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=links-%s.csv", time.Now().UTC().Format("20060102")))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvBytes)
}

// parseExportLimit caps the export row count (default 1000, PRD-sized
// workspaces stay far below this).
func parseExportLimit(r *http.Request) int {
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			return n
		}
	}
	return 1000
}
