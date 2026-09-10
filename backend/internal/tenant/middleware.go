package tenant

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"linkpulse/internal/httpx"
)

// Context keys for the tenant scope set by RequireMembership.
type (
    tenantIDKey struct{}
    roleKey     struct{}
)

// RequireMembership gates a tenant-scoped route: the caller must be a
// member of the tenant in the URL ({tenantId}). When roles are given,
// the caller's role must be one of them. On success the tenant ID and
// role are stored in the request context for the handler.
//
// Every future tenant-scoped module (links, analytics, API keys) mounts
// under this same gate, so role enforcement cannot be forgotten.
func (s *Service) RequireMembership(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID, ok := httpx.UserIDFrom(r.Context())
            if !ok {
                // RequireAuth always runs before this gate.
                httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
                return
            }

            rawID := chi.URLParam(r, "tenantId")
            tenantID, err := uuid.Parse(rawID)
            if err != nil {
                httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid tenant id")
                return
            }

            var role string
            err = s.db.QueryRow(r.Context(),
                `SELECT role FROM memberships WHERE tenant_id = $1 AND user_id = $2`,
                tenantID, userID).Scan(&role)
            if errors.Is(err, pgx.ErrNoRows) {
                httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "you are not a member of this workspace")
                return
            }
            if err != nil {
                s.log.Error("membership check failed",
                    "error", err,
                    "request_id", middleware.GetReqID(r.Context()))
                httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
                return
            }

            if len(roles) > 0 && !roleAllowed(roles, role) {
                httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "your role does not allow this action")
                return
            }

            ctx := context.WithValue(r.Context(), tenantIDKey{}, tenantID)
            ctx = context.WithValue(ctx, roleKey{}, role)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func roleAllowed(allowed []string, role string) bool {
    for _, a := range allowed {
        if a == role {
            return true
        }
    }
    return false
}

// TenantIDFrom extracts the tenant ID set by RequireMembership.
func TenantIDFrom(ctx context.Context) (uuid.UUID, bool) {
    id, ok := ctx.Value(tenantIDKey{}).(uuid.UUID)
    return id, ok
}

// RoleFrom extracts the caller's role set by RequireMembership.
func RoleFrom(ctx context.Context) (string, bool) {
    role, ok := ctx.Value(roleKey{}).(string)
    return role, ok
}
