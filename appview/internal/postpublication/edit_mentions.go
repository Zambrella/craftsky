package postpublication

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// AddedMentions compares an accepted live edit with the previous authoritative
// source. A first sighting or a backfill cannot prove that a mention was added.
func AddedMentions(ctx context.Context, tx Reader, uri syntax.ATURI, record json.RawMessage) ([]syntax.DID, error) {
	var current craftskylex.FeedPost
	if err := json.Unmarshal(record, &current); err != nil {
		return nil, err
	}
	mentions := MentionDIDs(current)
	if current.ExternalImport != nil || len(mentions) == 0 {
		return nil, nil
	}
	var previousRaw json.RawMessage
	err := tx.QueryRow(ctx, `SELECT record FROM tap_source_records
		WHERE uri=$1 AND action<>'delete' AND ordering_status='authoritative'
		  AND structural_validation_status='valid' AND semantic_validation_status='valid'`, uri).Scan(&previousRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var previous craftskylex.FeedPost
	if err := json.Unmarshal(previousRaw, &previous); err != nil {
		return nil, err
	}
	if previous.ExternalImport != nil {
		return nil, nil
	}
	previousMentions := MentionDIDs(previous)
	actor := uri.Authority().DID()
	var added []syntax.DID
	for _, raw := range mentions {
		did, err := syntax.ParseDID(raw)
		if err == nil && did != actor && !slices.Contains(previousMentions, raw) {
			added = append(added, did)
		}
	}
	return added, nil
}

func QueueEditMentions(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, recipients []syntax.DID, activityAt time.Time) error {
	if len(recipients) == 0 {
		return nil
	}
	states, err := ownerlifecycle.LockOwnerStatesTx(ctx, tx, recipients)
	if err != nil {
		return err
	}
	for _, recipient := range recipients {
		if states[recipient].State == ownerlifecycle.StateTerminal {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO post_publication_mentions(post_uri,recipient_did,edit_activity_at)
			SELECT uri,$2,$3 FROM post_publication_state WHERE uri=$1 AND origin_status<>'deleted'
			ON CONFLICT(post_uri,recipient_did) DO NOTHING`, uri, recipient, activityAt); err != nil {
			return err
		}
	}
	return nil
}

type EditMention struct {
	Recipient  syntax.DID
	ActivityAt time.Time
}

func PendingEditMentions(ctx context.Context, tx Reader, uri syntax.ATURI) ([]EditMention, error) {
	rows, err := tx.Query(ctx, `SELECT recipient_did,edit_activity_at FROM post_publication_mentions
		WHERE post_uri=$1 AND edit_activity_at IS NOT NULL ORDER BY recipient_did`, uri)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[EditMention])
}

func FinishEditMention(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, recipient syntax.DID) error {
	_, err := tx.Exec(ctx, `DELETE FROM post_publication_mentions
		WHERE post_uri=$1 AND recipient_did=$2 AND edit_activity_at IS NOT NULL`, uri, recipient)
	return err
}
