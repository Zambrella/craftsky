package instagram

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestAccountStoreReadsOnlyTheOwnersCurrentPrivateLink(t *testing.T) {
	store, pool := newAccountStoreTest(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	alice := syntax.DID("did:plc:synthetic-account-alice")
	bob := syntax.DID("did:plc:synthetic-account-bob")
	username := "synthetic.private.username"
	igsid := "synthetic-private-igsid-canary"

	insertAccountLink(t, pool, accountLinkFixture{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000201"),
		Owner:        alice,
		State:        LinkActive,
		IGSID:        igsid,
		Username:     username,
		Discoverable: true,
		VerifiedAt:   now.Add(-time.Hour),
		UpdatedAt:    now,
	})
	insertAccountLink(t, pool, accountLinkFixture{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000202"),
		Owner:      bob,
		State:      LinkRevoked,
		VerifiedAt: now.Add(-2 * time.Hour),
		UpdatedAt:  now,
	})

	account, err := store.GetAccount(ctx, alice)
	if err != nil {
		t.Fatalf("GetAccount Alice: %v", err)
	}
	if account == nil {
		t.Fatal("GetAccount Alice = nil")
	}
	if account.State != LinkActive || account.Username != username || !account.Discoverable || account.ConflictPending || account.ReactivationRequired || !account.VerifiedAt.Equal(now.Add(-time.Hour)) {
		t.Fatalf("Alice account = %+v", account)
	}

	missing, err := store.GetAccount(ctx, syntax.DID("did:plc:synthetic-account-charlie"))
	if err != nil {
		t.Fatalf("GetAccount absent owner: %v", err)
	}
	if missing != nil {
		t.Fatalf("absent account = %+v, want nil", missing)
	}
	revoked, err := store.GetAccount(ctx, bob)
	if err != nil {
		t.Fatalf("GetAccount revoked owner: %v", err)
	}
	if revoked != nil {
		t.Fatalf("revoked tombstone surfaced as current account: %+v", revoked)
	}

	diagnostic := fmt.Sprintf("account=%+v store=%+v", account, *store)
	for _, private := range []string{username, igsid} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("account/store String leaked %q: %s", private, diagnostic)
		}
	}
}

func TestAccountStoreUpdatesDiscoveryAndInvalidatesEveryUnfinishedDependent(t *testing.T) {
	store, pool := newAccountStoreTest(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	alice := syntax.DID("did:plc:synthetic-settings-alice")

	insertAccountLink(t, pool, accountLinkFixture{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000211"),
		Owner:        alice,
		State:        LinkActive,
		IGSID:        "synthetic-settings-igsid",
		Username:     "synthetic.settings.username",
		Discoverable: true,
		VerifiedAt:   now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Minute),
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions
			(id, importer_did, target_did, state, reason, accepting_since, created_at, updated_at)
		VALUES
			('00000000-0000-0000-0000-000000000212', 'did:plc:synthetic-importer-a', $1, 'accepting', 'verifiedInstagramFollow', $2, $2, $2),
			('00000000-0000-0000-0000-000000000213', 'did:plc:synthetic-importer-b', $1, 'accepted', 'verifiedInstagramFollow', NULL, $2, $2)
	`, alice, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert dependent suggestions: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_follow_operations(
			id,suggestion_id,owner_did,target_did,rkey,status,
			attempt_count,created_at,updated_at
		) VALUES(
			'00000000-0000-0000-0000-000000000214',
			'00000000-0000-0000-0000-000000000212',
			'did:plc:synthetic-importer-a',$1,'3kydiscoverydisable','writing',1,$2,$2
		)
	`, alice, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert dependent follow operation: %v", err)
	}
	seedLifecycleNotification(
		t,
		pool,
		uuid.MustParse("00000000-0000-0000-0000-000000000215"),
		syntax.DID("did:plc:synthetic-importer-a"),
		uuid.MustParse("00000000-0000-0000-0000-000000000212"),
		"00000000-0000-0000-0000-000000000216",
		"leased",
		now.Add(-time.Minute),
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs(
			id,owner_did,target_did,link_id,reason,status,next_attempt_at,created_at,updated_at
		) VALUES(
			'00000000-0000-0000-0000-000000000217',
			'did:plc:synthetic-importer-a',$1,'00000000-0000-0000-0000-000000000211',
			'syntheticDiscoveryRace','processing',$2,$2,$2
		)
	`, alice, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert dependent reconciliation: %v", err)
	}

	disabled, err := store.UpdateSettings(ctx, alice, AccountSettingsPatch{Discoverable: accountBool(false)})
	if err != nil {
		t.Fatalf("disable discovery: %v", err)
	}
	if disabled.State != LinkActive || disabled.Discoverable {
		t.Fatalf("disabled account = %+v", disabled)
	}

	var (
		pendingState     InstagramSuggestionState
		pendingTerminal  *time.Time
		acceptedState    InstagramSuggestionState
		acceptedTerminal *time.Time
		followStatus     string
		eventState       string
		deliveryStatus   string
		jobStatus        string
	)
	if err := pool.QueryRow(ctx, `
		SELECT p.state, p.terminal_at, a.state, a.terminal_at
		FROM instagram_follow_suggestions p
		JOIN instagram_follow_suggestions a ON a.id = '00000000-0000-0000-0000-000000000213'
		WHERE p.id = '00000000-0000-0000-0000-000000000212'
	`).Scan(&pendingState, &pendingTerminal, &acceptedState, &acceptedTerminal); err != nil {
		t.Fatalf("inspect suggestions: %v", err)
	}
	if pendingState != SuggestionInvalidated || pendingTerminal == nil || !pendingTerminal.Equal(now) {
		t.Fatalf("pending suggestion = state %q terminal %v", pendingState, pendingTerminal)
	}
	if acceptedState != SuggestionAccepted || acceptedTerminal != nil {
		t.Fatalf("accepted suggestion changed = state %q terminal %v", acceptedState, acceptedTerminal)
	}
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT status FROM pds_follow_operations WHERE suggestion_id='00000000-0000-0000-0000-000000000212'),
			(SELECT state FROM notification_events WHERE id='00000000-0000-0000-0000-000000000215'),
			(SELECT status FROM push_deliveries WHERE id='00000000-0000-0000-0000-000000000216'),
			(SELECT status FROM instagram_reconciliation_jobs WHERE id='00000000-0000-0000-0000-000000000217')
	`).Scan(&followStatus, &eventState, &deliveryStatus, &jobStatus); err != nil {
		t.Fatalf("inspect invalidated dependents: %v", err)
	}
	if followStatus != "failed" || eventState != "retracted" || deliveryStatus != "cancelled" || jobStatus != "ignored" {
		t.Fatalf("dependents follow=%s event=%s delivery=%s job=%s", followStatus, eventState, deliveryStatus, jobStatus)
	}

	enabled, err := store.UpdateSettings(ctx, alice, AccountSettingsPatch{Discoverable: accountBool(true)})
	if err != nil {
		t.Fatalf("enable discovery: %v", err)
	}
	if !enabled.Discoverable || enabled.State != LinkActive {
		t.Fatalf("enabled account = %+v", enabled)
	}
	if _, err := store.UpdateSettings(ctx, alice, AccountSettingsPatch{Discoverable: accountBool(true)}); err != nil {
		t.Fatalf("idempotent enable: %v", err)
	}
	var reconciliationCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM instagram_reconciliation_jobs
		WHERE owner_did = $1 AND reason = 'instagramLinkDiscoveryEnabled'
	`, alice).Scan(&reconciliationCount); err != nil {
		t.Fatalf("count reconciliation jobs: %v", err)
	}
	if reconciliationCount != 1 {
		t.Fatalf("enable reconciliation jobs = %d, want 1", reconciliationCount)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE instagram_account_links
		SET conflict_pending = true, discoverable = false
		WHERE owner_did = $1 AND state = 'active'
	`, alice); err != nil {
		t.Fatalf("mark conflict pending: %v", err)
	}
	if _, err := store.UpdateSettings(ctx, alice, AccountSettingsPatch{Discoverable: accountBool(true)}); !errors.Is(err, ErrInstagramLinkConflict) {
		t.Fatalf("conflicted enable error = %v, want ErrInstagramLinkConflict", err)
	}
	if _, err := store.UpdateSettings(ctx, alice, AccountSettingsPatch{Discoverable: accountBool(false)}); err != nil {
		t.Fatalf("conflicted privacy disable must remain available: %v", err)
	}
}

func TestAccountStoreRequiresExplicitConflictFreeReactivationAndDiscoveryChoice(t *testing.T) {
	store, pool := newAccountStoreTest(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	bob := syntax.DID("did:plc:synthetic-reactivation-bob")
	carol := syntax.DID("did:plc:synthetic-reactivation-carol")
	inactiveAt := now.Add(-24 * time.Hour)
	insertAccountLink(t, pool, accountLinkFixture{
		ID:                   uuid.MustParse("00000000-0000-0000-0000-000000000221"),
		Owner:                bob,
		State:                LinkMembershipInactive,
		IGSID:                "synthetic-reactivation-igsid-bob",
		Username:             "synthetic.reactivation.bob",
		VerifiedAt:           now.Add(-30 * 24 * time.Hour),
		MembershipInactiveAt: &inactiveAt,
		UpdatedAt:            inactiveAt,
	})
	insertAccountLink(t, pool, accountLinkFixture{
		ID:                   uuid.MustParse("00000000-0000-0000-0000-000000000222"),
		Owner:                carol,
		State:                LinkMembershipInactive,
		IGSID:                "synthetic-reactivation-igsid-carol",
		Username:             "synthetic.reactivation.carol",
		VerifiedAt:           now.Add(-30 * 24 * time.Hour),
		MembershipInactiveAt: &inactiveAt,
		UpdatedAt:            inactiveAt,
	})

	if _, err := store.UpdateSettings(ctx, bob, AccountSettingsPatch{Discoverable: accountBool(true)}); !errors.Is(err, ErrInstagramReactivationRequired) {
		t.Fatalf("implicit reactivation error = %v, want ErrInstagramReactivationRequired", err)
	}
	for _, invalid := range []AccountSettingsPatch{
		{},
		{Reactivate: accountBool(false)},
		{Reactivate: accountBool(true)},
		{Discoverable: accountBool(true), Reactivate: accountBool(false)},
	} {
		if _, err := store.UpdateSettings(ctx, bob, invalid); !errors.Is(err, ErrInvalidInstagramSettings) {
			t.Fatalf("invalid patch %+v error = %v, want ErrInvalidInstagramSettings", invalid, err)
		}
	}

	bobAccount, err := store.UpdateSettings(ctx, bob, AccountSettingsPatch{
		Discoverable: accountBool(false),
		Reactivate:   accountBool(true),
	})
	if err != nil {
		t.Fatalf("reactivate Bob hidden: %v", err)
	}
	if bobAccount.State != LinkActive || bobAccount.Discoverable || bobAccount.ReactivationRequired {
		t.Fatalf("Bob reactivated account = %+v", bobAccount)
	}

	carolAccount, err := store.UpdateSettings(ctx, carol, AccountSettingsPatch{
		Discoverable: accountBool(true),
		Reactivate:   accountBool(true),
	})
	if err != nil {
		t.Fatalf("reactivate Carol discoverable: %v", err)
	}
	if carolAccount.State != LinkActive || !carolAccount.Discoverable || carolAccount.ReactivationRequired {
		t.Fatalf("Carol reactivated account = %+v", carolAccount)
	}

	var (
		bobInactiveAt   *time.Time
		carolInactiveAt *time.Time
		carolJobs       int
	)
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT membership_inactive_at FROM instagram_account_links WHERE owner_did = $1 AND state = 'active'),
			(SELECT membership_inactive_at FROM instagram_account_links WHERE owner_did = $2 AND state = 'active'),
			(SELECT count(*) FROM instagram_reconciliation_jobs WHERE owner_did = $2 AND reason = 'instagramLinkReactivated')
	`, bob, carol).Scan(&bobInactiveAt, &carolInactiveAt, &carolJobs); err != nil {
		t.Fatalf("inspect reactivation: %v", err)
	}
	if bobInactiveAt != nil || carolInactiveAt != nil {
		t.Fatalf("membership inactivity not cleared: Bob=%v Carol=%v", bobInactiveAt, carolInactiveAt)
	}
	if carolJobs != 1 {
		t.Fatalf("Carol reactivation jobs = %d, want 1", carolJobs)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE instagram_account_links
		SET state = 'membershipInactive', discoverable = false,
		    conflict_pending = true, membership_inactive_at = $2
		WHERE owner_did = $1
	`, bob, now); err != nil {
		t.Fatalf("make Bob inactive/conflicted: %v", err)
	}
	if _, err := store.UpdateSettings(ctx, bob, AccountSettingsPatch{
		Discoverable: accountBool(true),
		Reactivate:   accountBool(true),
	}); !errors.Is(err, ErrInstagramLinkConflict) {
		t.Fatalf("conflicted reactivation error = %v, want ErrInstagramLinkConflict", err)
	}

	if _, err := store.UpdateSettings(ctx, syntax.DID("did:plc:synthetic-reactivation-absent"), AccountSettingsPatch{Discoverable: accountBool(false)}); !errors.Is(err, ErrInstagramLinkNotFound) {
		t.Fatalf("absent settings error = %v, want ErrInstagramLinkNotFound", err)
	}
}

func TestAccountStoreRevokesIdempotentlyAndKeepsOnlyTheKeyedCooldownTombstone(t *testing.T) {
	store, pool := newAccountStoreTest(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	alice := syntax.DID("did:plc:synthetic-revoke-alice")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000000231")
	claimID := uuid.MustParse("00000000-0000-0000-0000-000000000232")
	igsid := "synthetic-revoke-private-igsid"
	username := "synthetic.revoke.private"
	insertAccountLink(t, pool, accountLinkFixture{
		ID:              linkID,
		Owner:           alice,
		State:           LinkActive,
		IGSID:           igsid,
		Username:        username,
		Discoverable:    false,
		ConflictPending: true,
		VerifiedAt:      now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Minute),
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_identity_claims (
			id, link_id, owner_did, state, igsid_digest_version,
			igsid_digest, claimed_at, created_at, updated_at
		) SELECT $1, id, owner_did, 'active', igsid_digest_version,
		         igsid_digest, $3, $3, $3
		  FROM instagram_account_links WHERE id = $2
	`, claimID, linkID, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert revoke claim: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, following_count, created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000236', $1, 'active', 'manual', 1, $2, $2
		)
	`, alice, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert revoke import: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, matched, created_at
		) VALUES (
			'00000000-0000-0000-0000-000000000236', 'synthetic.imported.handle', true, $1
		)
	`, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert revoke handle: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_follow_suggestions
			(id, importer_did, target_did, state, reason, created_at, updated_at)
		VALUES
			('00000000-0000-0000-0000-000000000233', 'did:plc:synthetic-revoke-importer-a', $1, 'pending', 'verifiedInstagramFollow', $2, $2),
			('00000000-0000-0000-0000-000000000234', 'did:plc:synthetic-revoke-importer-b', $1, 'accepted', 'verifiedInstagramFollow', $2, $2),
			('00000000-0000-0000-0000-000000000237', $1, 'did:plc:synthetic-revoke-target', 'pending', 'verifiedInstagramFollow', $2, $2)
	`, alice, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert revoke suggestions: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_suggestion_sources (
			suggestion_id, import_id, created_at
		) VALUES (
			'00000000-0000-0000-0000-000000000237',
			'00000000-0000-0000-0000-000000000236',
			$1
		)
	`, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert revoke suggestion source: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs (
			id, owner_did, link_id, reason, status, next_attempt_at, created_at, updated_at
		) VALUES (
			'00000000-0000-0000-0000-000000000235', $1, $2,
			'instagramLinkDiscoveryEnabled', 'queued', $3, $3, $3
		)
	`, alice, linkID, now.Add(-time.Minute)); err != nil {
		t.Fatalf("insert revoke reconciliation: %v", err)
	}

	if err := store.RevokeAccount(ctx, alice); err != nil {
		t.Fatalf("RevokeAccount: %v", err)
	}
	if err := store.RevokeAccount(ctx, alice); err != nil {
		t.Fatalf("repeat RevokeAccount: %v", err)
	}
	if err := store.RevokeAccount(ctx, syntax.DID("did:plc:synthetic-revoke-absent")); err != nil {
		t.Fatalf("absent RevokeAccount: %v", err)
	}
	account, err := store.GetAccount(ctx, alice)
	if err != nil {
		t.Fatalf("GetAccount after revoke: %v", err)
	}
	if account != nil {
		t.Fatalf("revoked account still surfaced: %+v", account)
	}

	var (
		state                InstagramLinkState
		storedIGSID          *string
		storedUsername       *string
		storedNormalized     *string
		discoverable         bool
		conflictPending      bool
		digest               []byte
		revokedAt            *time.Time
		purgeAt              *time.Time
		claimState           string
		claimReleasedAt      *time.Time
		claimAnonymizeAt     *time.Time
		pendingState         InstagramSuggestionState
		acceptedState        InstagramSuggestionState
		importerPendingState InstagramSuggestionState
		reconciliationStatus string
		importCount          int
		handleCount          int
	)
	if err := pool.QueryRow(ctx, `
		SELECT l.state, l.igsid, l.username, l.username_normalized,
		       l.discoverable, l.conflict_pending, l.igsid_digest,
		       l.revoked_at, l.raw_identity_purge_at,
		       c.state, c.released_at, c.anonymize_at,
		       p.state, a.state, importer_pending.state, j.status,
		       (SELECT count(*) FROM instagram_graph_imports WHERE owner_did = $2),
		       (SELECT count(*)
		          FROM instagram_graph_handles
		         WHERE import_id = '00000000-0000-0000-0000-000000000236')
		FROM instagram_account_links l
		JOIN instagram_identity_claims c ON c.link_id = l.id
		JOIN instagram_follow_suggestions p ON p.id = '00000000-0000-0000-0000-000000000233'
		JOIN instagram_follow_suggestions a ON a.id = '00000000-0000-0000-0000-000000000234'
		JOIN instagram_follow_suggestions importer_pending ON importer_pending.id = '00000000-0000-0000-0000-000000000237'
		JOIN instagram_reconciliation_jobs j ON j.id = '00000000-0000-0000-0000-000000000235'
		WHERE l.id = $1
	`, linkID, alice).Scan(
		&state, &storedIGSID, &storedUsername, &storedNormalized,
		&discoverable, &conflictPending, &digest, &revokedAt, &purgeAt,
		&claimState, &claimReleasedAt, &claimAnonymizeAt,
		&pendingState, &acceptedState, &importerPendingState, &reconciliationStatus,
		&importCount, &handleCount,
	); err != nil {
		t.Fatalf("inspect revoked link: %v", err)
	}
	if state != LinkRevoked || storedIGSID != nil || storedUsername != nil || storedNormalized != nil || discoverable || conflictPending {
		t.Fatalf("revoked link retained plaintext/state: state=%q igsid=%v username=%v normalized=%v discoverable=%t conflict=%t", state, storedIGSID, storedUsername, storedNormalized, discoverable, conflictPending)
	}
	if len(digest) != 32 || revokedAt == nil || !revokedAt.Equal(now) || purgeAt == nil || !purgeAt.Equal(now.Add(90*24*time.Hour)) {
		t.Fatalf("revoked tombstone = digest length %d revokedAt %v purgeAt %v", len(digest), revokedAt, purgeAt)
	}
	if claimState != "revoked" || claimReleasedAt == nil || !claimReleasedAt.Equal(now) || claimAnonymizeAt == nil || !claimAnonymizeAt.Equal(now.Add(90*24*time.Hour)) {
		t.Fatalf("revoked claim = state %q released %v anonymize %v", claimState, claimReleasedAt, claimAnonymizeAt)
	}
	if pendingState != SuggestionInvalidated || acceptedState != SuggestionAccepted || reconciliationStatus != "ignored" {
		t.Fatalf("revoke dependents = pending %q accepted %q reconciliation %q", pendingState, acceptedState, reconciliationStatus)
	}
	if importerPendingState != SuggestionInvalidated || importCount != 0 || handleCount != 0 {
		t.Fatalf("revoke importer data = pending %q imports %d handles %d", importerPendingState, importCount, handleCount)
	}

	diagnostic := fmt.Sprintf("error=%v store=%+v", ErrInstagramLinkNotFound, *store)
	for _, private := range []string{igsid, username} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("revoke diagnostic leaked %q: %s", private, diagnostic)
		}
	}
}

type accountLinkFixture struct {
	ID                   uuid.UUID
	Owner                syntax.DID
	State                InstagramLinkState
	IGSID                string
	Username             string
	Discoverable         bool
	ConflictPending      bool
	VerifiedAt           time.Time
	MembershipInactiveAt *time.Time
	UpdatedAt            time.Time
}

func newAccountStoreTest(t *testing.T) (*AccountStore, *pgxpool.Pool) {
	t.Helper()
	var migration strings.Builder
	for _, name := range []string{
		"000021_appview_notifications.up.sql",
		"000022_notification_newness.up.sql",
		"000023_instagram_migration.up.sql",
		"000024_system_notifications.up.sql",
	} {
		contents, err := os.ReadFile("../../migrations/" + name)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		migration.Write(contents)
		migration.WriteByte('\n')
	}
	pool := testdb.WithSchema(t, migration.String())
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	return NewAccountStore(pool, func() time.Time { return now }), pool
}

func insertAccountLink(t *testing.T, pool *pgxpool.Pool, fixture accountLinkFixture) {
	t.Helper()
	ctx := context.Background()
	digest := bytes.Repeat([]byte{fixture.ID[len(fixture.ID)-1]}, 32)
	igsid := fixture.IGSID
	username := fixture.Username
	usernameNormalized := fixture.Username
	if fixture.State == LinkRevoked || fixture.State == LinkSuperseded {
		igsid = ""
		username = ""
		usernameNormalized = ""
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version, igsid_digest,
			username, username_normalized, discoverable, conflict_pending,
			verified_at, membership_inactive_at,
			revoked_at, superseded_at, raw_identity_purge_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, NULLIF($4, ''), 1, $5,
			NULLIF($6, ''), NULLIF($7, ''), $8, $9,
			$10, $11,
			CASE WHEN $3 = 'revoked' THEN $12::timestamptz ELSE NULL END,
			CASE WHEN $3 = 'superseded' THEN $12::timestamptz ELSE NULL END,
			CASE WHEN $3 IN ('revoked', 'superseded') THEN $12::timestamptz + interval '90 days' ELSE NULL END,
			$12::timestamptz, $12::timestamptz
		)
	`, fixture.ID, fixture.Owner, fixture.State, igsid, digest, username, usernameNormalized,
		fixture.Discoverable, fixture.ConflictPending, fixture.VerifiedAt,
		fixture.MembershipInactiveAt, fixture.UpdatedAt); err != nil {
		t.Fatalf("insert account link: %v", err)
	}
}

func accountBool(value bool) *bool {
	return &value
}
