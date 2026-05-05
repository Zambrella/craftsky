// appview/internal/api/post.go
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/postutil"
)

const craftskyPostNSID = "social.craftsky.feed.post"

// CreatePostHandler serves POST /v1/posts.
func CreatePostHandler(
	store PostReader,
	newPDS auth.PDSClientFactory,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())

		req, err := DecodePostCreate(r.Body)
		if err != nil {
			fe, isFE := err.(*FieldError)
			switch {
			case isFE && fe.Code == "malformed_body":
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, fe.Fields)
			case isFE:
				envelope.WriteError(w, http.StatusBadRequest,
					fe.Code, "request body rejected", runID, fe.Fields)
			default:
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, nil)
			}
			return
		}
		if err := ValidatePostCreate(req); err != nil {
			fe, isFE := err.(*FieldError)
			if isFE {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}

		body := lexiconRecordBody(req)

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), did, craftskyPostNSID, body)
		if err != nil {
			logger.Warn("post: CreateRecord failed",
				slog.String("did", did.String()), slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "PDS rejected the post", runID, nil)
			return
		}

		row, err := syntheticPostRow(r, store, did, uri, cid, req)
		if err != nil {
			logger.Error("post: hydrate author failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post created but hydrate failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// lexiconRecordBody translates the wire request into the lexicon-shaped
// record body that goes to the PDS. Facets are pass-through raw JSON so
// the PDS sees exactly what the client sent (including any "$type"
// discriminators on union variants).
func lexiconRecordBody(req PostCreateRequest) map[string]any {
	body := map[string]any{
		"$type":     craftskyPostNSID,
		"text":      req.Text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}
	if len(req.Facets) > 0 {
		body["facets"] = req.Facets
	}
	if req.Reply != nil {
		body["reply"] = map[string]any{
			"root":   map[string]any{"uri": req.Reply.Root.URI, "cid": req.Reply.Root.CID},
			"parent": map[string]any{"uri": req.Reply.Parent.URI, "cid": req.Reply.Parent.CID},
		}
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		body["embed"] = map[string]any{
			"$type": craftskyPostNSID + "#quoteEmbed",
			"record": map[string]any{
				"uri": req.Embed.Quote.URI,
				"cid": req.Embed.Quote.CID,
			},
		}
	}
	return body
}

// syntheticPostRow assembles the PostRow that BuildPostResponse needs
// from the request body, the PDS-assigned (uri, cid), and a single
// author lookup against the store. We don't wait for the firehose to
// land the row.
//
// Tags are extracted from req.Facets via a non-strict decode into the
// indigo richtext-facet typed slice. Errors on that decode produce
// empty tags (the PDS will still validate the lexicon shape) — we
// don't fail the whole request just because tag extraction couldn't
// parse facets the PDS may yet accept.
func syntheticPostRow(
	r *http.Request,
	store PostReader,
	did syntax.DID,
	uri syntax.ATURI,
	cid syntax.CID,
	req PostCreateRequest,
) (*PostRow, error) {
	now := time.Now().UTC()
	row := &PostRow{
		URI:       string(uri),
		DID:       did.String(),
		Rkey:      path.Base(string(uri)),
		CID:       string(cid),
		Text:      req.Text,
		Tags:      extractRequestTags(req.Facets),
		CreatedAt: now,
		IndexedAt: now,
	}
	if len(req.Facets) > 0 {
		row.Facets = req.Facets
	}
	if req.Reply != nil {
		row.ReplyRootURI = strPtr(req.Reply.Root.URI)
		row.ReplyRootCID = strPtr(req.Reply.Root.CID)
		row.ReplyParentURI = strPtr(req.Reply.Parent.URI)
		row.ReplyParentCID = strPtr(req.Reply.Parent.CID)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		row.QuoteURI = strPtr(req.Embed.Quote.URI)
		row.QuoteCID = strPtr(req.Embed.Quote.CID)
	}

	author, err := store.ReadAuthor(r.Context(), did.String())
	if err != nil {
		return nil, err
	}
	if author != nil {
		row.AuthorDisplayName = author.DisplayName
		row.AuthorAvatarCID = author.AvatarCID
	}
	return row, nil
}

// extractRequestTags decodes the raw facets JSON into the indigo typed
// slice (best effort) and returns the same tag set the indexer would
// produce when this record arrives via the firehose. Returns an empty
// (non-nil) slice on decode failure so the response always carries a
// valid tags array.
func extractRequestTags(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var typed []*appbsky.RichtextFacet
	if err := json.Unmarshal(raw, &typed); err != nil {
		return []string{}
	}
	return postutil.ExtractTags(typed)
}

func strPtr(s string) *string { return &s }

// GetPostHandler serves GET /v1/posts/{did}/{rkey}.
func GetPostHandler(store PostReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		row, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post: ReadOne failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				slog.String("did", did.String()),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// DeletePostHandler serves DELETE /v1/posts/{did}/{rkey}. Idempotent —
// returns 204 even if the underlying record was already gone.
func DeletePostHandler(newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		if did != caller {
			envelope.WriteError(w, http.StatusForbidden,
				"forbidden", "cannot delete another user's post", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		if err := pds.DeleteRecord(r.Context(), did, craftskyPostNSID, rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("post: DeleteRecord failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// ListPostsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/posts.
func ListPostsByAuthorHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("post list: ResolveDID failed",
					slog.String("input", raw),
					slog.String("err", err.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")

		rows, nextCursor, err := store.ListByAuthor(r.Context(), did.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("post list: ListByAuthor failed",
				slog.String("did", did.String()),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post list failed", runID, nil)
			return
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			// Only pay handle-resolution cost when there are rows to render.
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				logger.Warn("post list: ResolveHandle failed",
					slog.String("did", did.String()),
					slog.String("err", herr.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				items = append(items, BuildPostResponse(row, handle))
			}
		}
		body := struct {
			Items  []*PostResponse `json:"items"`
			Cursor string          `json:"cursor,omitempty"`
		}{Items: items, Cursor: nextCursor}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// parseLimit returns the validated limit, defaulting to 50 and capping
// at 100. Per pagination spec §5: caps are silent (we don't 400 on
// overshoot, we cap).
func parseLimit(raw string) int {
	const defaultLimit, maxLimit = 50, 100
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
