package business

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool              *pgxpool.Pool
	accessEnvironment string
}

var ErrEventNotFound = errors.New("business: event not found")

type EventReadInput struct {
	CallerDID syntax.DID
	OwnerDID  syntax.DID
	Rkey      syntax.RecordKey
	AsOf      time.Time
}

type UpcomingEventListInput struct {
	CallerDID syntax.DID
	OwnerDID  syntax.DID
	AsOf      time.Time
	Limit     int
	Seek      *UpcomingEventSeek
}

type UpcomingEventSeek struct {
	StartsAt time.Time
	URI      syntax.ATURI
}

type OwnerEventListInput struct {
	OwnerDID syntax.DID
	Filter   OwnerEventFilter
	AsOf     time.Time
	Limit    int
	Seek     *OwnerEventSeek
}

type OwnerEventSeek struct {
	StartsAt time.Time
	URI      syntax.ATURI
}

const eventModeratedSQL = `EXISTS (
	SELECT 1
	FROM moderation_outputs applied
	WHERE applied.action = 'apply'
	  AND NOT appview_owner_is_terminal(applied.source_did)
	  AND applied.value IN ('hide', 'takedown')
	  AND (applied.expires_at IS NULL OR applied.expires_at > now())
	  AND ((applied.subject_type = 'event' AND applied.subject_uri = event.uri)
	       OR (applied.subject_type = 'account' AND applied.subject_did = event.owner_did))
	  AND NOT EXISTS (
		SELECT 1
		FROM moderation_outputs negated
		WHERE negated.action = 'negate'
		  AND negated.source_did = applied.source_did
		  AND negated.subject_type = applied.subject_type
		  AND negated.subject_did = applied.subject_did
		  AND negated.value = applied.value
		  AND (negated.expires_at IS NULL OR negated.expires_at > now())
		  AND negated.indexed_at > applied.indexed_at
		  AND (applied.subject_type = 'account' OR negated.subject_uri = applied.subject_uri)
	  )
)`

const eventBlockedSQL = `EXISTS (
	SELECT 1
	FROM pds_set_aggregates block
	WHERE block.kind = 'block'
	  AND ((block.actor_did = $1 AND block.subject_did = event.owner_did)
	    OR (block.actor_did = event.owner_did AND block.subject_did = $1))
	  AND NOT appview_owner_is_terminal(block.actor_did)
	  AND NOT appview_owner_is_terminal(block.subject_did)
)`

func (s *Store) ReadEvent(ctx context.Context, input EventReadInput) (EventView, error) {
	var raw json.RawMessage
	var view EventView
	var startsAt, endsAt time.Time
	var ownerCurrent, blocked, moderated bool
	var accountType AccountType
	err := s.pool.QueryRow(ctx, `
		SELECT event.raw_record, event.uri, event.cid, event.starts_at, event.ends_at,
		       membership.did IS NOT NULL AND appview_owner_is_active(event.owner_did),
		       CASE WHEN EXISTS (
		           SELECT 1 FROM billing_licenses license
		           JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		           WHERE license.assigned_did=event.owner_did AND license.tier='business'
		             AND subscription.gives_access AND subscription.environment=$4
		             AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
		             AND subscription.anomaly='none'
		       ) THEN 'business' ELSE 'regular' END,
		       `+eventBlockedSQL+`, `+eventModeratedSQL+`
		FROM craftsky_business_events event
		LEFT JOIN craftsky_profiles membership ON membership.did = event.owner_did
		WHERE event.owner_did = $2 AND event.rkey = $3
		  AND appview_image_subject_is_clear(event.uri, event.cid)
	`, input.CallerDID, input.OwnerDID, input.Rkey, s.accessEnvironment).Scan(
		&raw, &view.URI, &view.CID, &startsAt, &endsAt,
		&ownerCurrent, &accountType, &blocked, &moderated,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EventView{}, ErrEventNotFound
	}
	if err != nil {
		return EventView{}, fmt.Errorf("read business event: %w", err)
	}
	hydrated, err := HydrateEvent(raw)
	if err != nil {
		return EventView{}, fmt.Errorf("hydrate business event: %w", err)
	}
	hydrated.DID = input.OwnerDID
	hydrated.Rkey = input.Rkey
	hydrated.URI = view.URI
	hydrated.CID = view.CID
	policy := EvaluateEvent(EventPolicyInput{
		CallerIsOwner: input.CallerDID == input.OwnerDID,
		OwnerCurrent:  ownerCurrent, AccountType: accountType, Blocked: blocked,
		StartsAt: startsAt, EndsAt: endsAt, Status: hydrated.Status.Value,
		Moderated: moderated, AsOf: input.AsOf,
	})
	if !policy.DirectVisible {
		return EventView{}, ErrEventNotFound
	}
	applyEventPolicy(&hydrated, policy)
	return hydrated, nil
}

func (s *Store) ListUpcomingEvents(ctx context.Context, input UpcomingEventListInput) ([]EventView, error) {
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 50 {
		input.Limit = 50
	}
	var seekStartsAt any
	var seekURI any
	if input.Seek != nil {
		seekStartsAt = input.Seek.StartsAt
		seekURI = input.Seek.URI
	}
	rows, err := s.pool.Query(ctx, `
		SELECT event.raw_record, event.uri, event.rkey, event.cid,
		       event.starts_at, event.ends_at, 'business'
		FROM craftsky_business_events event
		JOIN craftsky_profiles membership ON membership.did = event.owner_did
		JOIN billing_licenses license ON license.assigned_did=event.owner_did AND license.tier='business'
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		  AND subscription.gives_access AND subscription.environment=$7
		  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
		  AND subscription.anomaly='none'
		WHERE event.owner_did = $2
		  AND appview_owner_is_active(event.owner_did)
		  AND appview_image_subject_is_clear(event.uri, event.cid)
		  AND event.ends_at > $3
		  AND event.ends_at > event.starts_at
		  AND event.ends_at - event.starts_at <= interval '31 days'
		  AND COALESCE(event.status, 'scheduled') NOT IN ('cancelled', 'postponed')
		  AND NOT `+eventBlockedSQL+`
		  AND NOT `+eventModeratedSQL+`
		  AND ($4::timestamptz IS NULL OR
		       (event.starts_at, event.uri) > ($4::timestamptz, $5::text))
		ORDER BY event.starts_at ASC, event.uri ASC
		LIMIT $6
	`, input.CallerDID, input.OwnerDID, input.AsOf, seekStartsAt, seekURI, input.Limit+1, s.accessEnvironment)
	if err != nil {
		return nil, fmt.Errorf("list upcoming business events: %w", err)
	}
	defer rows.Close()
	views := make([]EventView, 0)
	for rows.Next() {
		var raw json.RawMessage
		var view EventView
		var startsAt, endsAt time.Time
		var accountType AccountType
		if err := rows.Scan(&raw, &view.URI, &view.Rkey, &view.CID, &startsAt, &endsAt, &accountType); err != nil {
			return nil, fmt.Errorf("scan upcoming business event: %w", err)
		}
		hydrated, err := HydrateEvent(raw)
		if err != nil {
			return nil, fmt.Errorf("hydrate upcoming business event: %w", err)
		}
		hydrated.DID = input.OwnerDID
		hydrated.URI, hydrated.Rkey, hydrated.CID = view.URI, view.Rkey, view.CID
		policy := EvaluateEvent(EventPolicyInput{
			CallerIsOwner: input.CallerDID == input.OwnerDID,
			OwnerCurrent:  true, AccountType: accountType,
			StartsAt: startsAt, EndsAt: endsAt, Status: hydrated.Status.Value, AsOf: input.AsOf,
		})
		if policy.Upcoming {
			applyEventPolicy(&hydrated, policy)
			views = append(views, hydrated)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate upcoming business events: %w", err)
	}
	return views, nil
}

func (s *Store) ListOwnerEvents(ctx context.Context, input OwnerEventListInput) ([]EventView, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 50 {
		input.Limit = 50
	}
	var seekStartsAt any
	var seekURI any
	if input.Seek != nil {
		seekStartsAt = input.Seek.StartsAt
		seekURI = input.Seek.URI
	}
	query := `
		SELECT event.raw_record, event.uri, event.rkey, event.cid,
		       event.starts_at, event.ends_at,
		       'business', ` + eventModeratedSQL + `
		FROM craftsky_business_events event
		JOIN craftsky_profiles membership ON membership.did = event.owner_did
		JOIN billing_licenses license ON license.assigned_did=event.owner_did AND license.tier='business'
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		  AND subscription.gives_access AND subscription.environment=$5
		  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
		  AND subscription.anomaly='none'
		WHERE event.owner_did = $1
		  AND appview_owner_is_active(event.owner_did)
		  AND appview_image_subject_is_clear(event.uri, event.cid)
		  AND ($2::timestamptz IS NULL OR
		       (event.starts_at, event.uri) < ($2::timestamptz, $3::text))
		ORDER BY event.starts_at DESC, event.uri DESC
		LIMIT $4
	`
	args := []any{input.OwnerDID, seekStartsAt, seekURI, input.Limit + 1, s.accessEnvironment}
	if input.Filter == OwnerEventUpcoming {
		query = `
			SELECT event.raw_record, event.uri, event.rkey, event.cid,
			       event.starts_at, event.ends_at,
			       'business', ` + eventModeratedSQL + `
			FROM craftsky_business_events event
			JOIN craftsky_profiles membership ON membership.did = event.owner_did
			JOIN billing_licenses license ON license.assigned_did=event.owner_did AND license.tier='business'
			JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
			  AND subscription.gives_access AND subscription.environment=$6
			  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
			  AND subscription.anomaly='none'
			WHERE event.owner_did = $1
			  AND appview_owner_is_active(event.owner_did)
			  AND appview_image_subject_is_clear(event.uri, event.cid)
			  AND COALESCE(event.status, 'scheduled') = 'scheduled'
			  AND event.ends_at > $2
			  AND ($3::timestamptz IS NULL OR
			       (event.starts_at, event.uri) > ($3::timestamptz, $4::text))
			ORDER BY event.starts_at ASC, event.uri ASC
			LIMIT $5
		`
		args = []any{input.OwnerDID, input.AsOf, seekStartsAt, seekURI, input.Limit + 1, s.accessEnvironment}
	} else if input.Filter == OwnerEventHistory {
		query = `
			SELECT event.raw_record, event.uri, event.rkey, event.cid,
			       event.starts_at, event.ends_at,
			       'business', ` + eventModeratedSQL + `
			FROM craftsky_business_events event
			JOIN craftsky_profiles membership ON membership.did = event.owner_did
			JOIN billing_licenses license ON license.assigned_did=event.owner_did AND license.tier='business'
			JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
			  AND subscription.gives_access AND subscription.environment=$6
			  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
			  AND subscription.anomaly='none'
			WHERE event.owner_did = $1
			  AND appview_owner_is_active(event.owner_did)
			  AND appview_image_subject_is_clear(event.uri, event.cid)
			  AND NOT (COALESCE(event.status, 'scheduled') = 'scheduled' AND event.ends_at > $2)
			  AND ($3::timestamptz IS NULL OR
			       (event.starts_at, event.uri) < ($3::timestamptz, $4::text))
			ORDER BY event.starts_at DESC, event.uri DESC
			LIMIT $5
		`
		args = []any{input.OwnerDID, input.AsOf, seekStartsAt, seekURI, input.Limit + 1, s.accessEnvironment}
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list owner business events: %w", err)
	}
	defer rows.Close()
	views := make([]EventView, 0)
	for rows.Next() {
		var raw json.RawMessage
		var identity EventView
		var startsAt, endsAt time.Time
		var accountType AccountType
		var moderated bool
		if err := rows.Scan(&raw, &identity.URI, &identity.Rkey, &identity.CID, &startsAt, &endsAt, &accountType, &moderated); err != nil {
			return nil, fmt.Errorf("scan owner business event: %w", err)
		}
		view, err := HydrateEvent(raw)
		if err != nil {
			return nil, fmt.Errorf("hydrate owner business event: %w", err)
		}
		view.DID = input.OwnerDID
		view.URI, view.Rkey, view.CID = identity.URI, identity.Rkey, identity.CID
		policy := EvaluateEvent(EventPolicyInput{
			CallerIsOwner: true, OwnerCurrent: true, AccountType: accountType,
			StartsAt: startsAt, EndsAt: endsAt, Status: view.Status.Value,
			Moderated: moderated, AsOf: input.AsOf,
		})
		applyEventPolicy(&view, policy)
		views = append(views, view)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner business events: %w", err)
	}
	return views, nil
}

func applyEventPolicy(view *EventView, policy EventPolicyResult) {
	view.Past = policy.Past
	view.PublicSuppressionReasons = make([]string, len(policy.PublicSuppressionReasons))
	copy(view.PublicSuppressionReasons, policy.PublicSuppressionReasons)
	view.UpcomingExclusionReasons = make([]string, len(policy.UpcomingExclusionReasons))
	copy(view.UpcomingExclusionReasons, policy.UpcomingExclusionReasons)
}

func (s *Store) ReadEligibleProfile(ctx context.Context, did syntax.DID) (*ProfileView, error) {
	var raw json.RawMessage
	var cid syntax.CID
	err := s.pool.QueryRow(ctx, `
		SELECT business_profile.raw_record, business_profile.cid
		FROM craftsky_business_profiles AS business_profile
		JOIN craftsky_profiles AS membership
		  ON membership.did = business_profile.owner_did
		JOIN billing_licenses AS license
		  ON license.assigned_did = business_profile.owner_did AND license.tier = 'business'
		JOIN provider_subscriptions AS subscription
		  ON subscription.id = license.provider_subscription_id
		 AND subscription.gives_access AND subscription.environment=$2
		 AND subscription.app_id IS NOT NULL AND subscription.mapped_tier = license.tier
		 AND subscription.anomaly = 'none'
		WHERE business_profile.owner_did = $1
		  AND appview_owner_is_active(business_profile.owner_did)
		  AND appview_image_subject_is_clear(business_profile.uri, business_profile.cid)
	`, did, s.accessEnvironment).Scan(&raw, &cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read eligible business profile: %w", err)
	}
	view, err := HydrateProfile(raw)
	if err != nil {
		return nil, fmt.Errorf("hydrate eligible business profile: %w", err)
	}
	view.CID = cid
	return &view, nil
}

func (s *Store) HasUpcomingEvents(ctx context.Context, owner syntax.DID, asOf time.Time) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM craftsky_business_events event
			JOIN craftsky_profiles membership ON membership.did = event.owner_did
			JOIN billing_licenses license ON license.assigned_did = event.owner_did AND license.tier = 'business'
			JOIN provider_subscriptions subscription ON subscription.id = license.provider_subscription_id
			  AND subscription.gives_access AND subscription.environment=$3
			  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier = license.tier
			  AND subscription.anomaly = 'none'
			WHERE event.owner_did = $1
			  AND appview_owner_is_active(event.owner_did)
			  AND appview_image_subject_is_clear(event.uri, event.cid)
			  AND event.ends_at > $2
			  AND event.ends_at > event.starts_at
			  AND event.ends_at - event.starts_at <= interval '31 days'
			  AND COALESCE(event.status, 'scheduled') NOT IN ('cancelled', 'postponed')
			  AND NOT `+eventModeratedSQL+`
		)
	`, owner, asOf, s.accessEnvironment).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("read upcoming business event availability: %w", err)
	}
	return exists, nil
}

func NewStore(pool *pgxpool.Pool) *Store {
	return NewStoreForEnvironment(pool, "production")
}

func NewStoreForEnvironment(pool *pgxpool.Pool, environment string) *Store {
	if environment != "sandbox" {
		environment = "production"
	}
	return &Store{pool: pool, accessEnvironment: environment}
}

func (s *Store) ReadAccountType(ctx context.Context, did syntax.DID) (AccountType, error) {
	var active bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM billing_licenses license
			JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
			WHERE license.assigned_did=$1 AND license.tier='business'
			  AND subscription.gives_access AND subscription.environment=$2
			  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
			  AND subscription.anomaly='none'
		)
	`, did, s.accessEnvironment).Scan(&active)
	if err != nil {
		return "", fmt.Errorf("read account type: %w", err)
	}
	if active {
		return AccountTypeBusiness, nil
	}
	return AccountTypeRegular, nil
}

func (s *Store) ReadAccountTypes(ctx context.Context, dids []syntax.DID) (map[syntax.DID]AccountType, error) {
	result := make(map[syntax.DID]AccountType)
	if len(dids) == 0 {
		return result, nil
	}
	values := make([]string, 0, len(dids))
	seen := make(map[syntax.DID]struct{}, len(dids))
	for _, did := range dids {
		if _, exists := seen[did]; exists {
			continue
		}
		seen[did] = struct{}{}
		values = append(values, did.String())
	}
	rows, err := s.pool.Query(ctx, `
		SELECT license.assigned_did
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE license.assigned_did=ANY($1) AND license.tier='business'
		  AND subscription.gives_access AND subscription.environment=$2
		  AND subscription.app_id IS NOT NULL AND subscription.mapped_tier=license.tier
		  AND subscription.anomaly='none'
	`, values, s.accessEnvironment)
	if err != nil {
		return nil, fmt.Errorf("read account types: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var did syntax.DID
		if err := rows.Scan(&did); err != nil {
			return nil, fmt.Errorf("scan account type: %w", err)
		}
		result[did] = AccountTypeBusiness
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account types: %w", err)
	}
	return result, nil
}
