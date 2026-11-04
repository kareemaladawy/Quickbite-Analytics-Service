// Package middleware holds stdlib-http middleware — the Go analogue of
// order-service's lib/correlation/correlationId.ts and its access-log
// middleware.
package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"quickbite/analytics-service/lib/appcontext"
	"quickbite/analytics-service/lib/logger"
)

const correlationHeader = "X-Correlation-Id"

// Correlation reads X-Correlation-Id (falling back to a fresh uuid), stashes
// it on the request context, and echoes it back on the response — mirrors
// correlationId.ts exactly, including accepting either header casing since
// net/http canonicalizes headers for us.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(correlationHeader)
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set(correlationHeader, id)
		ctx := appcontext.WithCorrelationID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AccessLog logs one structured line per request — method, path, status,
// duration — the Go analogue of order-service's access-log middleware.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		logger.FromContext(r.Context()).Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
