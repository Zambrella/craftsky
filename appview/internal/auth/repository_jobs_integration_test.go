package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

type authRepositoryJobAdapter struct {
	store *ingestion.Store
}

func (adapter authRepositoryJobAdapter) EnqueueRepositoryJobTx(
	ctx context.Context,
	tx pgx.Tx,
	did syntax.DID,
	kind auth.RepositoryJobKind,
) error {
	return adapter.store.EnqueueRepositoryJobTx(ctx, tx, did, ingestion.RepositoryJobKind(kind))
}

func TestAuthAndSourceUncertaintyDurablyCoalesceRepositoryJobsAcrossRestart(t *testing.T) {
	cases := []struct {
		name          string
		ownerActive   bool
		priorParent   bool
		wantAddRepo   int
		wantReconcile int
	}{
		{name: "onboarding", wantAddRepo: 1},
		{name: "ordinary sign-in existing DID", ownerActive: true, wantAddRepo: 1},
		{name: "same-DID new-parent reauthorization", ownerActive: true, priorParent: true, wantAddRepo: 1, wantReconcile: 1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pool := withRepositoryJobAuthSchema(t)
			now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
			clock := func() time.Time { return now }
			store, err := ingestion.NewStore(pool, clock)
			if err != nil {
				t.Fatalf("new ingestion store: %v", err)
			}
			owners := newAuthOwnerStore(t, pool)
			children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
				Inactivity: 30 * 24 * time.Hour, ActivityWriteInterval: 15 * time.Minute,
			})
			if err != nil {
				t.Fatalf("new child session store: %v", err)
			}
			handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
				Pool: pool, Owners: owners, Sessions: children,
				RepositoryJobs: authRepositoryJobAdapter{store: store},
				ExchangeTTL:    5 * time.Minute, ConfirmationTTL: 2 * time.Minute,
				ReceiptKey: []byte("0123456789abcdef0123456789abcdef"), ReceiptKeyVersion: 1,
				Now: clock,
			})
			if err != nil {
				t.Fatalf("new handoff service: %v", err)
			}
			owner := syntax.DID("did:plc:" + repositoryJobTestSlug(testCase.name))
			if testCase.ownerActive {
				if err := owners.WithOnboardingAuth(context.Background(), owner, func(context.Context, ownerlifecycle.Lifecycle) error {
					return nil
				}); err != nil {
					t.Fatalf("initialize existing owner: %v", err)
				}
				if _, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
					Owner: owner, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileCreated",
				}); err != nil {
					t.Fatalf("activate existing owner: %v", err)
				}
			}
			if testCase.priorParent {
				seedPriorOAuthParent(t, pool, owner, now)
			}

			token, receiptID := prepareRepositoryJobHandoff(t, pool, owners, handoffs, owner)
			if !testCase.ownerActive {
				if _, err := owners.Transition(context.Background(), ownerlifecycle.TransitionRequest{
					Owner: owner, ExpectedGeneration: 1, To: ownerlifecycle.StateActive, Reason: "profileCreated",
				}); err != nil {
					t.Fatalf("complete onboarding profile: %v", err)
				}
			}
			if err := handoffs.Confirm(context.Background(), token, receiptID, "repository-job-device"); err != nil {
				t.Fatalf("confirm handoff: %v", err)
			}
			if err := handoffs.Confirm(context.Background(), token, receiptID, "repository-job-device"); err != nil {
				t.Fatalf("duplicate handoff confirmation: %v", err)
			}
			assertRepositoryJobKinds(t, pool, owner, testCase.wantAddRepo, testCase.wantReconcile)

			claims, err := store.ClaimRepositoryJobs(context.Background(), ingestion.RepositoryClaimRequest{
				Worker: "tap-before-restart", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 8,
			})
			addRepoClaim, ok := repositoryClaimOfKind(claims, ingestion.RepositoryJobTapAddRepo)
			if err != nil || !ok {
				t.Fatalf("initial Tap claim=%+v err=%v", claims, err)
			}
			remoteErr := errors.New("Tap AddRepo unavailable")
			if err := store.RunRepositoryJob(context.Background(), addRepoClaim, func(context.Context, ingestion.RepositoryClaim) (string, error) {
				return "", remoteErr
			}); !errors.Is(err, remoteErr) {
				t.Fatalf("failed Tap attempt: %v", err)
			}

			now = now.Add(2 * time.Second)
			restarted, err := ingestion.NewStore(pool, clock)
			if err != nil {
				t.Fatalf("restart ingestion store: %v", err)
			}
			claims, err = restarted.ClaimRepositoryJobs(context.Background(), ingestion.RepositoryClaimRequest{
				Worker: "tap-after-restart", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
			})
			if err != nil || len(claims) != 1 || claims[0].Kind != ingestion.RepositoryJobTapAddRepo || claims[0].Attempts != 2 {
				t.Fatalf("restarted Tap claim=%+v err=%v", claims, err)
			}
			remoteCalls := 0
			if err := restarted.RunRepositoryJob(context.Background(), claims[0], func(_ context.Context, claim ingestion.RepositoryClaim) (string, error) {
				remoteCalls++
				if claim.DID != owner {
					t.Fatalf("Tap claim owner=%s want=%s", claim.DID, owner)
				}
				return "", nil
			}); err != nil {
				t.Fatalf("recover Tap tracking: %v", err)
			}
			if remoteCalls != 1 {
				t.Fatalf("recovered Tap effects=%d want=1", remoteCalls)
			}
			assertRepositoryJobKinds(t, pool, owner, testCase.wantAddRepo, testCase.wantReconcile)
		})
	}

	t.Run("source-order uncertainty", func(t *testing.T) {
		pool := withRepositoryJobAuthSchema(t)
		store, err := ingestion.NewStore(pool, time.Now)
		if err != nil {
			t.Fatalf("new ingestion store: %v", err)
		}
		owner := syntax.DID("did:plc:source-order-uncertain")
		if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')`, owner); err != nil {
			t.Fatalf("seed profile: %v", err)
		}
		event := tap.Event{
			ID: 7, URI: "at://did:plc:source-order-uncertain/social.craftsky.feed.post/one",
			DID: owner, Collection: "social.craftsky.feed.post", Rkey: "one",
			Rev: "3aaaaaaaaaaaz", CID: "bafy-first", Action: "create",
			Record: json.RawMessage(`{"text":"first","createdAt":"2026-09-03T12:00:00Z"}`),
		}
		if _, err := store.IngestRecord(context.Background(), event); err != nil {
			t.Fatalf("ingest initial source: %v", err)
		}
		event.Record = json.RawMessage(`{"text":"conflict","createdAt":"2026-09-03T12:00:00Z"}`)
		outcome, err := store.IngestRecord(context.Background(), event)
		if err != nil || outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
			t.Fatalf("uncertain outcome=%+v err=%v", outcome, err)
		}
		if _, err := store.IngestRecord(context.Background(), event); err != nil {
			t.Fatalf("duplicate uncertain source: %v", err)
		}
		assertRepositoryJobKinds(t, pool, owner, 0, 1)
	})
}

func repositoryClaimOfKind(claims []ingestion.RepositoryClaim, kind ingestion.RepositoryJobKind) (ingestion.RepositoryClaim, bool) {
	for _, claim := range claims {
		if claim.Kind == kind {
			return claim, true
		}
	}
	return ingestion.RepositoryClaim{}, false
}

func withRepositoryJobAuthSchema(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := withAuthSchema(t)
	if _, err := pool.Exec(context.Background(), `ALTER TABLE craftsky_profiles ADD COLUMN record_cid TEXT`); err != nil {
		t.Fatalf("extend profile fixture: %v", err)
	}
	for _, path := range []string{
		"../../migrations/000045_tap_ingestion_durability.up.sql",
		"../../migrations/000058_tap_projection_generation_column.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read Tap durability migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply Tap durability migration %s: %v", path, err)
		}
	}
	return pool
}

func prepareRepositoryJobHandoff(
	t *testing.T,
	pool *pgxpool.Pool,
	owners *ownerlifecycle.Store,
	handoffs *auth.HandoffService,
	owner syntax.DID,
) (string, uuid.UUID) {
	t.Helper()
	oauthStore := auth.NewPostgresAuthStore(pool, func() auth.StoreConfig {
		config := testStoreConfig()
		config.OwnerLifecycles = owners
		return config
	}())
	state := "parent-" + repositoryJobTestSlug(owner.String())
	var code string
	err := owners.WithOnboardingAuth(context.Background(), owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		requestContext := auth.WithLoginAuthRequest(
			authCtx, owner, authority.Generation, authority.AuthEpoch,
			"https://pds.example.com", "https://auth.example.com",
			auth.HandoffVerifiedLink, "repository-job-device", "",
		)
		if err := oauthStore.SaveAuthRequestInfo(requestContext, oauth.AuthRequestData{
			State: state, RequestURI: "urn:request:" + state, AuthServerURL: "https://auth.example.com",
		}); err != nil {
			return err
		}
		attemptID, err := oauthStore.BeginExchange(authCtx, state)
		if err != nil {
			return err
		}
		attempt := auth.CallbackAttempt{
			State: state, AttemptID: attemptID, Owner: owner,
			OwnerGeneration: authority.Generation, AuthEpoch: authority.AuthEpoch,
			Purpose: auth.LoginOAuthPurpose,
		}
		callbackContext := auth.WithCallbackAttempt(authCtx, attempt)
		if err := oauthStore.SaveSession(callbackContext, validOAuthSession(owner, state)); err != nil {
			return err
		}
		code, err = handoffs.CreateExchange(callbackContext, attempt, "alice.example", "repository-job-device")
		return err
	})
	if err != nil {
		t.Fatalf("prepare callback handoff: %v", err)
	}
	exchange, err := handoffs.Exchange(context.Background(), code, "repository-job-device")
	if err != nil {
		t.Fatalf("exchange handoff: %v", err)
	}
	if exchange.Token == "" || exchange.ReceiptID == uuid.Nil {
		t.Fatalf("invalid handoff exchange=%+v", exchange)
	}
	return exchange.Token, exchange.ReceiptID
}

func seedPriorOAuthParent(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, now time.Time) {
	t.Helper()
	data, err := json.Marshal(validOAuthSession(owner, "prior-parent"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,'prior-parent',$2,'active',1,1,1,$3,$4,$4)
	`, owner, data, now.Add(time.Hour), now); err != nil {
		t.Fatalf("seed prior OAuth parent: %v", err)
	}
}

func assertRepositoryJobKinds(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, wantAddRepo, wantReconcile int) {
	t.Helper()
	var addRepo, reconcile int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FILTER (WHERE job_kind='tap_add_repo'),
		       count(*) FILTER (WHERE job_kind='pds_reconcile')
		FROM tap_repository_jobs WHERE did=$1
	`, owner).Scan(&addRepo, &reconcile); err != nil {
		t.Fatalf("count repository jobs: %v", err)
	}
	if addRepo != wantAddRepo || reconcile != wantReconcile {
		t.Fatalf("repository jobs addRepo=%d reconcile=%d want=%d/%d", addRepo, reconcile, wantAddRepo, wantReconcile)
	}
}

func repositoryJobTestSlug(value string) string {
	result := make([]byte, 0, len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' {
			result = append(result, character)
		} else {
			result = append(result, '-')
		}
	}
	return string(result)
}
