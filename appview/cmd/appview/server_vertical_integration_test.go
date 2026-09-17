package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const (
	verticalViewer syntax.DID = "did:plc:verticalviewer"
	verticalAuthor syntax.DID = "did:plc:verticalauthor"
)

type verticalAuthService struct{}

func (verticalAuthService) Authenticate(context.Context, string) (auth.AuthInfo, error) {
	return auth.AuthInfo{DID: verticalViewer, SessionID: "vertical-session"}, nil
}

func (service verticalAuthService) AuthenticateRecovery(ctx context.Context, token string) (auth.AuthInfo, error) {
	return service.Authenticate(ctx, token)
}

type verticalPDSBoundary struct {
	auth.PDSClient
	lifecycles *ownerlifecycle.Store
}

func (boundary verticalPDSBoundary) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	return boundary.lifecycles.WithActiveEffects(ctx, expected, func(effectCtx context.Context) error {
		return operation(effectCtx, boundary.PDSClient)
	})
}

func TestNewServer_MigratedPostgresVerticalSlices(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did, record_cid) VALUES
			($1, 'viewer-profile-cid'),
			($2, 'author-profile-cid')
	`, verticalViewer, verticalAuthor); err != nil {
		t.Fatalf("seed migrated profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did, state, generation, auth_epoch, transition_reason,
			transitioned_at, created_at, updated_at
		) VALUES
			($1, 'active', 1, 1, 'verticalTest', now(), now(), now()),
			($2, 'active', 1, 1, 'verticalTest', now(), now(), now());
	`, verticalViewer, verticalAuthor); err != nil {
		t.Fatalf("seed migrated owner lifecycles: %v", err)
	}
	languagePreferences := languages.NewStore(pool)
	if _, err := languagePreferences.Initialize(ownerlifecycle.WithExpectedGeneration(ctx, 1), verticalViewer, languages.Preferences{
		PrimaryLanguage:  "en",
		ContentLanguages: []string{"en"},
	}); err != nil {
		t.Fatalf("initialize migrated language preferences: %v", err)
	}

	followRecord := json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"did:plc:verticalauthor","createdAt":"2026-09-17T10:00:00Z"}`)
	if err := index.NewBlueskyFollow(pool).Handle(ctx, tap.Event{
		URI:        "at://did:plc:verticalviewer/app.bsky.graph.follow/vertical-follow",
		CID:        "bafy-vertical-follow",
		DID:        verticalViewer,
		Collection: "app.bsky.graph.follow",
		Rkey:       "vertical-follow",
		Action:     "create",
		Record:     followRecord,
	}); err != nil {
		t.Fatalf("index follow against migrated schema: %v", err)
	}
	postRecord := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"from migrated indexer","langs":["en"],"sponsored":false,"createdAt":"2026-09-17T10:00:00Z"}`)
	if err := index.NewCraftskyPost(pool, logger).Handle(ctx, tap.Event{
		URI:        "at://did:plc:verticalauthor/social.craftsky.feed.post/vertical-post",
		CID:        "bafy-vertical-post",
		DID:        verticalAuthor,
		Collection: "social.craftsky.feed.post",
		Rkey:       "vertical-post",
		Action:     "create",
		Record:     postRecord,
	}); err != nil {
		t.Fatalf("index post against migrated schema: %v", err)
	}
	assertFollowLookupUsesCompositeIndex(t, pool, verticalViewer, verticalAuthor)

	pds := newVerticalPDSServer(t)
	pdsClient := newVerticalIndigoPDSClient(t, pds.server.URL)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("new owner lifecycle store: %v", err)
	}
	effects, err := pdseffects.NewExecutorFactory(
		lifecycles,
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return verticalPDSBoundary{PDSClient: pdsClient, lifecycles: lifecycles}, nil
		},
		2*time.Second,
		time.Now,
	)
	if err != nil {
		t.Fatalf("new PDS effect factory: %v", err)
	}

	server := NewServer(ctx, &app.Deps{
		Config: app.Config{
			Env:            app.EnvProd,
			AllowedOrigins: []string{"https://app.craftsky.social"},
		},
		Logger:      logger,
		DB:          pool,
		AuthService: verticalAuthService{},
		CraftskySessionStore: auth.NewCraftskySessionStore(
			pool,
			15*time.Minute,
		),
		Observability:       observability.New(observability.Config{Env: "test"}),
		HandleResolver:      serverStubResolver{handle: "vertical.example"},
		ProfileStore:        api.NewProfileStore(pool),
		FollowStore:         api.NewFollowStore(pool),
		LanguagePreferences: languagePreferences,
		ModerationCases:     moderation.NewStore(pool),
		OwnerLifecycles:     lifecycles,
		NewPDSEffects:       effects,
	})

	t.Run("authenticated timeline", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/v1/feed/timeline", nil)
		authorizeVerticalRequest(request)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("timeline status = %d, want 200; body=%s logs=%s", response.Code, response.Body.String(), logs.String())
		}
		var page struct {
			Items []struct {
				Post struct {
					Text string `json:"text"`
				} `json:"post"`
			} `json:"items"`
		}
		if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
			t.Fatalf("decode timeline: %v", err)
		}
		if len(page.Items) != 1 || page.Items[0].Post.Text != "from migrated indexer" {
			t.Fatalf("timeline items = %#v", page.Items)
		}
	})

	t.Run("AppView-mediated PDS write", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"routed write","sponsored":false}`))
		request.Header.Set("Content-Type", "application/json")
		authorizeVerticalRequest(request)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("create status = %d, want 201; body=%s pds_puts=%d pds_gets=%d logs=%s", response.Code, response.Body.String(), pds.puts, pds.gets, logs.String())
		}
		if pds.puts != 1 || pds.gets != 1 {
			t.Fatalf("fake PDS put/get calls = %d/%d, want 1/1", pds.puts, pds.gets)
		}
		var attempts int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM owner_effect_attempts WHERE owner_did=$1`, verticalViewer).Scan(&attempts); err != nil {
			t.Fatalf("count durable PDS attempts: %v", err)
		}
		if attempts != 1 {
			t.Fatalf("durable PDS attempts = %d, want 1", attempts)
		}
	})
}

func authorizeVerticalRequest(request *http.Request) {
	request.Header.Set("Authorization", "Bearer vertical-token")
	request.Header.Set("X-Craftsky-Device-Id", "vertical-device")
}

func assertFollowLookupUsesCompositeIndex(t *testing.T, pool *pgxpool.Pool, follower, subject syntax.DID) {
	t.Helper()
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin explain transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(t.Context(), `SET LOCAL enable_seqscan = off`); err != nil {
		t.Fatalf("disable sequential scans for index eligibility check: %v", err)
	}
	rows, err := tx.Query(t.Context(), `
		EXPLAIN (COSTS OFF)
		SELECT uri, did, rkey, cid, subject_did, created_at
		FROM atproto_follows
		WHERE did = $1 AND subject_did = $2
		  AND NOT appview_owner_is_terminal(did)
		  AND NOT appview_owner_is_terminal(subject_did)
		LIMIT 1
	`, follower, subject)
	if err != nil {
		t.Fatalf("explain follow lookup: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan follow lookup plan: %v", err)
		}
		plan.WriteString(line)
		plan.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read follow lookup plan: %v", err)
	}
	if !strings.Contains(plan.String(), "atproto_follows_did_subject_did") {
		t.Fatalf("follow lookup plan does not use migrated composite index:\n%s", plan.String())
	}
}

type verticalPDSServer struct {
	server *httptest.Server
	puts   int
	gets   int
}

func newVerticalPDSServer(t *testing.T) *verticalPDSServer {
	t.Helper()
	fake := &verticalPDSServer{}
	var record json.RawMessage
	var repo, collection, rkey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/xrpc/com.atproto.repo.putRecord":
			fake.puts++
			var body struct {
				Repo       string          `json:"repo"`
				Collection string          `json:"collection"`
				Rkey       string          `json:"rkey"`
				Record     json.RawMessage `json:"record"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var value struct {
				Text string `json:"text"`
			}
			if request.Method != http.MethodPost || body.Repo != verticalViewer.String() ||
				body.Collection != "social.craftsky.feed.post" || json.Unmarshal(body.Record, &value) != nil ||
				value.Text != "routed write" {
				http.Error(w, "unexpected putRecord request", http.StatusBadRequest)
				return
			}
			repo, collection, rkey, record = body.Repo, body.Collection, body.Rkey, append(record[:0], body.Record...)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{}`)
		case "/xrpc/com.atproto.repo.getRecord":
			fake.gets++
			if request.Method != http.MethodGet || request.URL.Query().Get("repo") != repo ||
				request.URL.Query().Get("collection") != collection || request.URL.Query().Get("rkey") != rkey {
				http.Error(w, "unexpected getRecord request", http.StatusBadRequest)
				return
			}
			if len(record) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"RecordNotFound","message":"missing"}`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"uri":   "at://" + repo + "/" + collection + "/" + rkey,
				"cid":   "bafy-vertical-write",
				"value": json.RawMessage(record),
			})
		default:
			http.NotFound(w, request)
		}
	}))
	fake.server = server
	t.Cleanup(server.Close)
	return fake
}

func newVerticalIndigoPDSClient(t *testing.T, host string) *auth.IndigoPDSClient {
	t.Helper()
	newAPIClient := func() *atclient.APIClient {
		client := atclient.NewAPIClient(host)
		client.Client = &http.Client{Transport: http.DefaultTransport}
		return client
	}
	client, err := auth.NewIndigoPDSClient(newAPIClient(), newAPIClient(), nil)
	if err != nil {
		t.Fatalf("new Indigo PDS client: %v", err)
	}
	return client
}
