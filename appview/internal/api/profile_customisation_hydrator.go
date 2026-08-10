package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type ProfileCustomisationBatchReader interface {
	ReadBatch(context.Context, []syntax.DID) (map[syntax.DID]ProfileCustomisation, error)
}

type IdentityCustomisationHydrator struct {
	reader ProfileCustomisationBatchReader
}

func NewIdentityCustomisationHydrator(reader ProfileCustomisationBatchReader) *IdentityCustomisationHydrator {
	return &IdentityCustomisationHydrator{reader: reader}
}

func (h *IdentityCustomisationHydrator) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured := &profileCustomisationResponseCapture{header: make(http.Header)}
		next.ServeHTTP(captured, r)
		status := captured.status
		if status == 0 {
			status = http.StatusOK
		}
		body := captured.body.Bytes()
		if status >= 200 && status < 300 && strings.Contains(captured.header.Get("Content-Type"), "application/json") {
			hydrated, err := h.HydrateJSON(r.Context(), body)
			if err != nil {
				envelope.WriteError(
					w,
					http.StatusInternalServerError,
					"internal_error",
					"profile customisation hydration failed",
					middleware.GetRunID(r.Context()),
					nil,
				)
				return
			}
			body = hydrated
		}
		copyProfileCustomisationHeaders(w.Header(), captured.header)
		w.Header().Del("Content-Length")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}

type profileCustomisationResponseCapture struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *profileCustomisationResponseCapture) Header() http.Header {
	return w.header
}

func (w *profileCustomisationResponseCapture) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *profileCustomisationResponseCapture) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}

func copyProfileCustomisationHeaders(destination, source http.Header) {
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}

// HydrateJSON decorates every retained identity object in a response using a
// single deduplicated member lookup. Identity objects are recognized by their
// existing did+handle contract; stripped placeholders and actorless responses
// therefore remain untouched.
func (h *IdentityCustomisationHydrator) HydrateJSON(
	ctx context.Context,
	raw []byte,
) ([]byte, error) {
	if h == nil || h.reader == nil || len(bytes.TrimSpace(raw)) == 0 {
		return raw, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("profile customisation response decode: %w", err)
	}

	identities := make(map[syntax.DID][]map[string]any)
	collectResponseIdentities(root, identities)
	if len(identities) == 0 {
		return raw, nil
	}
	dids := make([]syntax.DID, 0, len(identities))
	for did := range identities {
		dids = append(dids, did)
	}
	values, err := h.reader.ReadBatch(ctx, dids)
	if err != nil {
		return nil, err
	}
	for did, value := range values {
		for _, identity := range identities[did] {
			identity["customisation"] = value
		}
	}
	hydrated, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("profile customisation response encode: %w", err)
	}
	return hydrated, nil
}

func collectResponseIdentities(value any, identities map[syntax.DID][]map[string]any) {
	switch value := value.(type) {
	case map[string]any:
		rawDID, hasDID := value["did"].(string)
		handle, hasHandle := value["handle"].(string)
		if hasDID && hasHandle && handle != "" {
			if did, err := syntax.ParseDID(rawDID); err == nil {
				identities[did] = append(identities[did], value)
			}
		}
		for _, child := range value {
			collectResponseIdentities(child, identities)
		}
	case []any:
		for _, child := range value {
			collectResponseIdentities(child, identities)
		}
	}
}
