package index

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/languages"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	lexiconschema "social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/postutil"
	"social.craftsky/appview/internal/tap"
)

// TransactionalIndexer projects one already-durable source event using the
// transaction owned by the projection worker. Implementations must not commit
// the outer transaction; serving mutations and job completion succeed or roll
// back together.
type TransactionalIndexer interface {
	Project(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error)
}

// TransactionalDispatcher routes durable source rows to transaction-aware
// serving projectors. It validates deterministic record semantics before any
// serving-table mutation so malformed supported records can be quarantined.
type TransactionalDispatcher struct {
	handlers map[syntax.NSID]TransactionalIndexer
}

type projectionOwnerRole uint8

const (
	projectionActorRole projectionOwnerRole = 1 << iota
	projectionRelationTargetRole
	projectionMentionTargetRole
	projectionReferenceTargetRole
)

type terminalProjectionMentionsContextKey struct{}

func NewTransactionalDispatcher() *TransactionalDispatcher {
	return &TransactionalDispatcher{handlers: make(map[syntax.NSID]TransactionalIndexer)}
}

func (dispatcher *TransactionalDispatcher) Register(collection syntax.NSID, indexer TransactionalIndexer) {
	if indexer == nil {
		panic("index.TransactionalDispatcher.Register: indexer must not be nil")
	}
	dispatcher.handlers[collection] = indexer
}

func (dispatcher *TransactionalDispatcher) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	indexer, ok := dispatcher.handlers[source.Collection]
	if !ok {
		return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("no transactional projector registered for %s", source.Collection)
	}
	event := tap.Event{
		URI: source.URI, CID: source.CID, DID: source.DID,
		Collection: source.Collection, Rkey: source.Rkey,
		Action: source.Action, Record: source.Record, Live: source.Live,
		ID: source.SourceEventID, Rev: source.Revision,
	}
	if err := validateProjectionRecord(event); err != nil {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	roles, err := projectionOwnerRoles(event)
	if err != nil {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}
	// Unit-only dispatch tests may omit a transaction. Production projection
	// always supplies the worker-owned transaction and therefore always enters
	// the owner lifecycle fence below.
	if tx != nil {
		owners := make([]syntax.DID, 0, len(roles))
		for owner := range roles {
			owners = append(owners, owner)
		}
		states, err := ownerlifecycle.LockOwnerStatesTx(ctx, tx, owners)
		if err != nil {
			return tap.Retryable(tap.ReasonProjectionFailure), fmt.Errorf("lock projection owner lifecycles: %w", err)
		}
		outcome, ready, cleanupRelation := projectionLifecycleReady(source, roles, states)
		if !ready {
			return outcome, nil
		}
		if cleanupRelation {
			// An update can retarget an existing source URI. Projecting a
			// cleanup delete removes any old non-terminal edge instead of
			// leaving stale serving state while still preventing the new
			// terminal-target edge from being created.
			event.Action = "delete"
			event.Record = nil
			return indexer.Project(ctx, tx, event)
		}
		terminalMentions := make(map[syntax.DID]struct{})
		for owner, role := range roles {
			lifecycle, exists := states[owner]
			if exists && lifecycle.State == ownerlifecycle.StateTerminal && role&projectionMentionTargetRole != 0 {
				terminalMentions[owner] = struct{}{}
			}
		}
		if len(terminalMentions) > 0 {
			ctx = context.WithValue(ctx, terminalProjectionMentionsContextKey{}, terminalMentions)
		}
	}
	return indexer.Project(ctx, tx, event)
}

func projectionLifecycleReady(
	source ingestion.SourceRecord,
	roles map[syntax.DID]projectionOwnerRole,
	states map[syntax.DID]ownerlifecycle.Lifecycle,
) (tap.Outcome, bool, bool) {
	actor, exists := states[source.DID]
	if source.Collection == businessProfileCollection || source.Collection == businessEventCollection {
		if exists && actor.State == ownerlifecycle.StateTerminal {
			return tap.PermanentInvalid(tap.ReasonOwnerTerminal), false, false
		}
		if exists && source.ProjectionGeneration != nil && actor.Generation != *source.ProjectionGeneration {
			return tap.Blocked(tap.ReasonSourceOrderUncertain, tap.Dependency{Kind: "repository_did", Key: source.DID.String()}), false, false
		}
		return tap.Outcome{}, true, false
	}
	if !exists {
		return tap.Blocked(tap.ReasonMissingMember, tap.Dependency{Kind: "member_did", Key: source.DID.String()}), false, false
	}
	if actor.State == ownerlifecycle.StateTerminal {
		return tap.PermanentInvalid(tap.ReasonOwnerTerminal), false, false
	}
	if source.ProjectionGeneration == nil || actor.Generation != *source.ProjectionGeneration {
		return tap.Blocked(tap.ReasonSourceOrderUncertain, tap.Dependency{Kind: "repository_did", Key: source.DID.String()}), false, false
	}
	// A current-generation delete is cleanup and remains valid after the
	// profile transition has moved the actor to departed. Creates and updates
	// require current active authority at this exact transaction boundary.
	if source.Action != "delete" && actor.State != ownerlifecycle.StateActive {
		return tap.Blocked(tap.ReasonOwnerDeparted, tap.Dependency{Kind: "member_did", Key: source.DID.String()}), false, false
	}
	for owner, role := range roles {
		if role&projectionRelationTargetRole == 0 {
			continue
		}
		lifecycle, known := states[owner]
		if known && lifecycle.State == ownerlifecycle.StateTerminal {
			// The source remains durable, but no DID-bearing edge may be
			// recreated after that target's terminal purge. The caller still
			// runs the projector's cleanup path for an older edge at this URI.
			return tap.Outcome{}, true, true
		}
	}
	return tap.Outcome{}, true, false
}

func projectionOwnerRoles(event tap.Event) (map[syntax.DID]projectionOwnerRole, error) {
	roles := map[syntax.DID]projectionOwnerRole{event.DID: projectionActorRole}
	if event.Action == "delete" {
		return roles, nil
	}
	add := func(owner syntax.DID, role projectionOwnerRole) {
		if owner != "" {
			roles[owner] |= role
		}
	}
	addATURIAuthority := func(raw string, role projectionOwnerRole) error {
		if raw == "" {
			return nil
		}
		uri, err := syntax.ParseATURI(raw)
		if err != nil {
			return err
		}
		add(uri.Authority().DID(), role)
		return nil
	}

	switch event.Collection {
	case blueskyFollowNSID:
		var record blueskyFollowRecord
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return nil, err
		}
		add(record.Subject, projectionRelationTargetRole)
	case blueskyBlockNSID:
		var record bsky.GraphBlock
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return nil, err
		}
		subject, err := syntax.ParseDID(record.Subject)
		if err != nil {
			return nil, err
		}
		add(subject, projectionRelationTargetRole)
	case craftskyLikeNSID:
		record, err := decodeCraftskyLike(event.Record)
		if err != nil {
			return nil, err
		}
		if err := addATURIAuthority(record.SubjectURI, projectionRelationTargetRole); err != nil {
			return nil, err
		}
	case craftskyRepostNSID:
		record, err := decodeCraftskyRepost(event.Record)
		if err != nil {
			return nil, err
		}
		if err := addATURIAuthority(record.SubjectURI, projectionRelationTargetRole); err != nil {
			return nil, err
		}
	case craftskyPostNSID:
		var record craftskylex.FeedPost
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return nil, err
		}
		var rawRecord struct {
			Facets json.RawMessage `json:"facets"`
		}
		if err := json.Unmarshal(event.Record, &rawRecord); err != nil {
			return nil, err
		}
		project, err := extractProjectForIndex(event.Record)
		if err != nil {
			return nil, err
		}
		mentions := postutil.MergeMentionDIDs(
			postutil.ExtractMentionDIDsForText(record.Text, postutil.DecodeFacets(rawRecord.Facets)),
			projectMentionDIDs(project),
		)
		for _, rawDID := range mentions {
			mention, err := syntax.ParseDID(rawDID)
			if err != nil {
				return nil, err
			}
			add(mention, projectionMentionTargetRole)
		}
		if record.Reply != nil {
			if record.Reply.Root != nil {
				if err := addATURIAuthority(record.Reply.Root.Uri, projectionReferenceTargetRole); err != nil {
					return nil, err
				}
			}
			if record.Reply.Parent != nil {
				if err := addATURIAuthority(record.Reply.Parent.Uri, projectionReferenceTargetRole); err != nil {
					return nil, err
				}
			}
		}
		if record.Embed != nil && record.Embed.FeedPost_QuoteEmbed != nil && record.Embed.FeedPost_QuoteEmbed.Record != nil {
			if err := addATURIAuthority(record.Embed.FeedPost_QuoteEmbed.Record.Uri, projectionReferenceTargetRole); err != nil {
				return nil, err
			}
		}
	}
	return roles, nil
}

func filterTerminalProjectionMentions(ctx context.Context, mentionedDIDs []string) []string {
	terminal, _ := ctx.Value(terminalProjectionMentionsContextKey{}).(map[syntax.DID]struct{})
	if len(terminal) == 0 || len(mentionedDIDs) == 0 {
		return mentionedDIDs
	}
	filtered := make([]string, 0, len(mentionedDIDs))
	for _, rawDID := range mentionedDIDs {
		if _, denied := terminal[syntax.DID(rawDID)]; !denied {
			filtered = append(filtered, rawDID)
		}
	}
	return filtered
}

func validateProjectionRecord(event tap.Event) error {
	switch event.Collection {
	case businessProfileCollection:
		if event.Rkey != "self" {
			return errors.New("business profile record key must be self")
		}
	case businessEventCollection:
		if _, err := syntax.ParseTID(event.Rkey.String()); err != nil {
			return errors.New("business event record key must be a TID")
		}
	}
	switch event.Action {
	case "delete":
		return nil
	case "create", "update":
	default:
		return errors.New("unsupported source action")
	}
	if len(event.Record) == 0 || !json.Valid(event.Record) {
		return errors.New("missing or malformed source record")
	}
	switch event.Collection {
	case businessProfileCollection:
		if err := lexiconschema.ValidateBusinessRecord(event.Record, event.Collection.String()); err != nil {
			return err
		}
		var record craftskylex.BusinessProfile
		return json.Unmarshal(event.Record, &record)
	case businessEventCollection:
		if err := lexiconschema.ValidateBusinessRecord(event.Record, event.Collection.String()); err != nil {
			return err
		}
		var record craftskylex.BusinessEvent
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339Nano, record.StartsAt); err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339Nano, record.EndsAt); err != nil {
			return err
		}
		_, err := time.Parse(time.RFC3339Nano, record.CreatedAt)
		return err
	case craftskyProfileNSID:
		var record craftskylex.ActorProfile
		return json.Unmarshal(event.Record, &record)
	case craftskyPostNSID:
		var record craftskylex.FeedPost
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return err
		}
		if err := languages.ValidatePostTags(record.Langs); err != nil {
			return err
		}
		if _, err := time.Parse(time.RFC3339, record.CreatedAt); err != nil {
			return err
		}
		project, err := extractProjectForIndex(event.Record)
		if err != nil {
			return err
		}
		for _, rawDID := range projectMentionDIDs(project) {
			if _, err := syntax.ParseDID(rawDID); err != nil {
				return err
			}
		}
		return nil
	case craftskyLikeNSID:
		return validateInteractionRecord(event.Record, decodeCraftskyLike)
	case craftskyRepostNSID:
		return validateInteractionRecord(event.Record, decodeCraftskyRepost)
	case blueskyProfileNSID:
		var record blueskyProfileRecord
		return json.Unmarshal(event.Record, &record)
	case blueskyFollowNSID:
		var record blueskyFollowRecord
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return err
		}
		if _, err := syntax.ParseDID(record.Subject.String()); err != nil {
			return err
		}
		_, err := time.Parse(time.RFC3339, record.CreatedAt)
		return err
	case blueskyBlockNSID:
		var record bsky.GraphBlock
		if err := json.Unmarshal(event.Record, &record); err != nil {
			return err
		}
		if _, err := syntax.ParseDID(record.Subject); err != nil {
			return err
		}
		_, err := time.Parse(time.RFC3339, record.CreatedAt)
		return err
	default:
		return fmt.Errorf("unsupported source collection %s", event.Collection)
	}
}

func validateInteractionRecord(raw json.RawMessage, decode func(json.RawMessage) (craftskyInteractionRecord, error)) error {
	record, err := decode(raw)
	if err != nil {
		return err
	}
	if record.SubjectURI == "" || record.SubjectCID == "" {
		return errors.New("interaction subject is required")
	}
	if _, err := syntax.ParseATURI(record.SubjectURI); err != nil {
		return err
	}
	_, err = time.Parse(time.RFC3339, record.CreatedAt)
	return err
}
