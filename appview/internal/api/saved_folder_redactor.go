package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/subscriptions"
)

type folderResponseRedactor struct {
	access interface {
		SelfAccess(context.Context, syntax.DID, time.Time) (subscriptions.SelfAccess, error)
	}
}

// SavedFolderRedactor removes private folder identifiers from embedded post
// projections while leaving the underlying saved-post associations untouched.
func SavedFolderRedactor(access interface {
	SelfAccess(context.Context, syntax.DID, time.Time) (subscriptions.SelfAccess, error)
}) *folderResponseRedactor {
	return &folderResponseRedactor{access: access}
}

func (h *folderResponseRedactor) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture := &profileCustomisationResponseCapture{header: make(http.Header)}
		next.ServeHTTP(capture, r)
		status := capture.status
		if status == 0 {
			status = http.StatusOK
		}
		body := capture.body.Bytes()
		if status >= 200 && status < 300 && strings.Contains(capture.header.Get("Content-Type"), "application/json") && bytes.Contains(body, []byte(`"viewerSavedFolderId"`)) {
			did, ok := middleware.GetDID(r.Context())
			if !ok {
				envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
				return
			}
			access, err := h.access.SelfAccess(r.Context(), did, time.Now())
			if err != nil {
				envelope.WriteError(w, http.StatusServiceUnavailable, "subscription_unavailable", "subscription access unavailable", middleware.GetRunID(r.Context()), nil)
				return
			}
			if !access.AllowsPlus() {
				decoder := json.NewDecoder(bytes.NewReader(body))
				decoder.UseNumber()
				var root any
				if err := decoder.Decode(&root); err != nil {
					envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "post response unavailable", middleware.GetRunID(r.Context()), nil)
					return
				}
				redactViewerSavedFolderIDs(root)
				body, err = json.Marshal(root)
				if err != nil {
					envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "post response unavailable", middleware.GetRunID(r.Context()), nil)
					return
				}
			}
		}
		copyProfileCustomisationHeaders(w.Header(), capture.header)
		w.Header().Del("Content-Length")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
}

func redactViewerSavedFolderIDs(value any) {
	switch value := value.(type) {
	case map[string]any:
		if _, present := value["viewerSavedFolderId"]; present {
			value["viewerSavedFolderId"] = nil
		}
		for _, child := range value {
			redactViewerSavedFolderIDs(child)
		}
	case []any:
		for _, child := range value {
			redactViewerSavedFolderIDs(child)
		}
	}
}
