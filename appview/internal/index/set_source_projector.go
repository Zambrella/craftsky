package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/sourcevalidation"
	"social.craftsky/appview/internal/tap"
)

func projectSetSourceTx(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord, now time.Time) ([]SetAggregateChange, error) {
	if source.Action == "delete" || source.OrderingStatus != "authoritative" ||
		source.StructuralValidationStatus != sourcevalidation.Valid ||
		source.SemanticValidationStatus != sourcevalidation.Valid {
		return replaceSetSourceTx(ctx, tx, source.URI, nil, now)
	}
	fact, err := normalizeSetSource(source)
	if err != nil {
		return nil, err
	}
	actorEligible, err := activeSetActorTx(ctx, tx, source.DID)
	if err != nil {
		return nil, err
	}
	if !actorEligible {
		dependency := tap.Dependency{Kind: "member_did", Key: source.DID.String()}
		fact = classifySetSource(fact, false, dependency, false, "owner_departed")
	} else if fact.Scope.Kind == "like" || fact.Scope.Kind == "repost" {
		available, err := setSubjectAvailableTx(ctx, tx, fact.SubjectURI)
		if err != nil {
			return nil, err
		}
		dependency := tap.Dependency{Kind: "subject_uri", Key: fact.SubjectURI.String()}
		fact = classifySetSource(fact, true, dependency, available, "missing_subject")
	} else {
		fact = classifySetSource(fact, true, tap.Dependency{}, true, "")
	}
	return replaceSetSourceTx(ctx, tx, source.URI, &fact, now)
}

func normalizeSetSource(source ingestion.SourceRecord) (SetSource, error) {
	fact := SetSource{
		URI:   source.URI,
		Scope: SetScope{Actor: source.DID},
	}
	switch source.Collection {
	case blueskyFollowNSID:
		var record bsky.GraphFollow
		if err := json.Unmarshal(source.Record, &record); err != nil {
			return SetSource{}, err
		}
		return normalizeDIDSetSource(fact, "follow", record.Subject, record.CreatedAt)
	case blueskyBlockNSID:
		var record bsky.GraphBlock
		if err := json.Unmarshal(source.Record, &record); err != nil {
			return SetSource{}, err
		}
		return normalizeDIDSetSource(fact, "block", record.Subject, record.CreatedAt)
	case craftskyLikeNSID:
		var record craftskylex.FeedLike
		if err := json.Unmarshal(source.Record, &record); err != nil {
			return SetSource{}, err
		}
		return normalizeRecordSetSource(fact, "like", record.Subject, record.CreatedAt)
	case craftskyRepostNSID:
		var record craftskylex.FeedRepost
		if err := json.Unmarshal(source.Record, &record); err != nil {
			return SetSource{}, err
		}
		return normalizeRecordSetSource(fact, "repost", record.Subject, record.CreatedAt)
	default:
		return SetSource{}, fmt.Errorf("collection %s is not set-like", source.Collection)
	}
}

func normalizeDIDSetSource(fact SetSource, kind, rawSubject, rawActivity string) (SetSource, error) {
	subject, err := syntax.ParseDID(rawSubject)
	if err != nil {
		return SetSource{}, err
	}
	activityAt, err := time.Parse(time.RFC3339Nano, rawActivity)
	if err != nil {
		return SetSource{}, err
	}
	fact.Scope.Kind = kind
	fact.Scope.Key = subject.String()
	fact.SubjectDID = subject
	fact.ActivityAt = activityAt
	return fact, nil
}

func normalizeRecordSetSource(fact SetSource, kind string, subject *comatproto.RepoStrongRef, rawActivity string) (SetSource, error) {
	if subject == nil {
		return SetSource{}, errors.New("set source subject is required")
	}
	uri, err := syntax.ParseATURI(subject.Uri)
	if err != nil {
		return SetSource{}, err
	}
	activityAt, err := time.Parse(time.RFC3339Nano, rawActivity)
	if err != nil {
		return SetSource{}, err
	}
	fact.Scope.Kind = kind
	fact.Scope.Key = uri.String()
	fact.SubjectURI = uri
	fact.SubjectCID = syntax.CID(subject.Cid)
	fact.ActivityAt = activityAt
	return fact, nil
}

func activeSetActorTx(ctx context.Context, tx pgx.Tx, actor syntax.DID) (bool, error) {
	var active bool
	err := tx.QueryRow(ctx, `SELECT state='active' FROM owner_lifecycles WHERE owner_did=$1`, actor).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read set actor lifecycle: %w", err)
	}
	return active, nil
}

func setSubjectAvailableTx(ctx context.Context, tx pgx.Tx, subject syntax.ATURI) (bool, error) {
	var available bool
	err := tx.QueryRow(ctx, `SELECT true FROM craftsky_posts WHERE uri=$1`, subject).Scan(&available)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read set subject: %w", err)
	}
	return available, nil
}
