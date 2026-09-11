package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"linkpulse/internal/apikey"
	"linkpulse/internal/httpx"
	"linkpulse/internal/tenant"
)

// Service records and reads audit events.
type Service struct {
	db     *pgxpool.Pool
	log    *slog.Logger
	ipSalt string
}

// NewService builds an audit Service. ipSalt salts the IP hash — raw
// IPs are never stored (PRD 9.10, 16.11).
func NewService(db *pgxpool.Pool, log *slog.Logger, ipSalt string) *Service {
	return &Service{db: db, log: log, ipSalt: ipSalt}
}

// Event is one audit record.
type Event struct {
	TenantID     *uuid.UUID
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   string
	Metadata     map[string]any
	IPHash       string
	UserAgent    string
}

// Record persists one event. Fire-and-forget by design: errors are
// logged, never propagated — auditing must not break the main flow.
func (s *Service) Record(ctx context.Context, ev Event) {
	if ev.Metadata == nil {
		ev.Metadata = map[string]any{}
	}
	meta, err := json.Marshal(ev.Metadata)
	if err != nil {
		meta = []byte("{}")
	}

	_, err = s.db.Exec(ctx, `
        INSERT INTO audit_logs (id, tenant_id, user_id, action, resource_type,
            resource_id, metadata, ip_hash, user_agent)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.New(), ev.TenantID, ev.UserID, ev.Action, ev.ResourceType,
		ev.ResourceID, meta, ev.IPHash, ev.UserAgent)
	if err != nil {
		s.log.Error("audit record failed", "error", err, "action", ev.Action)
	}
}

// ---- capture plumbing ----

// teeWriter captures the status code and a bounded slice of the body,
// so the middleware can read created-resource IDs from the envelope.
type teeWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

// maxCapture is plenty for single-resource mutation responses.
const maxCapture = 8192

func (t *teeWriter) WriteHeader(code int) {
	t.status = code
	t.ResponseWriter.WriteHeader(code)
}

func (t *teeWriter) Write(b []byte) (int, error) {
	if t.body.Len() < maxCapture {
		t.body.Write(b)
	}
	return t.ResponseWriter.Write(b)
}

func isMutation(method string) bool {
	return method == http.MethodPost || method == http.MethodPatch || method == http.MethodDelete
}

// extractBodyIDs pulls user/tenant/resource identifiers out of a
// standard response envelope.
func extractBodyIDs(body []byte) (userID, tenantID, resourceID string) {
	var env struct {
		Data struct {
			ID         string `json:"id"`
			InviteCode string `json:"invite_code"`
			User       struct {
				ID string `json:"id"`
			} `json:"user"`
			Tenant struct {
				ID string `json:"id"`
			} `json:"tenant"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &env) != nil {
		return "", "", ""
	}
	d := env.Data
	switch {
	case d.ID != "":
		resourceID = d.ID
	case d.InviteCode != "":
		resourceID = d.InviteCode
	}
	return d.User.ID, d.Tenant.ID, resourceID
}

// classify maps (method, route pattern) to an audit action. Patterns
// come from chi, e.g. "/api/v1/tenants/{tenantId}/links/{linkId}" and
// "/api/v1/public/links/{shortCode}".
func classify(method, pattern string) (action, resourceType, resourceKey string, fromBody bool) {
	switch {
	case strings.Contains(pattern, "/links"):
		switch method {
		case http.MethodPost:
			return "link.create", "link", "", true
		case http.MethodPatch:
			return "link.update", "link", "linkId", false
		case http.MethodDelete:
			return "link.delete", "link", "linkId", false
		}
	case strings.Contains(pattern, "/members"):
		switch method {
		case http.MethodPatch:
			return "member.role_update", "member", "userId", false
		case http.MethodDelete:
			return "member.remove", "member", "userId", false
		}
	case strings.Contains(pattern, "/api-keys"):
		switch method {
		case http.MethodPost:
			return "api_key.create", "api_key", "", true
		case http.MethodDelete:
			return "api_key.revoke", "api_key", "keyId", false
		}
	case strings.HasSuffix(pattern, "/invitations") && method == http.MethodPost:
		return "invitation.create", "invitation", "", true
	case strings.HasSuffix(pattern, "/leave") && method == http.MethodPost:
		return "member.leave", "member", "", false
	case strings.HasSuffix(pattern, "/{tenantId}") && method == http.MethodPatch:
		return "tenant.update", "tenant", "tenantId", false
	}
	return "", "", "", false
}

// TrackMutations records every successful mutation (2xx POST/PATCH/
// DELETE). Mount after the auth/membership middleware: the tenant and
// caller are read from the context. Covers both session routes and the
// public API (via the API key's identity).
func (s *Service) TrackMutations(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMutation(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		tee := &teeWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(tee, r)

		// Only successful mutations are audited.
		if tee.status < 200 || tee.status >= 300 {
			return
		}

		pattern := chi.RouteContext(r.Context()).RoutePattern()
		action, resourceType, resourceKey, fromBody := classify(r.Method, pattern)
		if action == "" {
			return
		}

		resourceID := ""
		if fromBody {
			_, _, resourceID = extractBodyIDs(tee.body.Bytes())
		} else if resourceKey != "" {
			resourceID = chi.URLParam(r, resourceKey)
			if resourceID == "" {
				// Public API routes name the param {shortCode}.
				resourceID = chi.URLParam(r, "shortCode")
			}
		}

		// Identity: session first, API key as fallback (public API).
		tenantID, tenantOK := tenant.TenantIDFrom(r.Context())
		userID, userOK := httpx.UserIDFrom(r.Context())
		viaAPIKey := false
		if !tenantOK || !userOK {
			if a, ok := apikey.FromContext(r.Context()); ok {
				if !tenantOK {
					tenantID = a.TenantID
					tenantOK = true
				}
				if !userOK {
					userID = a.CreatedBy
					userOK = true
					viaAPIKey = true
				}
			}
		}

		var tid, uid *uuid.UUID
		if tenantOK {
			tid = &tenantID
		}
		if userOK {
			uid = &userID
			if action == "member.leave" {
				resourceID = uid.String()
			}
		}

		meta := map[string]any{"method": r.Method, "path": pattern}
		if viaAPIKey {
			meta["via"] = "api_key"
		}

		s.Record(r.Context(), Event{
			TenantID:     tid,
			UserID:       uid,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Metadata:     meta,
			IPHash:       hashIP(r.RemoteAddr, s.ipSalt),
			UserAgent:    r.UserAgent(),
		})
	})
}

// TrackAction wraps one route with an explicit action — for events the
// pattern classifier cannot see (login, register, logout, tenant
// create, invite acceptance). User and tenant IDs come from the
// response envelope, with the authenticated caller as fallback.
func (s *Service) TrackAction(action, resourceType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tee := &teeWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(tee, r)

			if tee.status < 200 || tee.status >= 300 {
				return
			}

			bodyUserID, bodyTenantID, resourceID := extractBodyIDs(tee.body.Bytes())

			var uid *uuid.UUID
			if bodyUserID != "" {
				if id, err := uuid.Parse(bodyUserID); err == nil {
					uid = &id
				}
			}
			var tid *uuid.UUID
			if bodyTenantID != "" {
				if id, err := uuid.Parse(bodyTenantID); err == nil {
					tid = &id
				}
			}
			// Fallback: the authenticated caller (e.g. invite accept).
			if uid == nil {
				if id, ok := httpx.UserIDFrom(r.Context()); ok {
					uid = &id
				}
			}

			s.Record(r.Context(), Event{
				TenantID:     tid,
				UserID:       uid,
				Action:       action,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Metadata:     map[string]any{"method": r.Method},
				IPHash:       hashIP(r.RemoteAddr, s.ipSalt),
				UserAgent:    r.UserAgent(),
			})
		})
	}
}

// hashIP converts "host:port" into a salted SHA-256 hash — raw IPs are
// never stored.
func hashIP(remoteAddr, salt string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	sum := sha256.Sum256([]byte(salt + host))
	return hex.EncodeToString(sum[:])
}
