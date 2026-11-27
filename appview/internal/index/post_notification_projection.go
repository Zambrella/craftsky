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
	"social.craftsky/appview/internal/ownerlifecycle"
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
	currentRecord := rec
	// Reply and quote reasons retain their original publication references.
	rec.Reply = nil
	if original.ParentURI != "" {
		rec.Reply = &craftskylex.FeedPost_ReplyRef{Parent: &comatproto.RepoStrongRef{Uri: original.ParentURI, Cid: original.ParentCID}, Root: &comatproto.RepoStrongRef{Uri: original.RootURI, Cid: original.RootCID}}
	}
	rec.Embed = nil
	if original.QuotedURI != "" {
		rec.Embed = &craftskylex.FeedPost_Embed{FeedPost_QuoteEmbed: &craftskylex.FeedPost_QuoteEmbed{Record: &comatproto.RepoStrongRef{Uri: original.QuotedURI, Cid: original.QuotedCID}}}
	}
	references := [...]struct {
		uri   syntax.ATURI
		reply bool
	}{{postReplyParentURI(rec), true}, {postQuotedURI(rec), false}}
	var missingMember syntax.DID
	var replyRecipient syntax.DID
	var missingSubject syntax.ATURI
	pendingRecipients := make(map[syntax.DID]bool, len(references))
	for _, reference := range references {
		uri := reference.uri
		if uri == "" {
			continue
		}
		var member, terminal bool
		var did syntax.DID
		err := tx.QueryRow(ctx, `
			SELECT p.did, EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=p.did), appview_owner_is_terminal(p.did)
			FROM craftsky_posts p WHERE uri=$1
		`, uri).Scan(&did, &member, &terminal)
		if errors.Is(err, pgx.ErrNoRows) {
			// Another reason can still be classified for a different recipient.
			// If we can't tell who owns the missing post, wait before creating
			// an event whose category the missing reference might outrank.
			target, parseErr := syntax.ParseATURI(uri.String())
			if parseErr != nil || target.Authority().DID() == "" {
				return tap.Blocked(tap.ReasonMissingSubject, tap.Dependency{Kind: "subject_uri", Key: uri.String()}), nil
			}
			did := target.Authority().DID()
			// A quote to the known reply recipient cannot change its category.
			// A self-reference cannot create another notification.
			if did == source.DID || (!reference.reply && did == replyRecipient) {
				continue
			}
			pendingRecipients[did] = true
			if missingSubject == "" {
				missingSubject = uri
			}
			continue
		}
		if err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), err
		}
		if reference.reply {
			replyRecipient = did
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
	event := eventFromSource(source)
	if original.Known && original.CreatedAt != nil {
		if err := c.activatePostNotifications(ctx, tx, event, rec, mentions, *original.CreatedAt, pendingRecipients); err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("project post notifications: %w", err)
		}
	}
	editMissingMember, err := c.projectEditMentions(ctx, tx, event, currentRecord, pendingRecipients)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("project post edit mentions: %w", err)
	}
	if missingMember == "" {
		missingMember = editMissingMember
	}
	// Available recipients are notified even if another recipient is not yet
	// a member. Replays use the notification service's duplicate protection.
	if missingSubject != "" {
		return tap.Blocked(tap.ReasonMissingSubject, tap.Dependency{Kind: "subject_uri", Key: missingSubject.String()}), nil
	}
	if missingMember != "" {
		return tap.Blocked(tap.ReasonMissingMember, tap.Dependency{Kind: "member_did", Key: missingMember.String()}), nil
	}
	return tap.Applied(), nil
}

func (c *CraftskyPost) projectEditMentions(ctx context.Context, tx pgx.Tx, event tap.Event, rec craftskylex.FeedPost, pendingRecipients map[syntax.DID]bool) (syntax.DID, error) {
	mentions, err := postpublication.PendingEditMentions(ctx, tx, event.URI)
	if err != nil || len(mentions) == 0 {
		return "", err
	}
	recipients := make([]syntax.DID, 0, len(mentions))
	for _, mention := range mentions {
		recipients = append(recipients, mention.Recipient)
	}
	// A coalesced edit may have removed a recipient from the current record,
	// so the dispatcher's locks for current mentions do not cover every alert.
	states, err := ownerlifecycle.LockOwnerStatesTx(ctx, tx, recipients)
	if err != nil {
		return "", err
	}
	var missingMember syntax.DID
	for _, mention := range mentions {
		recipient := mention.Recipient
		if pendingRecipients[recipient] {
			continue
		}
		if states[recipient].State != ownerlifecycle.StateTerminal {
			var member bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1)`, recipient).Scan(&member); err != nil {
				return "", err
			}
			if !member {
				if missingMember == "" {
					missingMember = recipient
				}
				continue
			}
			activation := postNotificationActivation(event, rec, recipient, notifications.Mention, mention.ActivityAt)
			if err := notifications.ActivatePublication(ctx, c.lifecycle, tx, activation); err != nil {
				return "", err
			}
		}
		if err := postpublication.FinishEditMention(ctx, tx, event.URI, recipient); err != nil {
			return "", err
		}
	}
	return missingMember, nil
}
