package httpx

import (
	"encoding/json"
	"net/http"
)

const (
	CodeValidationError = "VALIDATION_ERROR"
	CodeBaqRequest			= "BAD_REQUEST"
	CodeConflict				= "CONFLICT"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeInvalidToken		= "INVALID_TOKEN"
  CodeInternalError   = "INTERNAL_ERROR"
)

// Succes writes {"success": true, "data":...} with the given status code
func Success(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{Success:true, Data:data })
}

func Error(w http.ResponseWriter, status int, code, message string, details ...string) {
	if details == nil {
		details = []string{}
	}
	write(w, status, envelope{Success:false, Err: &errBody{
		Code: code,
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
	Success bool `json:"success"`
	Data any `json:"data,omitempty"`
	Err *errBody `json:"error,omitempty"`
}

type errBody struct {
	Code string `json:"code"`
	Message string `json:"message"`
	Details []string `json:"details"`
}
