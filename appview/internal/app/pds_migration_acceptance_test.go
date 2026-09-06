package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	atrepo "github.com/bluesky-social/indigo/atproto/repo"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// AT-001 / IT-009: migration changes authority, not the durable owner boundary.
func TestSameDIDPDSMigrationConvergesAndPreservesAccount(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		handle syntax.Handle
	}{
		{name: "unchanged handle", handle: "maker.example"},
		{name: "changed handle", handle: "maker-new.example"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			pool := phase10MigrationPool(t)
			owner := syntax.DID("did:plc:phase10" + strings.ReplaceAll(testCase.name, " ", ""))
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			if _, err := pool.Exec(ctx, `
				INSERT INTO owner_lifecycles(
					owner_did,state,generation,auth_epoch,transition_reason,
					transitioned_at,created_at,updated_at
				) VALUES($1,'active',1,1,'phase10 fixture',now(),now(),now())
			`, owner); err != nil {
				t.Fatalf("seed migrated owner lifecycle: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO craftsky_profiles(did,crafts,record_cid,indexed_at)
				VALUES($1,'{}','bafyprofile',now())
			`, owner); err != nil {
				t.Fatalf("seed migrated profile: %v", err)
			}
			seedPhase10PrivateState(t, pool, owner)

			owners := phase10OwnerStore(t, pool)
			store, err := ingestion.NewStore(pool, time.Now)
			if err != nil {
				t.Fatalf("new ingestion store: %v", err)
			}
			service, err := ingestion.NewService(ingestion.ServiceConfig{
				Store: store, Lifecycles: owners,
				ProfileParticipant: func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error { return nil },
			})
			if err != nil {
				t.Fatalf("new ingestion service: %v", err)
			}
			dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(pool, logger, nil, nil, nil)

			oldKey := repositorySnapshotTestKey(t, 21)
			newKey := repositorySnapshotTestKey(t, 22)
			oldUpdated := phase10Post("old version")
			oldDeleted := phase10Post("removed on B")
			phase10SeedPost(t, store, service, dispatcher, owner, "updated", "3aaaaaaaaaaa2", oldUpdated)
			phase10SeedPost(t, store, service, dispatcher, owner, "deleted", "3aaaaaaaaaaa2", oldDeleted)

			snapshotCAR := buildRepositorySnapshotCARWithRecords(t, owner, newKey, "3bbbbbbbbbbb2", []repositorySnapshotFixtureRecord{
				{path: "social.craftsky.feed.post/updated", record: phase10Post("authoritative update")},
				{path: "social.craftsky.feed.post/created", record: phase10Post("missed live create")},
			})
			oldCAR := buildRepositorySnapshotCARWithRecords(t, owner, oldKey, "3aaaaaaaaaaa2", []repositorySnapshotFixtureRecord{
				{path: "social.craftsky.feed.post/updated", record: oldUpdated},
				{path: "social.craftsky.feed.post/deleted", record: oldDeleted},
			})
			if oldCAR.root.Equals(snapshotCAR.root) {
				t.Fatal("rotated-key reset chain unexpectedly retained the old root")
			}
			oldPublic, err := oldKey.PublicKey()
			if err != nil {
				t.Fatal(err)
			}
			newPublic, err := newKey.PublicKey()
			if err != nil {
				t.Fatal(err)
			}
			if oldPublic.Multibase() == newPublic.Multibase() {
				t.Fatal("migration fixture did not rotate its signing key")
			}
			commit, _, err := atrepo.LoadRepoFromCAR(ctx, bytes.NewReader(snapshotCAR.data))
			if err != nil || commit.Prev != nil {
				t.Fatalf("load reset-chain repository: prev=%v err=%v", commit.Prev, err)
			}

			var oldRequests, newWrites, staleCredentialLeaks atomic.Int32
			oldPDS := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				oldRequests.Add(1)
			}))
			t.Cleanup(oldPDS.Close)
			newPDS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.Header.Get("Authorization"), "old-parent-token") || strings.Contains(r.Header.Get("DPoP"), "old-parent-token") {
					staleCredentialLeaks.Add(1)
				}
				switch r.URL.Path {
				case "/xrpc/com.atproto.sync.getRepo":
					if r.URL.Query().Get("did") != owner.String() {
						t.Errorf("getRepo DID = %q", r.URL.Query().Get("did"))
					}
					w.Header().Set("Content-Type", "application/vnd.ipld.car")
					_, _ = w.Write(snapshotCAR.data)
				case "/xrpc/com.atproto.repo.putRecord":
					newWrites.Add(1)
					if got := r.Header.Get("Authorization"); got != "Bearer new-parent-token" {
						t.Errorf("new PDS authorization = %q", got)
					}
					w.WriteHeader(http.StatusOK)
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(newPDS.Close)

			issuerA := oldPDS.URL
			issuerB := newPDS.URL
			authStore, oauthApp := phase10AuthStore(t, pool, owners)
			phase10InsertParent(t, pool, owner, "parent-a", oldPDS.URL, issuerA, "old-parent-token")
			if _, err := pool.Exec(ctx, `
				INSERT INTO craftsky_sessions(
					token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
					last_seen_at,idle_expires_at
				) VALUES(decode(repeat('01',32),'hex'),$1,'parent-a','active',1,now(),now()+interval '1 day')
			`, owner); err != nil {
				t.Fatalf("seed old child: %v", err)
			}
			currentAuthority := phase10AuthorityVerifier{authority: auth.OAuthAuthority{
				DID: owner, PDSOrigin: newPDS.URL, IssuerOrigin: issuerB,
			}}
			oldCoordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
				App: oauthApp, Store: authStore, Owners: owners, AuthorityVerifier: currentAuthority,
				OperationTimeout: time.Second,
			})
			if err != nil {
				t.Fatalf("new old-parent coordinator: %v", err)
			}
			staleEffectCalled := false
			err = oldCoordinator.WithActiveSession(ctx, owner, "parent-a", func(context.Context, *oauth.ClientSession) error {
				staleEffectCalled = true
				return nil
			})
			if staleEffectCalled || !errors.Is(err, auth.ErrPDSSessionExpired) {
				t.Fatalf("stale protected effect called=%t err=%v", staleEffectCalled, err)
			}

			// This transaction is the fresh-auth handoff commit boundary: activate the
			// new exact parent and make repair durable before any worker side effect.
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			phase10InsertParentTx(t, tx, owner, "parent-b", newPDS.URL, issuerB, "new-parent-token")
			if err := store.EnqueueRepositoryJobTx(ctx, tx, owner, ingestion.RepositoryJobPDSReconcile); err != nil {
				_ = tx.Rollback(ctx)
				t.Fatalf("enqueue durable repair with fresh parent: %v", err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("commit fresh auth: %v", err)
			}

			directory := repositorySnapshotDirectoryFunc(func(context.Context, syntax.DID) (*identity.Identity, error) {
				return &identity.Identity{
					DID: owner, Handle: testCase.handle,
					Keys: map[string]identity.VerificationMethod{"atproto": repositorySnapshotVerificationMethod(t, newKey)},
					Services: map[string]identity.ServiceEndpoint{"atproto_pds": {
						Type: "AtprotoPersonalDataServer", URL: newPDS.URL,
					}},
				}, nil
			})
			fetcher, err := ingestion.NewRepositorySnapshotFetcher(directory, newPDS.Client())
			if err != nil {
				t.Fatalf("new snapshot fetcher: %v", err)
			}
			repair, err := ingestion.NewRepositoryRepair(ingestion.RepositoryRepairConfig{
				Store: store, Ingestor: service, Projector: dispatcher.Project,
			})
			if err != nil {
				t.Fatalf("new repository repair: %v", err)
			}
			claims, err := store.ClaimRepositoryJobs(ctx, ingestion.RepositoryClaimRequest{
				Worker: "phase10", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
			})
			if err != nil || len(claims) != 1 || claims[0].Kind != ingestion.RepositoryJobPDSReconcile {
				t.Fatalf("claim durable repair = %+v err=%v", claims, err)
			}
			recorder := observability.NewInMemoryMetricRecorder()
			observer := observability.New(observability.Config{MetricRecorder: recorder})
			handler := newTapRepositoryJobHandler(phase10RepositoryTracker{}, fetcher, dispatcher, repair, observer)
			if err := store.RunRepositoryJob(ctx, claims[0], handler); err != nil {
				t.Fatalf("run migration repair worker: %v", err)
			}

			assertPhase10MigrationState(t, pool, owner)
			requestsBeforeLocalRead := newWrites.Load()
			var texts []string
			rows, err := pool.Query(ctx, `SELECT text FROM craftsky_posts WHERE did=$1 ORDER BY text`, owner)
			if err != nil {
				t.Fatalf("local AppView read: %v", err)
			}
			for rows.Next() {
				var text string
				if err := rows.Scan(&text); err != nil {
					t.Fatal(err)
				}
				texts = append(texts, text)
			}
			rows.Close()
			if fmt.Sprint(texts) != "[authoritative update missed live create]" || newWrites.Load() != requestsBeforeLocalRead {
				t.Fatalf("local read texts=%v new writes before/after=%d/%d", texts, requestsBeforeLocalRead, newWrites.Load())
			}

			newCoordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
				App: oauthApp, Store: authStore, Owners: owners, AuthorityVerifier: currentAuthority,
				Observer:         observer,
				OperationTimeout: time.Second,
			})
			if err != nil {
				t.Fatal(err)
			}
			err = newCoordinator.WithActiveSession(ctx, owner, "parent-b", func(ctx context.Context, session *oauth.ClientSession) error {
				body := bytes.NewBufferString(`{"repo":"` + owner.String() + `"}`)
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, session.Data.HostURL+"/xrpc/com.atproto.repo.putRecord", body)
				if err != nil {
					return err
				}
				req.Header.Set("Authorization", "Bearer "+session.Data.AccessToken)
				response, err := newPDS.Client().Do(req)
				if err != nil {
					return err
				}
				defer response.Body.Close()
				if response.StatusCode != http.StatusOK {
					return fmt.Errorf("new PDS write status %d", response.StatusCode)
				}
				return nil
			})
			if err != nil || oldRequests.Load() != 0 || newWrites.Load() != 1 || staleCredentialLeaks.Load() != 0 {
				t.Fatalf("post-migration write err=%v old requests=%d new writes=%d stale leaks=%d", err, oldRequests.Load(), newWrites.Load(), staleCredentialLeaks.Load())
			}
			phase10AssertMigrationMetrics(t, recorder.Calls())
		})
	}
}

func phase10AssertMigrationMetrics(t *testing.T, calls []observability.MetricCall) {
	t.Helper()
	var snapshotVerified, authorityVerified bool
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("unsafe migration metric %#v: %v", call, err)
		}
		switch call.Name {
		case "craftsky_appview_repository_snapshot_verifications_total":
			snapshotVerified = call.Attributes["result"] == "success"
		case "craftsky_appview_authority_verifications_total":
			authorityVerified = call.Attributes["result"] == "success"
		}
	}
	if !snapshotVerified || !authorityVerified {
		t.Fatalf("migration metrics snapshot=%v authority=%v calls=%#v", snapshotVerified, authorityVerified, calls)
	}
}

type phase10AuthorityVerifier struct{ authority auth.OAuthAuthority }

func (verifier phase10AuthorityVerifier) ResolveCurrent(context.Context, syntax.DID) (auth.OAuthAuthority, error) {
	return verifier.authority, nil
}

type phase10EndpointValidator struct{}

func (phase10EndpointValidator) ValidateOrigin(_ context.Context, raw string) (*url.URL, error) {
	return url.Parse(raw)
}

func (phase10EndpointValidator) ValidateOAuthEndpoint(_ context.Context, _ string, endpoint string) (*url.URL, error) {
	return url.Parse(endpoint)
}

type phase10RepositoryTracker struct{}

func (phase10RepositoryTracker) AddRepo(context.Context, syntax.DID) error { return nil }

func phase10Post(text string) *craftskylex.FeedPost {
	return &craftskylex.FeedPost{
		LexiconTypeID: "social.craftsky.feed.post",
		Text:          text, CreatedAt: "2026-09-03T12:00:00Z",
	}
}

func phase10MigrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	var schema string
	if err := pool.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema); err != nil {
		t.Fatalf("read isolated schema: %v", err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire migration connection: %v", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SET search_path TO "+pgx.Identifier{schema}.Sanitize()+", public"); err != nil {
		t.Fatalf("set migration search path: %v", err)
	}
	paths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	for _, path := range paths {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := conn.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
	if _, err := conn.Exec(ctx, `
		CREATE TABLE phase10_private_state(
			owner_did TEXT NOT NULL REFERENCES owner_lifecycles(owner_did),
			kind TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY(owner_did,kind)
		)
	`); err != nil {
		t.Fatalf("create private-state acceptance fixture: %v", err)
	}
	return pool
}

func phase10OwnerStore(t *testing.T, pool *pgxpool.Pool) *ownerlifecycle.Store {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func seedPhase10PrivateState(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) {
	t.Helper()
	for _, kind := range []string{"draft", "saved", "notification", "moderation", "relationship", "scheduled"} {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO phase10_private_state(owner_did,kind,value) VALUES($1,$2,'preserved')
		`, owner, kind); err != nil {
			t.Fatalf("seed %s state: %v", kind, err)
		}
	}
}

func phase10AuthStore(t *testing.T, pool *pgxpool.Pool, owners *ownerlifecycle.Store) (*auth.PostgresAuthStore, *oauth.ClientApp) {
	t.Helper()
	config := realFlowStoreConfig()
	config.OwnerLifecycles = owners
	config.EndpointValidator = phase10EndpointValidator{}
	store := auth.NewPostgresAuthStore(pool, config)
	publicConfig := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	return store, oauth.NewClientApp(&publicConfig, store)
}

func phase10InsertParent(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, sessionID, pds, issuer, token string) {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	phase10InsertParentTx(t, tx, owner, sessionID, pds, issuer, token)
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func phase10InsertParentTx(t *testing.T, tx pgx.Tx, owner syntax.DID, sessionID, pds, issuer, token string) {
	t.Helper()
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(oauth.ClientSessionData{
		AccountDID: owner, SessionID: sessionID, HostURL: pds, AuthServerURL: issuer,
		AuthServerTokenEndpoint: issuer + "/oauth/token", Scopes: []string{"atproto"},
		AccessToken: token, RefreshToken: token + "-refresh", DPoPPrivateKeyMultibase: key.Multibase(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			row_version,absolute_expires_at,created_at,updated_at
		) VALUES($1,$2,$3,'active',1,1,1,now()+interval '1 day',now(),now())
	`, owner, sessionID, data); err != nil {
		t.Fatalf("insert OAuth parent %s: %v", sessionID, err)
	}
}

func phase10SeedPost(
	t *testing.T,
	store *ingestion.Store,
	service *ingestion.Service,
	dispatcher *index.TransactionalDispatcher,
	owner syntax.DID,
	rkey syntax.RecordKey,
	revision syntax.TID,
	record *craftskylex.FeedPost,
) {
	t.Helper()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	event := tap.Event{
		ID:  uint64(digest[0]) + 1,
		URI: syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + rkey.String()),
		DID: owner, Collection: "social.craftsky.feed.post", Rkey: rkey, Rev: revision,
		CID: syntax.CID(fmt.Sprintf("bafyphase10%x", digest[:8])), Action: "create", Record: raw,
	}
	if outcome, err := service.IngestRecord(context.Background(), event); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("seed post %s: outcome=%+v err=%v", rkey, outcome, err)
	}
	claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
		Worker: "phase10-seed", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim seed projection %s: claims=%+v err=%v", rkey, claims, err)
	}
	if err := store.Project(context.Background(), claims[0], dispatcher.Project); err != nil {
		t.Fatalf("project seed post %s: %v", rkey, err)
	}
}

func assertPhase10MigrationState(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) {
	t.Helper()
	var owners, profiles, privateRows, posts, wrongPostOwners int
	var lifecycle, scheduledState string
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM owner_lifecycles),
			(SELECT count(*) FROM craftsky_profiles),
			(SELECT count(*) FROM phase10_private_state WHERE owner_did=$1),
			(SELECT count(*) FROM craftsky_posts WHERE did=$1),
			(SELECT count(*) FROM craftsky_posts WHERE uri LIKE 'at://' || $1 || '/%' AND did<>$1),
			(SELECT state FROM owner_lifecycles WHERE owner_did=$1),
			(SELECT value FROM phase10_private_state WHERE owner_did=$1 AND kind='scheduled')
	`, owner).Scan(&owners, &profiles, &privateRows, &posts, &wrongPostOwners, &lifecycle, &scheduledState); err != nil {
		t.Fatalf("read migration state: %v", err)
	}
	if owners != 1 || profiles != 1 || privateRows != 6 || posts != 2 || wrongPostOwners != 0 || lifecycle != "active" || scheduledState != "preserved" {
		t.Fatalf("migration state owners=%d profiles=%d private=%d posts=%d wrongOwners=%d lifecycle=%s scheduled=%s",
			owners, profiles, privateRows, posts, wrongPostOwners, lifecycle, scheduledState)
	}
	var oldParent, oldChild, newParent string
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='parent-a'),
			(SELECT lifecycle_state FROM craftsky_sessions WHERE account_did=$1 AND oauth_session_id='parent-a'),
			(SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='parent-b')
	`, owner).Scan(&oldParent, &oldChild, &newParent); err != nil {
		t.Fatalf("read parent states: %v", err)
	}
	if oldParent != "revocation_pending" || oldChild != "revoked" || newParent != "active" {
		t.Fatalf("parent states old/child/new=%s/%s/%s", oldParent, oldChild, newParent)
	}
}
