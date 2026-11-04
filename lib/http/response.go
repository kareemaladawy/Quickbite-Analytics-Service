// Package http holds response-envelope helpers — the Go analogue of
// order-service's lib/http/response.ts (SendSuccess/SendPaginated).
package http

import (
	"encoding/json"
	"net/http"
)

type successEnvelope[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
	Meta    any  `json:"meta,omitempty"`
}

func SendSuccess[T any](w http.ResponseWriter, statusCode int, data T) {
	sendJSON(w, statusCode, successEnvelope[T]{Success: true, Data: data})
}

type paginationMeta struct {
	NextCursor *string `json:"nextCursor,omitempty"`
	Total      *int    `json:"total,omitempty"`
}

func SendPaginated[T any](w http.ResponseWriter, data []T, meta paginationMeta) {
	sendJSON(w, http.StatusOK, successEnvelope[[]T]{Success: true, Data: data, Meta: meta})
}

func sendJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
