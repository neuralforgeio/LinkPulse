// Package httpx contains shared HTTP helpers: the standard JSON response
// envelope, error codes, and the UserError contract used by every API
// handler (PRD section 15).
package httpx

import (
	"encoding/json"
	"net/http"
)

// Error codes (PRD section 15).
const (
    CodeValidationError = "VALIDATION_ERROR"
    CodeBadRequest      = "BAD_REQUEST"
    CodeConflict        = "CONFLICT"
    CodeForbidden       = "FORBIDDEN"
    CodeNotFound        = "NOT_FOUND"
    CodeUnauthorized    = "UNAUTHORIZED"
    CodeInvalidToken    = "INVALID_TOKEN"
    CodeInviteExpired   = "INVITE_EXPIRED"
    CodeRateLimited     = "RATE_LIMITED"
    CodeInternalError   = "INTERNAL_ERROR"
)

// UserError is an error caused by the request — bad input, a conflict,
// or a missing permission — that maps to a 4xx response, never a 500.
// Every module's service layer returns these; handlers translate them.
type UserError struct {
    Status  int
    Code    string
    Message string
}

func (e *UserError) Error() string { return e.Message }

// Success writes {"success":true,"data":...} with the given status code.
func Success(w http.ResponseWriter, status int, data any) {
    write(w, status, envelope{Success: true, Data: data})
}

// Error writes {"success":false,"error":{...}} with the given status code.
// details are optional human-readable field problems.
func Error(w http.ResponseWriter, status int, code, message string, details ...string) {
    if details == nil {
        details = []string{}
    }
    write(w, status, envelope{Success: false, Err: &errBody{
        Code:    code,
        Message: message,
        Details: details,
    }})
}

func write(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}

type envelope struct {
    Success bool     `json:"success"`
    Data    any      `json:"data,omitempty"`
    Err     *errBody `json:"error,omitempty"`
}

type errBody struct {
    Code    string   `json:"code"`
    Message string   `json:"message"`
    Details []string `json:"details"`
}
