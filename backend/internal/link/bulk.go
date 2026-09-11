// bulk.go — bulk link creation and CSV import/export (PRD 9.4.6–9.4.8).
package link

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"linkpulse/internal/httpx"
)

// Bulk limits per PRD 9.4.6 / 9.4.8.
const (
	maxBulkCreate = 100
	maxImportRows = 500
)

// BulkItemResult reports one row of a bulk create / import operation.
type BulkItemResult struct {
	Index       int      `json:"index"`
	OK          bool     `json:"ok"`
	ShortCode   string   `json:"short_code,omitempty"`
	ShortURL    string   `json:"short_url,omitempty"`
	Errors      []string `json:"errors,omitempty"`
	Destination string   `json:"destination_url,omitempty"`
	Title       string   `json:"title,omitempty"`
}

// BulkResult summarizes a bulk operation: per-item outcomes plus totals.
type BulkResult struct {
	Total   int              `json:"total"`
	Success int              `json:"success"`
	Failed  int              `json:"failed"`
	Items   []BulkItemResult `json:"items"`
}

// BulkCreate makes up to 100 links in one transaction-per-item batch.
// Per PRD 9.4.6: partial failures are reported per item, not fatal.
func (s *Service) BulkCreate(ctx context.Context, tenantID, userID uuid.UUID, inputs []CreateInput) (BulkResult, error) {
	if len(inputs) == 0 {
		return BulkResult{}, &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: "links must contain at least one item",
		}
	}
	if len(inputs) > maxBulkCreate {
		return BulkResult{}, &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: fmt.Sprintf("bulk create is limited to %d links per request", maxBulkCreate),
		}
	}

	res := BulkResult{Total: len(inputs), Items: make([]BulkItemResult, 0, len(inputs))}
	for i, in := range inputs {
		item := BulkItemResult{Index: i}
		out, err := s.Create(ctx, tenantID, userID, in)
		if err != nil {
			item.OK = false
			item.Errors = []string{userErrorMessage(err)}
			item.Destination = in.DestinationURL
			item.Title = in.Title
			res.Failed++
		} else {
			item.OK = true
			item.ShortCode = out.ShortCode
			item.ShortURL = out.ShortURL
			item.Destination = out.DestinationURL
			item.Title = out.Title
			res.Success++
		}
		res.Items = append(res.Items, item)
	}
	return res, nil
}

// ImportCSVRow is one parsed row of an imported CSV.
type ImportCSVRow struct {
	DestinationURL string
	Title          string
	CustomCode     string
	Tags           []string
}

// ImportResult summarizes a CSV import (PRD 9.4.8): per-row validation
// errors are collected, valid rows are created.
func (s *Service) ImportCSV(ctx context.Context, tenantID, userID uuid.UUID, csvData []byte) (BulkResult, error) {
	rows, uerr := ParseImportCSV(csvData)
	if uerr != nil {
		return BulkResult{}, uerr
	}
	if len(rows) == 0 {
		return BulkResult{}, &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: "CSV contains no data rows",
		}
	}
	if len(rows) > maxImportRows {
		return BulkResult{}, &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: fmt.Sprintf("import is limited to %d rows per request (got %d)", maxImportRows, len(rows)),
		}
	}

	res := BulkResult{Total: len(rows), Items: make([]BulkItemResult, 0, len(rows))}
	for i, row := range rows {
		item := BulkItemResult{Index: i}
		out, err := s.Create(ctx, tenantID, userID, CreateInput{
			DestinationURL: row.DestinationURL,
			Title:          row.Title,
			CustomCode:     row.CustomCode,
			Tags:           row.Tags,
		})
		if err != nil {
			item.OK = false
			item.Errors = []string{userErrorMessage(err)}
			item.Destination = row.DestinationURL
			item.Title = row.Title
			res.Failed++
		} else {
			item.OK = true
			item.ShortCode = out.ShortCode
			item.ShortURL = out.ShortURL
			item.Destination = out.DestinationURL
			item.Title = out.Title
			res.Success++
		}
		res.Items = append(res.Items, item)
	}
	return res, nil
}

// ParseImportCSV parses import CSV data. The header row is required and
// must contain at least destination_url. Columns are matched by name, so
// order does not matter.
func ParseImportCSV(data []byte) ([]ImportCSVRow, *httpx.UserError) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1 // tolerate ragged rows; validated below
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: "CSV is empty",
		}
	}
	if err != nil {
		return nil, &httpx.UserError{
			Status:  http.StatusBadRequest,
			Code:    httpx.CodeBadRequest,
			Message: "CSV could not be parsed",
		}
	}

	colIdx := map[string]int{}
	for i, name := range header {
		colIdx[strings.ToLower(strings.TrimSpace(name))] = i
	}
	destCol, hasDest := colIdx["destination_url"]
	if !hasDest {
		return nil, &httpx.UserError{
			Status:  http.StatusUnprocessableEntity,
			Code:    httpx.CodeValidationError,
			Message: "CSV header must contain a destination_url column",
		}
	}
	titleCol, hasTitle := colIdx["title"]
	codeCol, hasCode := colIdx["custom_code"]
	tagsCol, hasTags := colIdx["tags"]

	var out []ImportCSVRow
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, &httpx.UserError{
				Status:  http.StatusBadRequest,
				Code:    httpx.CodeBadRequest,
				Message: "CSV could not be parsed (check quoting)",
			}
		}
		if len(rec) == 0 {
			continue
		}
		row := ImportCSVRow{}
		if destCol < len(rec) {
			row.DestinationURL = strings.TrimSpace(rec[destCol])
		}
		if row.DestinationURL == "" && len(rec) == 1 && rec[0] == "" {
			continue // skip fully blank lines
		}
		if hasTitle && titleCol < len(rec) {
			row.Title = strings.TrimSpace(rec[titleCol])
		}
		if hasCode && codeCol < len(rec) {
			row.CustomCode = strings.TrimSpace(rec[codeCol])
		}
		if hasTags && tagsCol < len(rec) {
			row.Tags = ParseTagsField(rec[tagsCol])
		}
		out = append(out, row)
	}
	return out, nil
}

// ParseTagsField splits a CSV tags cell ("a; b;c" or "a,b,c") into clean,
// lowercase tags. Semicolons are preferred because the comma is the CSV
// delimiter, but both are accepted for user convenience.
func ParseTagsField(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	sep := ";"
	if strings.Contains(raw, ",") && !strings.Contains(raw, ";") {
		sep = ","
	}
	parts := strings.Split(raw, sep)
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.ToLower(strings.TrimSpace(p)); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// ExportCSVHeader is the canonical export header (PRD 9.4.7 columns).
var ExportCSVHeader = []string{
	"short_code", "short_url", "title", "destination_url", "status",
	"click_count", "created_at", "expires_at", "tags",
}

// ExportCSV serializes links to CSV. Uses the LinkOut shape — password
// hashes are never exported.
func (s *Service) ExportCSV(links []LinkOut) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(ExportCSVHeader); err != nil {
		return nil, fmt.Errorf("write CSV header: %w", err)
	}
	for _, l := range links {
		expires := ""
		if l.ExpiresAt != nil {
			expires = l.ExpiresAt.UTC().Format(time.RFC3339)
		}
		rec := []string{
			l.ShortCode,
			l.ShortURL,
			l.Title,
			l.DestinationURL,
			l.Status,
			fmt.Sprintf("%d", l.ClickCount),
			l.CreatedAt.UTC().Format(time.RFC3339),
			expires,
			strings.Join(l.Tags, ";"),
		}
		if err := w.Write(rec); err != nil {
			return nil, fmt.Errorf("write CSV row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush CSV: %w", err)
	}
	return buf.Bytes(), nil
}

// userErrorMessage maps an error to a safe per-row message. UserError
// values carry client-safe text; anything else is internal and generic.
func userErrorMessage(err error) string {
	var uerr *httpx.UserError
	if ok := asUserError(err, &uerr); ok {
		return uerr.Message
	}
	return "internal error creating link"
}

// asUserError is a tiny errors.As wrapper so bulk.go stays free of an
// errors import used once.
func asUserError(err error, target **httpx.UserError) bool {
	for err != nil {
		if u, is := err.(*httpx.UserError); is {
			*target = u
			return true
		}
		unw, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unw.Unwrap()
	}
	return false
}
