package tenant

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    "linkpulse/internal/httpx"
)

// ErrLastOwner protects the workspace from losing its only owner.
var ErrLastOwner = &httpx.UserError{
    Status:  http.StatusConflict,
    Code:    httpx.CodeConflict,
    Message: "the last owner cannot be demoted, removed, or leave — transfer ownership first",
}

// MemberOut is a workspace member as seen in the roster.
type MemberOut struct {
    UserID   uuid.UUID `json:"user_id"`
    Name     string    `json:"name"`
    Email    string    `json:"email"`
    Role     string    `json:"role"`
    JoinedAt time.Time `json:"joined_at"`
}

// UpdateRoleInput is the request body for PATCH .../members/{userId}.
type UpdateRoleInput struct {
    Role string `json:"role"`
}

// assignableRoles: every role a member can hold.
var assignableRoles = map[string]bool{
    RoleOwner:  true,
    RoleAdmin:  true,
    RoleMember: true,
    RoleViewer: true,
}

// ListMembers returns the workspace roster (PRD 9.3.3). Any current
// member may view it.
func (s *Service) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]MemberOut, error) {
    rows, err := s.db.Query(ctx, `
        SELECT u.id, u.name, u.email, m.role, m.created_at
        FROM memberships m
        JOIN users u ON u.id = m.user_id
        WHERE m.tenant_id = $1
        ORDER BY m.created_at ASC`, tenantID)
    if err != nil {
        return nil, fmt.Errorf("load members: %w", err)
    }
    defer rows.Close()

    members := []MemberOut{}
    for rows.Next() {
        var m MemberOut
        if err := rows.Scan(&m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt); err != nil {
            return nil, fmt.Errorf("scan member: %w", err)
        }
        members = append(members, m)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate members: %w", err)
    }
    return members, nil
}

// UpdateMemberRole changes a member's role (PRD 9.3.4). You cannot
// change your own role, and the last owner can never be demoted.
func (s *Service) UpdateMemberRole(ctx context.Context, actorID, tenantID, targetID uuid.UUID, newRole string) (MemberOut, error) {
    newRole = strings.TrimSpace(newRole)
    if !assignableRoles[newRole] {
        return MemberOut{}, &httpx.UserError{
            Status:  http.StatusUnprocessableEntity,
            Code:    httpx.CodeValidationError,
            Message: "role must be one of: owner, admin, member, viewer",
        }
    }
    if actorID == targetID {
        return MemberOut{}, &httpx.UserError{
            Status:  http.StatusConflict,
            Code:    httpx.CodeConflict,
            Message: "you cannot change your own role",
        }
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return MemberOut{}, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    var currentRole string
    err = tx.QueryRow(ctx,
        `SELECT role FROM memberships WHERE tenant_id = $1 AND user_id = $2 FOR UPDATE`,
        tenantID, targetID).Scan(&currentRole)
    if errors.Is(err, pgx.ErrNoRows) {
        return MemberOut{}, &httpx.UserError{
            Status:  http.StatusNotFound,
            Code:    httpx.CodeNotFound,
            Message: "member not found",
        }
    }
    if err != nil {
        return MemberOut{}, fmt.Errorf("load member: %w", err)
    }

    if currentRole == RoleOwner && newRole != RoleOwner {
        owners, err := countOwners(ctx, tx, tenantID)
        if err != nil {
            return MemberOut{}, fmt.Errorf("count owners: %w", err)
        }
        if owners <= 1 {
            return MemberOut{}, ErrLastOwner
        }
    }

    _, err = tx.Exec(ctx,
        `UPDATE memberships SET role = $1 WHERE tenant_id = $2 AND user_id = $3`,
        newRole, tenantID, targetID)
    if err != nil {
        return MemberOut{}, fmt.Errorf("update role: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return MemberOut{}, fmt.Errorf("commit: %w", err)
    }

    return s.member(ctx, tenantID, targetID)
}

// RemoveMember removes another member from the workspace (PRD 9.3.5).
// The last owner can never be removed. Leaving is a separate endpoint.
func (s *Service) RemoveMember(ctx context.Context, actorID, tenantID, targetID uuid.UUID) error {
    if actorID == targetID {
        return &httpx.UserError{
            Status:  http.StatusBadRequest,
            Code:    httpx.CodeBadRequest,
            Message: "use the leave endpoint to remove yourself",
        }
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    var currentRole string
    err = tx.QueryRow(ctx,
        `SELECT role FROM memberships WHERE tenant_id = $1 AND user_id = $2 FOR UPDATE`,
        tenantID, targetID).Scan(&currentRole)
    if errors.Is(err, pgx.ErrNoRows) {
        return &httpx.UserError{
            Status:  http.StatusNotFound,
            Code:    httpx.CodeNotFound,
            Message: "member not found",
        }
    }
    if err != nil {
        return fmt.Errorf("load member: %w", err)
    }

    if currentRole == RoleOwner {
        owners, err := countOwners(ctx, tx, tenantID)
        if err != nil {
            return fmt.Errorf("count owners: %w", err)
        }
        if owners <= 1 {
            return ErrLastOwner
        }
    }

    _, err = tx.Exec(ctx,
        `DELETE FROM memberships WHERE tenant_id = $1 AND user_id = $2`,
        tenantID, targetID)
    if err != nil {
        return fmt.Errorf("delete membership: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit: %w", err)
    }
    return nil
}

// Leave removes the caller's own membership (PRD 9.3.5). The last owner
// must transfer ownership first.
func (s *Service) Leave(ctx context.Context, userID, tenantID uuid.UUID) error {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    var role string
    err = tx.QueryRow(ctx,
        `SELECT role FROM memberships WHERE tenant_id = $1 AND user_id = $2 FOR UPDATE`,
        tenantID, userID).Scan(&role)
    if errors.Is(err, pgx.ErrNoRows) {
        return &httpx.UserError{
            Status:  http.StatusForbidden,
            Code:    httpx.CodeForbidden,
            Message: "you are not a member of this workspace",
        }
    }
    if err != nil {
        return fmt.Errorf("load membership: %w", err)
    }

    if role == RoleOwner {
        owners, err := countOwners(ctx, tx, tenantID)
        if err != nil {
            return fmt.Errorf("count owners: %w", err)
        }
        if owners <= 1 {
            return ErrLastOwner
        }
    }

    _, err = tx.Exec(ctx,
        `DELETE FROM memberships WHERE tenant_id = $1 AND user_id = $2`,
        tenantID, userID)
    if err != nil {
        return fmt.Errorf("delete membership: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit: %w", err)
    }
    return nil
}

// member loads a single roster row for response building.
func (s *Service) member(ctx context.Context, tenantID, userID uuid.UUID) (MemberOut, error) {
    var m MemberOut
    err := s.db.QueryRow(ctx, `
        SELECT u.id, u.name, u.email, m.role, m.created_at
        FROM memberships m
        JOIN users u ON u.id = m.user_id
        WHERE m.tenant_id = $1 AND m.user_id = $2`, tenantID, userID,
    ).Scan(&m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt)
    if err != nil {
        return MemberOut{}, fmt.Errorf("load member: %w", err)
    }
    return m, nil
}

func countOwners(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (int, error) {
    var count int
    err := tx.QueryRow(ctx,
        `SELECT COUNT(*) FROM memberships WHERE tenant_id = $1 AND role = 'owner'`,
        tenantID).Scan(&count)
    return count, err
}

// ListMembers handles GET /api/v1/tenants/{tenantId}/members.
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
    tenantID, ok := TenantIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    res, err := h.svc.ListMembers(r.Context(), tenantID)
    if err != nil {
        h.handleErr(w, err, "member list")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// UpdateMemberRole handles PATCH /api/v1/tenants/{tenantId}/members/{userId}
// (owner/admin only; only an owner may grant the owner role).
func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
    actorRole, ok := RoleFrom(r.Context())
    if !ok || (actorRole != RoleOwner && actorRole != RoleAdmin) {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "requires owner or admin role")
        return
    }

    targetID, err := uuid.Parse(chi.URLParam(r, "userId"))
    if err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid user id")
        return
    }

    actorID, okU := httpx.UserIDFrom(r.Context())
    tenantID, okT := TenantIDFrom(r.Context())
    if !okU || !okT {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in UpdateRoleInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    // Only an owner can grant the owner role (least privilege).
    if in.Role == RoleOwner && actorRole != RoleOwner {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "only an owner can grant the owner role")
        return
    }

    res, err := h.svc.UpdateMemberRole(r.Context(), actorID, tenantID, targetID, in.Role)
    if err != nil {
        h.handleErr(w, err, "member role update")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}

// RemoveMember handles DELETE /api/v1/tenants/{tenantId}/members/{userId}
// (owner/admin only).
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
    actorRole, ok := RoleFrom(r.Context())
    if !ok || (actorRole != RoleOwner && actorRole != RoleAdmin) {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "requires owner or admin role")
        return
    }

    targetID, err := uuid.Parse(chi.URLParam(r, "userId"))
    if err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid user id")
        return
    }

    actorID, okU := httpx.UserIDFrom(r.Context())
    tenantID, okT := TenantIDFrom(r.Context())
    if !okU || !okT {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    if err := h.svc.RemoveMember(r.Context(), actorID, tenantID, targetID); err != nil {
        h.handleErr(w, err, "member remove")
        return
    }
    httpx.Success(w, http.StatusOK, map[string]string{"message": "member removed"})
}

// Leave handles POST /api/v1/tenants/{tenantId}/leave.
func (h *Handler) Leave(w http.ResponseWriter, r *http.Request) {
    userID, okU := httpx.UserIDFrom(r.Context())
    tenantID, okT := TenantIDFrom(r.Context())
    if !okU || !okT {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    if err := h.svc.Leave(r.Context(), userID, tenantID); err != nil {
        h.handleErr(w, err, "member leave")
        return
    }
    httpx.Success(w, http.StatusOK, map[string]string{"message": "you left the workspace"})
}
