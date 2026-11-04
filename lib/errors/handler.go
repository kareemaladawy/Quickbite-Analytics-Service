package errors

import (
	"encoding/json"
	"errors"
	"net/http"

	"quickbite/analytics-service/lib/logger"
)

// HandlerFunc is the Go analogue of an Express controller method that throws
// instead of calling next(err) — it returns error instead of writing one
// itself. Wrap is the Go analogue of order-service's errorHandler.ts
// middleware: the one place an AppError (or any error) becomes a response.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

type errorEnvelope struct {
	Success bool      `json:"success"`
	Error   errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Wrap renders any error HandlerFunc returns as the service's error
// envelope: {"success": false, "error": {"code", "message"}}.
func Wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		var appErr *AppError
		if !errors.As(err, &appErr) {
			appErr = Internal(err)
		}

		logger.FromContext(r.Context()).Error(appErr.Message,
			"code", appErr.Code,
			"status", appErr.StatusCode,
			"path", r.URL.Path,
			"method", r.Method,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		_ = json.NewEncoder(w).Encode(errorEnvelope{
			Success: false,
			Error: errorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
				Details: appErr.Details,
			},
		})
	}
}
