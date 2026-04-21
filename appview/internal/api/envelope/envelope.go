// Package envelope provides shared helpers for emitting the API's
// canonical JSON shapes.
//
// Every 4xx/5xx response produced by a v1 handler should go through
// WriteError. See docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §6.
package envelope

import (
	"encoding/json"
	"net/http"
)

// Error is the JSON body shape for every non-2xx response.
type Error struct {
	Error     string            `json:"error"`
	Message   string            `json:"message"`
	RequestID string            `json:"requestId"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// WriteError serialises a canonical error response to w with the given
// HTTP status code. fields may be nil; it is omitted from the JSON when
// empty.
//
// requestID should be the per-request correlation ID (in this codebase:
// middleware.GetRunID(r.Context())). Pass "" only from tests or from
// code paths that run before the Logging middleware.
func WriteError(w http.ResponseWriter, status int, code, message, requestID string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Ignore the encode error: the status header is already sent;
	// an encode failure here cannot be surfaced to the client.
	_ = json.NewEncoder(w).Encode(Error{
		Error:     code,
		Message:   message,
		RequestID: requestID,
		Fields:    fields,
	})
}
