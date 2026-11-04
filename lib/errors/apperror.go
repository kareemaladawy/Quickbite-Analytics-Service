// Package errors is the Go analogue of order-service's lib/error/AppError.ts,
// extended with a stable `code` field so the response envelope can carry
// {code, message} as the brief requires (order-service's own envelope is
// {error, details} without a code — this service's contract is the richer
// one, matching docs/api-contracts.md).
package errors

import "net/http"

type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Details    any
}

func (e *AppError) Error() string {
	return e.Message
}

// New constructs a named, stable AppError — the Go analogue of a module-level
// `export const SomeError = new AppError(...)` in a Node errors.ts. Call sites
// never construct ad-hoc AppErrors for cases that have a stable name.
func New(code, message string, statusCode int) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

// WithDetails returns a copy carrying extra structured detail (e.g. validator
// field errors) — used for VALIDATION_ERROR, which is dynamic per-request.
func (e *AppError) WithDetails(details any) *AppError {
	return &AppError{Code: e.Code, Message: e.Message, StatusCode: e.StatusCode, Details: details}
}

// Internal wraps an unexpected error (the Go analogue of errorHandler.ts's
// `new AppError(err.message, 500, false)` fallback branch) — the message is
// never leaked to the client; only logged.
func Internal(err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "Something went wrong",
		StatusCode: http.StatusInternalServerError,
		Details:    err.Error(),
	}
}
