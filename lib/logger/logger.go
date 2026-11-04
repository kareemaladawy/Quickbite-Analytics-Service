// Package logger wraps stdlib log/slog — the Go analogue of order-service's
// lib/logger/logger.ts Logger class, structured JSON out, with a
// FromContext helper so handlers can log with correlation_id attached
// without threading a logger through every call site.
package logger

import (
	"context"
	"log/slog"
	"os"

	"quickbite/analytics-service/lib/appcontext"
)

var base *slog.Logger

func init() {
	base = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// Base returns the process-wide logger with no request context attached —
// used at boot, before any request exists.
func Base() *slog.Logger {
	return base
}

// FromContext returns a logger with correlation_id attached, if present on
// ctx, the Go analogue of passing req.correlationId into every log call.
func FromContext(ctx context.Context) *slog.Logger {
	correlationID := appcontext.CorrelationID(ctx)
	if correlationID == "" {
		return base
	}
	return base.With("correlation_id", correlationID)
}
