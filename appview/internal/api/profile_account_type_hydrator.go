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
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
)

type AccountTypeBatchReader interface {
	ReadAccountTypes(context.Context, []syntax.DID) (map[syntax.DID]business.AccountType, error)
}

type IdentityAccountTypeHydrator struct {
	reader AccountTypeBatchReader
}

func NewIdentityAccountTypeHydrator(reader AccountTypeBatchReader) *IdentityAccountTypeHydrator {
	return &IdentityAccountTypeHydrator{reader: reader}
}

func (h *IdentityAccountTypeHydrator) Handler(next http.Handler) http.Handler {
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
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "account type hydration failed", middleware.GetRunID(r.Context()), nil)
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

func (h *IdentityAccountTypeHydrator) HydrateJSON(ctx context.Context, raw []byte) ([]byte, error) {
	if h == nil || h.reader == nil || len(bytes.TrimSpace(raw)) == 0 {
		return raw, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("account type response decode: %w", err)
	}
	identities := make(map[syntax.DID][]map[string]any)
	collectAccountTypeIdentities(root, identities)
	if len(identities) == 0 {
		return raw, nil
	}
	dids := make([]syntax.DID, 0, len(identities))
	for did := range identities {
		dids = append(dids, did)
	}
	values, err := h.reader.ReadAccountTypes(ctx, dids)
	if err != nil {
		return nil, err
	}
	for did, objects := range identities {
		accountType := business.AccountTypeRegular
		if stored, ok := values[did]; ok {
			accountType = stored
		}
		for _, identity := range objects {
			identity["accountType"] = accountType
		}
	}
	hydrated, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("account type response encode: %w", err)
	}
	return hydrated, nil
}

func collectAccountTypeIdentities(value any, identities map[syntax.DID][]map[string]any) {
	switch value := value.(type) {
	case map[string]any:
		blocked, _ := value["blocking"].(bool)
		blockedBy, _ := value["blockedBy"].(bool)
		available, hasAvailable := value["available"].(bool)
		rawDID, hasDID := value["did"].(string)
		handle, hasHandle := value["handle"].(string)
		if hasDID && hasHandle && handle != "" {
			if blocked || blockedBy || (hasAvailable && !available) {
				delete(value, "accountType")
				delete(value, "business")
			} else {
				if did, err := syntax.ParseDID(rawDID); err == nil {
					identities[did] = append(identities[did], value)
				}
			}
		}
		for _, child := range value {
			collectAccountTypeIdentities(child, identities)
		}
	case []any:
		for _, child := range value {
			collectAccountTypeIdentities(child, identities)
		}
	}
}
