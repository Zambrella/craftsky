package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/postpublication"
	"social.craftsky/appview/internal/tap"
)

// ProjectNotifications retries notification dependencies independently of the
// serving post. The worker owns the transaction and source-version fence.
func (c *CraftskyPost) ProjectNotifications(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	if source.Action == "delete" {
		return appliedAfter(c.lifecycle.Retract(ctx, tx, notifications.Retraction{SourceURI: source.URI, Reason: "sourceDeleted"}))
	}
	var rec craftskylex.FeedPost
	if err := json.Unmarshal(source.Record, &rec); err != nil {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	if recognizedExternalImportSource(&rec) != "" {
		return tap.Applied(), nil
	}
	// A newer source can be claimed before its serving projection runs.
	var current bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_posts WHERE uri=$1 AND cid=$2)`, source.URI, source.CID).Scan(&current); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	if !current {
		return tap.Blocked(tap.ReasonMissingSubject, tap.Dependency{Kind: "subject_uri", Key: source.URI.String()}), nil
	}
	original, err := postpublication.Read(ctx, tx, source.URI)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	if !original.Known || original.CreatedAt == nil {
		return tap.Applied(), nil
	}
	// Notification reasons belong to original publication, not edited content.
	rec.Reply = nil
	if original.ParentURI != "" {
		rec.Reply = &craftskylex.FeedPost_ReplyRef{Parent: &comatproto.RepoStrongRef{Uri: original.ParentURI, Cid: original.ParentCID}, Root: &comatproto.RepoStrongRef{Uri: original.RootURI, Cid: original.RootCID}}
	}
	rec.Embed = nil
	if original.QuotedURI != "" {
		rec.Embed = &craftskylex.FeedPost_Embed{FeedPost_QuoteEmbed: &craftskylex.FeedPost_QuoteEmbed{Record: &comatproto.RepoStrongRef{Uri: original.QuotedURI, Cid: original.QuotedCID}}}
	}
	var references []string
	if rec.Reply != nil && rec.Reply.Parent != nil {
		references = append(references, rec.Reply.Parent.Uri)
	}
	if rec.Embed != nil && rec.Embed.FeedPost_QuoteEmbed != nil && rec.Embed.FeedPost_QuoteEmbed.Record != nil {
		references = append(references, rec.Embed.FeedPost_QuoteEmbed.Record.Uri)
	}
	var missingMember syntax.DID
	for _, uri := range references {
		var member, terminal bool
		var did syntax.DID
		err := tx.QueryRow(ctx, `
			SELECT p.did, EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=p.did), appview_owner_is_terminal(p.did)
			FROM craftsky_posts p WHERE uri=$1
		`, uri).Scan(&did, &member, &terminal)
		if errors.Is(err, pgx.ErrNoRows) {
			return tap.Blocked(tap.ReasonMissingSubject, tap.Dependency{Kind: "subject_uri", Key: uri}), nil
		}
		if err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
		if !member && !terminal {
			missingMember = did
		}
	}
	mentions, err := postpublication.OriginalMentions(ctx, tx, source.URI)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	for _, did := range mentions {
		if did == source.DID.String() {
			continue
		}
		var member, terminal bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1),appview_owner_is_terminal($1)`, did).Scan(&member, &terminal); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
		if !member && !terminal && missingMember == "" {
			missingMember = syntax.DID(did)
		}
	}
	createdAt := *original.CreatedAt
	if err := c.activatePostNotifications(ctx, tx, eventFromSource(source), rec, mentions, createdAt); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("project post notifications: %w", err)
	}
	// Available recipients are notified even if another recipient is not yet
	// a member. Replays use the notification service's duplicate protection.
	if missingMember != "" {
		return tap.Blocked(tap.ReasonMissingMember, tap.Dependency{Kind: "member_did", Key: missingMember.String()}), nil
	}
	return tap.Applied(), nil
}
