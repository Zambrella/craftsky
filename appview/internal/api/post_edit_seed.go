package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

var ErrPostEditPending = errors.New("post: edit projection pending")

// PostEditSnapshot is server-only. The raw source is never sent to an editor.
type PostEditSnapshot struct {
	Post     *PostRow
	Record   json.RawMessage
	Revision syntax.TID
}

// ReadEditSnapshot reads the source and its eligible presentation in one
// repeatable-read snapshot. An ahead-of-projection source cannot seed an older form.
func (s *PostStore) ReadEditSnapshot(ctx context.Context, owner syntax.DID, key syntax.RecordKey) (*PostEditSnapshot, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("post edit snapshot begin: %w", err)
	}
	defer tx.Rollback(ctx)
	q := `SELECT ` + postSelectColumns + `, source.record, source.revision,
		(source.cid = p.cid AND source.ordering_status = 'authoritative'
		 AND source.structural_validation_status = 'valid' AND source.semantic_validation_status = 'valid'
		 AND source.projection_disposition = 'eligible'
		 AND source.projection_generation = lifecycle.generation)
		FROM craftsky_posts p
		JOIN craftsky_profiles member ON member.did = p.did
		JOIN owner_lifecycles lifecycle ON lifecycle.owner_did = p.did AND lifecycle.state = 'active'
		JOIN tap_source_records source ON source.uri = p.uri AND source.action <> 'delete'
		LEFT JOIN craftsky_project_posts pp ON pp.uri = p.uri
		LEFT JOIN bluesky_profiles bp ON bp.did = p.did
		WHERE p.did = $1 AND p.rkey = $2 ` + postVisibleModerationPredicate
	out := &PostEditSnapshot{}
	var ready *bool
	row, err := scanPostRowWithExtra(tx.QueryRow(ctx, q, owner, key), &out.Record, &out.Revision, &ready)
	if errors.Is(err, pgx.ErrNoRows) {
		var awaitingInitial bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1 FROM tap_source_records source
			JOIN craftsky_profiles member ON member.did = source.did
			JOIN owner_lifecycles lifecycle ON lifecycle.owner_did = source.did AND lifecycle.state = 'active'
			WHERE source.did = $1 AND source.rkey = $2 AND source.collection = 'social.craftsky.feed.post'
			 AND source.action <> 'delete' AND source.projection_disposition <> 'denied_terminal'
			 AND NOT EXISTS (SELECT 1 FROM craftsky_posts p WHERE p.uri = source.uri)
		)`, owner, key).Scan(&awaitingInitial); err != nil {
			return nil, fmt.Errorf("post edit initial projection: %w", err)
		}
		if awaitingInitial {
			return nil, ErrPostEditPending
		}
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("post edit snapshot: %w", err)
	}
	if ready == nil || !*ready {
		return nil, ErrPostEditPending
	}
	out.Post = row
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("post edit snapshot commit: %w", err)
	}
	return out, nil
}

type postEditSeedStore interface {
	readPostStore
	ReadEditSnapshot(context.Context, syntax.DID, syntax.RecordKey) (*PostEditSnapshot, error)
}

// GetPostEditHandler serves the private owner-only editor seed.
func GetPostEditHandler(store postEditSeedStore, resolver HandleResolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		runID := middleware.GetRunID(r.Context())
		owner, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, 400, "invalid_identifier", "invalid owner", runID, nil)
			return
		}
		caller, ok := middleware.GetDID(r.Context())
		if !ok || caller != owner {
			envelope.WriteError(w, 403, "forbidden", "cannot edit this post", runID, nil)
			return
		}
		key, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			envelope.WriteError(w, 400, "invalid_identifier", "invalid record key", runID, nil)
			return
		}
		snapshot, err := store.ReadEditSnapshot(r.Context(), owner, key)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, 404, "post_not_found", "post not found", runID, nil)
			return
		}
		if errors.Is(err, ErrPostEditPending) {
			w.Header().Set("Retry-After", "1")
			envelope.WriteError(w, 503, "post_edit_pending", "post is still being indexed", runID, nil)
			return
		}
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "query")
			envelope.WriteError(w, 500, "internal_error", "post read failed", runID, nil)
			return
		}
		record, err := postEditEditable(snapshot.Record)
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "decode")
			envelope.WriteError(w, 500, "internal_error", "post read failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), owner)
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "identity")
			envelope.WriteError(w, 502, "identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		post := buildPostResponse(snapshot.Post, handle, store)
		summaries, err := store.EngagementSummaries(r.Context(), owner.String(), nil, []string{snapshot.Post.URI})
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "engagement")
			envelope.WriteError(w, 500, "internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		applyEngagementSummary(post, summaries[snapshot.Post.URI])
		if err := attachQuoteView(r.Context(), store, resolver, post); err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "quote")
			envelope.WriteError(w, 500, "internal_error", "post quote lookup failed", runID, nil)
			return
		}
		writeJSON(w, 200, struct {
			Post           *PostResponse `json:"post"`
			BaseCID        string        `json:"baseCid"`
			RecordRevision syntax.TID    `json:"recordRevision"`
			Editable       any           `json:"editable"`
		}{post, snapshot.Post.CID, snapshot.Revision, record})
	})
}
