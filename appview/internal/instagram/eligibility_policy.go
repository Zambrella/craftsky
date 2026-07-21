package instagram

import (
	"context"
	"database/sql"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SuggestionEligibilityRequest struct {
	ImporterDID      syntax.DID
	TargetDID        syntax.DID
	ImportedUsername string
	Direction        ImportDirection
}

func (SuggestionEligibilityRequest) String() string {
	return "Instagram suggestion eligibility request [REDACTED]"
}

type InstagramSuggestionEligibilityPolicy interface {
	Evaluate(context.Context, EligibilityStage, SuggestionEligibilityRequest) (EligibilityDecision, error)
}

type RelationshipSafetyFacts struct {
	Available            bool
	ImporterBlocksTarget bool
	TargetBlocksImporter bool
	ImporterMutesTarget  bool
}

type RelationshipSafetyProvider interface {
	RelationshipSafety(context.Context, syntax.DID, syntax.DID) (RelationshipSafetyFacts, error)
}

// PostgresInstagramSuggestionEligibilityPolicy gathers every currently
// available membership, identity, follow, and moderation fact and then calls
// the one pure policy used at every stage. Relationship block/mute data has no
// production source in the repository yet, so a nil or failing provider is
// deliberately represented as unavailable and the decision fails closed.
type PostgresInstagramSuggestionEligibilityPolicy struct {
	pool   *pgxpool.Pool
	safety RelationshipSafetyProvider
	now    func() time.Time
}

func NewPostgresInstagramSuggestionEligibilityPolicy(pool *pgxpool.Pool, safety RelationshipSafetyProvider, now func() time.Time) *PostgresInstagramSuggestionEligibilityPolicy {
	if now == nil {
		now = time.Now
	}
	return &PostgresInstagramSuggestionEligibilityPolicy{pool: pool, safety: safety, now: now}
}

func (p *PostgresInstagramSuggestionEligibilityPolicy) Evaluate(ctx context.Context, stage EligibilityStage, request SuggestionEligibilityRequest) (EligibilityDecision, error) {
	if p == nil || p.pool == nil {
		return EligibilityDecision{Reason: EligibilitySafetyUnavailable}, nil
	}
	if request.ImporterDID == "" || request.TargetDID == "" || !request.Direction.Valid() {
		return EligibilityDecision{Reason: EligibilityInvalidInput}, nil
	}
	var (
		importerMember bool
		targetMember   bool
		linkStateRaw   string
		discoverable   bool
		conflict       bool
		currentName    sql.NullString
		verifiedAt     sql.NullTime
		alreadyFollow  bool
		hidden         bool
		takenDown      bool
	)
	err := p.pool.QueryRow(ctx, eligibilitySnapshotQuery,
		request.ImporterDID, request.TargetDID, p.now().UTC()).Scan(
		&importerMember,
		&targetMember,
		&linkStateRaw,
		&discoverable,
		&conflict,
		&currentName,
		&verifiedAt,
		&alreadyFollow,
		&hidden,
		&takenDown,
	)
	if err != nil {
		return EligibilityDecision{}, err
	}
	safety := RelationshipSafetyFacts{}
	if p.safety != nil {
		facts, err := p.safety.RelationshipSafety(ctx, request.ImporterDID, request.TargetDID)
		if err == nil {
			safety = facts
		}
	}
	snapshot := EligibilitySnapshot{
		ImporterCurrentMember: importerMember,
		TargetCurrentMember:   targetMember,
		LinkState:             InstagramLinkState(linkStateRaw),
		DMVerified:            verifiedAt.Valid,
		Discoverable:          discoverable,
		ConflictFree:          linkStateRaw != string(LinkDisputed) && !conflict,
		ImportDirection:       request.Direction,
		ImportedUsername:      request.ImportedUsername,
		CurrentUsername:       currentName.String,
		Self:                  request.ImporterDID == request.TargetDID,
		AlreadyFollowing:      alreadyFollow,
		TargetHidden:          hidden,
		TargetTakenDown:       takenDown,
		ImporterBlocksTarget:  safety.ImporterBlocksTarget,
		TargetBlocksImporter:  safety.TargetBlocksImporter,
		ImporterMutesTarget:   safety.ImporterMutesTarget,
		SafetyDataAvailable:   safety.Available,
	}
	return EvaluateInstagramSuggestionEligibility(stage, snapshot), nil
}

func (*PostgresInstagramSuggestionEligibilityPolicy) String() string {
	return "Postgres InstagramSuggestionEligibilityPolicy{facts:[REDACTED]}"
}

var _ InstagramSuggestionEligibilityPolicy = (*PostgresInstagramSuggestionEligibilityPolicy)(nil)

const eligibilitySnapshotQuery = `
	SELECT
		EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $1),
		EXISTS (SELECT 1 FROM craftsky_profiles WHERE did = $2),
		COALESCE(link.state, ''),
		COALESCE(link.discoverable, false),
		COALESCE(link.conflict_pending, false),
		link.username_normalized,
		link.verified_at,
		EXISTS (
			SELECT 1 FROM atproto_follows
			WHERE did = $1 AND subject_did = $2
		),
		EXISTS (` + effectiveAccountModerationQuery + ` AND applied.value = 'hide'),
		EXISTS (` + effectiveAccountModerationQuery + ` AND applied.value = 'takedown')
	FROM (VALUES (1)) singleton(value)
	LEFT JOIN LATERAL (
		SELECT state, discoverable, conflict_pending, username_normalized, verified_at
		FROM instagram_account_links
		WHERE owner_did = $2
		  AND state IN ('active', 'membershipInactive', 'disputed')
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
	) link ON true`

const effectiveAccountModerationQuery = `
		SELECT 1
		FROM moderation_outputs applied
		WHERE applied.subject_type = 'account'
		  AND applied.subject_did = $2
		  AND applied.action = 'apply'
		  AND (applied.expires_at IS NULL OR applied.expires_at > $3)
		  AND NOT EXISTS (
			SELECT 1 FROM moderation_outputs negated
			WHERE negated.source_did = applied.source_did
			  AND negated.subject_type = applied.subject_type
			  AND negated.subject_did = applied.subject_did
			  AND negated.value = applied.value
			  AND negated.action = 'negate'
			  AND negated.indexed_at > applied.indexed_at
		  )`
