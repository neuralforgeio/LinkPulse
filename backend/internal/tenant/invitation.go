package tenant

// invitation.go — invite codes and the acceptance flow (PRD 9.3.1, 9.3.2).

import (
    "context"
    "crypto/rand"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    "linkpulse/internal/httpx"
)

// Invitation rules (PRD 9.3.1).
const (
    inviteCodePrefix = "INV-"
    inviteCodeLength = 8
    defaultInviteTTL = 72 * time.Hour
    maxInviteTTL     = 168 * time.Hour // one week
)

var (
    // ErrInviteNotFound: the code does not match any invitation.
    ErrInviteNotFound = &httpx.UserError{
        Status:  http.StatusNotFound,
        Code:    httpx.CodeNotFound,
        Message: "invitation not found",
    }
    // ErrInviteExpired: time is up or all uses are spent.
    ErrInviteExpired = &httpx.UserError{
        Status:  http.StatusGone,
        Code:    httpx.CodeInviteExpired,
        Message: "this invitation has expired",
    }
)

// invitableRoles: roles an invitation can grant. Owner is transfer-only
// and can never arrive through an invite.
var invitableRoles = map[string]bool{
    RoleAdmin:  true,
    RoleMember: true,
    RoleViewer: true,
}

// InviteInput is the request body for POST /api/v1/tenants/{tenantId}/invitations.
type InviteInput struct {
    Role           string `json:"role"`
    ExpiresInHours *int   `json:"expires_in_hours"`
}

// InviteResult is the response for creating an invitation.
type InviteResult struct {
    InviteCode string    `json:"invite_code"`
    InviteURL  string    `json:"invite_url"`
    Role       string    `json:"role"`
    ExpiresAt  time.Time `json:"expires_at"`
}

// AcceptInviteInput is the request body for POST /api/v1/invitations/accept.
type AcceptInviteInput struct {
    InviteCode string `json:"invite_code"`
}

// AcceptInviteResult reports what happened when accepting an invitation.
type AcceptInviteResult struct {
    Joined        bool          `json:"joined"`
    AlreadyMember bool          `json:"already_member"`
    Tenant        TenantSummary `json:"tenant"`
}

// CreateInvite issues an invitation code for this workspace (PRD 9.3.1).
func (s *Service) CreateInvite(ctx context.Context, tenantID, createdBy uuid.UUID, in InviteInput) (InviteResult, error) {
    role := strings.TrimSpace(in.Role)
    if !invitableRoles[role] {
        return InviteResult{}, &httpx.UserError{
            Status:  http.StatusUnprocessableEntity,
            Code:    httpx.CodeValidationError,
            Message: "role must be one of: admin, member, viewer",
        }
    }

    ttl := defaultInviteTTL
    if in.ExpiresInHours != nil {
        hours := *in.ExpiresInHours
        if hours < 1 || hours > int(maxInviteTTL.Hours()) {
            return InviteResult{}, &httpx.UserError{
                Status:  http.StatusUnprocessableEntity,
                Code:    httpx.CodeValidationError,
                Message: "expires_in_hours must be between 1 and 168",
            }
        }
        ttl = time.Duration(hours) * time.Hour
    }

    code, err := s.uniqueInviteCode(ctx)
    if err != nil {
        return InviteResult{}, fmt.Errorf("generate invite code: %w", err)
    }

    expiresAt := time.Now().Add(ttl)
    // max_uses = 1: single-use by default. The table supports multi-use
    // for a future without any schema change.
    _, err = s.db.Exec(ctx,
        `INSERT INTO invitations (id, tenant_id, invite_code, role, created_by, expires_at, max_uses)
         VALUES ($1, $2, $3, $4, $5, $6, 1)`,
        uuid.New(), tenantID, code, role, createdBy, expiresAt)
    if err != nil {
        return InviteResult{}, fmt.Errorf("insert invitation: %w", err)
    }

    return InviteResult{
        InviteCode: code,
        InviteURL:  "/invite/" + code,
        Role:       role,
        ExpiresAt:  expiresAt,
    }, nil
}

// uniqueInviteCode generates INV-XXXXXXXX until an unused code is found.
func (s *Service) uniqueInviteCode(ctx context.Context) (string, error) {
    for attempt := 0; attempt < 5; attempt++ {
        code, err := generateInviteCode()
        if err != nil {
            return "", err
        }
        var exists bool
        err = s.db.QueryRow(ctx,
            `SELECT EXISTS (SELECT 1 FROM invitations WHERE invite_code = $1)`, code,
        ).Scan(&exists)
        if err != nil {
            return "", fmt.Errorf("check invite code: %w", err)
        }
        if !exists {
            return code, nil
        }
    }
    return "", errors.New("could not generate a unique invite code")
}

// generateInviteCode produces INV-XXXXXXXX using only unambiguous
// characters (no I, L, O, 0, 1) so codes survive being read aloud or
// hand-copied.
func generateInviteCode() (string, error) {
    const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
    raw := make([]byte, inviteCodeLength)
    if _, err := rand.Read(raw); err != nil {
        return "", fmt.Errorf("read random bytes: %w", err)
    }
    out := make([]byte, inviteCodeLength)
    for i, v := range raw {
        out[i] = alphabet[int(v)%len(alphabet)]
    }
    return inviteCodePrefix + string(out), nil
}

// AcceptInvite joins the caller to the invitation's workspace (PRD 9.3.2).
// Re-accepting as an existing member is NOT an error — the response
// simply reports already_member: true.
func (s *Service) AcceptInvite(ctx context.Context, userID uuid.UUID, rawCode string) (AcceptInviteResult, error) {
    code := strings.ToUpper(strings.TrimSpace(rawCode))
    if code == "" {
        return AcceptInviteResult{}, &httpx.UserError{
            Status:  http.StatusUnprocessableEntity,
            Code:    httpx.CodeValidationError,
            Message: "invite_code is required",
        }
    }

    tx, err := s.db.Begin(ctx)
    if err != nil {
        return AcceptInviteResult{}, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // Lock the invitation row so limited-use codes cannot oversubscribe.
    var (
        inviteID  uuid.UUID
        tenantID  uuid.UUID
        role      string
        expiresAt time.Time
        maxUses   *int
        usedCount int
    )
    err = tx.QueryRow(ctx, `
        SELECT id, tenant_id, role, expires_at, max_uses, used_count
        FROM invitations
        WHERE invite_code = $1
        FOR UPDATE`, code,
    ).Scan(&inviteID, &tenantID, &role, &expiresAt, &maxUses, &usedCount)
    if errors.Is(err, pgx.ErrNoRows) {
        return AcceptInviteResult{}, ErrInviteNotFound
    }
    if err != nil {
        return AcceptInviteResult{}, fmt.Errorf("load invitation: %w", err)
    }

    if time.Now().After(expiresAt) {
        return AcceptInviteResult{}, ErrInviteExpired
    }

    // Already a member? Report it instead of erroring (PRD 9.3.2).
    var existingRole string
    err = tx.QueryRow(ctx,
        `SELECT role FROM memberships WHERE tenant_id = $1 AND user_id = $2`,
        tenantID, userID).Scan(&existingRole)
    if err == nil {
        tenant, terr := s.tenantSummary(ctx, tenantID, existingRole)
        if terr != nil {
            return AcceptInviteResult{}, terr
        }
        return AcceptInviteResult{Joined: false, AlreadyMember: true, Tenant: tenant}, nil
    }
    if !errors.Is(err, pgx.ErrNoRows) {
        return AcceptInviteResult{}, fmt.Errorf("check membership: %w", err)
    }

    if maxUses != nil && usedCount >= *maxUses {
        return AcceptInviteResult{}, ErrInviteExpired
    }

    _, err = tx.Exec(ctx,
        `INSERT INTO memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, $4)`,
        uuid.New(), tenantID, userID, role)
    if err != nil {
        return AcceptInviteResult{}, fmt.Errorf("insert membership: %w", err)
    }

    _, err = tx.Exec(ctx,
        `UPDATE invitations SET used_count = used_count + 1 WHERE id = $1`, inviteID)
    if err != nil {
        return AcceptInviteResult{}, fmt.Errorf("increment usage: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return AcceptInviteResult{}, fmt.Errorf("commit: %w", err)
    }

    tenant, err := s.tenantSummary(ctx, tenantID, role)
    if err != nil {
        return AcceptInviteResult{}, err
    }
    return AcceptInviteResult{Joined: true, AlreadyMember: false, Tenant: tenant}, nil
}

// tenantSummary loads id/name/slug/created_at for response building.
func (s *Service) tenantSummary(ctx context.Context, tenantID uuid.UUID, role string) (TenantSummary, error) {
    var t TenantSummary
    t.Role = role
    err := s.db.QueryRow(ctx,
        `SELECT id, name, slug, created_at FROM tenants WHERE id = $1`, tenantID,
    ).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt)
    if err != nil {
        return TenantSummary{}, fmt.Errorf("load tenant: %w", err)
    }
    return t, nil
}

// CreateInvite handles POST /api/v1/tenants/{tenantId}/invitations
// (owner/admin only — PRD 9.3 permission matrix: manage members).
func (h *Handler) CreateInvite(w http.ResponseWriter, r *http.Request) {
    role, ok := RoleFrom(r.Context())
    if !ok || (role != RoleOwner && role != RoleAdmin) {
        httpx.Error(w, http.StatusForbidden, httpx.CodeForbidden, "requires owner or admin role")
        return
    }

    tenantID, okT := TenantIDFrom(r.Context())
    userID, okU := httpx.UserIDFrom(r.Context())
    if !okT || !okU {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in InviteInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.CreateInvite(r.Context(), tenantID, userID, in)
    if err != nil {
        h.handleErr(w, err, "invite create")
        return
    }
    httpx.Success(w, http.StatusCreated, res)
}

// AcceptInvite handles POST /api/v1/invitations/accept.
func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
    userID, ok := httpx.UserIDFrom(r.Context())
    if !ok {
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "missing auth context")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in AcceptInviteInput
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    res, err := h.svc.AcceptInvite(r.Context(), userID, in.InviteCode)
    if err != nil {
        h.handleErr(w, err, "invite accept")
        return
    }
    httpx.Success(w, http.StatusOK, res)
}
