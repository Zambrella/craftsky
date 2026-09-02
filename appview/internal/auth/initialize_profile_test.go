// appview/internal/auth/initialize_profile_test.go
package auth_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

type fakeIdentityCacheUpdater struct {
	dids []syntax.DID
	err  error
}

type fakeRepositoryTracker struct {
	dids []syntax.DID
	err  error
}

type recordingBlueskyProfileProjector struct {
	did    syntax.DID
	cid    syntax.CID
	record map[string]any
	order  *[]string
	calls  int
	err    error
}

type recordingCraftskyProfileProjector struct {
	did    syntax.DID
	cid    syntax.CID
	record map[string]any
	order  *[]string
	err    error
}

func (p *recordingCraftskyProfileProjector) ProjectCraftskyProfile(
	_ context.Context,
	did syntax.DID,
	cid syntax.CID,
	record map[string]any,
) error {
	p.did = did
	p.cid = cid
	p.record = record
	*p.order = append(*p.order, "craftsky-project")
	return p.err
}

func (p *recordingBlueskyProfileProjector) ProjectBlueskyProfile(
	_ context.Context,
	did syntax.DID,
	cid syntax.CID,
	record map[string]any,
) error {
	p.calls++
	p.did = did
	p.cid = cid
	p.record = record
	*p.order = append(*p.order, "project")
	return p.err
}

type orderedRepositoryTracker struct{ order *[]string }

func (t orderedRepositoryTracker) AddRepo(context.Context, syntax.DID) error {
	*t.order = append(*t.order, "track")
	return nil
}

type orderedIdentityCacheUpdater struct{ order *[]string }

func (u orderedIdentityCacheUpdater) UpsertCurrentHandle(context.Context, syntax.DID) error {
	*u.order = append(*u.order, "cache")
	return nil
}

func (f *fakeRepositoryTracker) AddRepo(_ context.Context, did syntax.DID) error {
	f.dids = append(f.dids, did)
	return f.err
}

func (f *fakeIdentityCacheUpdater) UpsertCurrentHandle(_ context.Context, did syntax.DID) error {
	f.dids = append(f.dids, did)
	return f.err
}

type mockPDS struct {
	getCalls []getCall
	putCalls []putCall

	getRecord func(collection, rkey string, out any) (string, error)
	putRecord func(collection, rkey string, record any) error
}

type getCall struct{ Collection, Rkey string }
type putCall struct {
	Collection, Rkey string
	Record           any
}

type testOnboardingProfileWriter struct{}

func (testOnboardingProfileWriter) PutOnboardingProfile(
	ctx context.Context,
	client auth.PDSClient,
	request auth.OnboardingProfileWrite,
) (syntax.CID, error) {
	if err := client.PutRecord(ctx, request.Owner, cskyNSID, "self", request.Record); err != nil {
		return "", err
	}
	return "new-craftsky-cid", nil
}

func loginAttempt(owner syntax.DID) auth.CallbackAttempt {
	return auth.CallbackAttempt{
		State: "test-oauth-parent", AttemptID: uuid.New(), Owner: owner,
		OwnerGeneration: 1, AuthEpoch: 1, Purpose: auth.LoginOAuthPurpose,
	}
}

func initializeProfileForTest(
	ctx context.Context,
	client auth.PDSClient,
	attempt auth.CallbackAttempt,
	writer auth.OnboardingProfileWriter,
) error {
	return auth.InitializeProfileAndIdentityCache(
		ctx, client, attempt, writer, nil, nil, nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

func (m *mockPDS) GetRecord(_ context.Context, _ syntax.DID, collection, rkey string, out any) (string, error) {
	m.getCalls = append(m.getCalls, getCall{collection, rkey})
	return m.getRecord(collection, rkey, out)
}
func (m *mockPDS) PutRecord(_ context.Context, _ syntax.DID, collection, rkey string, record any) error {
	m.putCalls = append(m.putCalls, putCall{collection, rkey, record})
	return m.putRecord(collection, rkey, record)
}
func (m *mockPDS) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}
func (m *mockPDS) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return nil
}
func (m *mockPDS) UploadBlob(_ context.Context, _ string, _ []byte) (*auth.UploadedBlob, error) {
	return nil, nil
}

const (
	bskyNSID = "app.bsky.actor.profile"
	cskyNSID = "social.craftsky.actor.profile"
)

func TestInitializeProfileAndIdentityCacheProjectsNewCraftskyProfileBeforeAuxiliaryEffects(t *testing.T) {
	t.Parallel()
	var order []string
	m := &mockPDS{
		getRecord: func(collection, _ string, _ any) (string, error) {
			if collection == bskyNSID {
				return "", auth.ErrRecordNotFound
			}
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(string, string, any) error { return nil },
	}
	projector := &recordingCraftskyProfileProjector{order: &order}

	err := auth.InitializeProfileAndIdentityCache(
		context.Background(), m, loginAttempt("did:plc:new"),
		testOnboardingProfileWriter{}, nil, projector,
		orderedIdentityCacheUpdater{order: &order},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		orderedRepositoryTracker{order: &order},
	)
	if err != nil {
		t.Fatalf("InitializeProfileAndIdentityCache: %v", err)
	}
	if got, want := order, []string{"craftsky-project", "track", "cache"}; !slices.Equal(got, want) {
		t.Fatalf("effect order = %v; want %v", got, want)
	}
	if projector.did != "did:plc:new" || projector.cid != "new-craftsky-cid" {
		t.Fatalf("projected identity = (%q, %q)", projector.did, projector.cid)
	}
	if got := projector.record["$type"]; got != cskyNSID {
		t.Fatalf("projected $type = %v; want %s", got, cskyNSID)
	}
}

func TestInitializeProfileAndIdentityCacheFailsBeforeHandoffEffectsWhenCraftskyProjectionFails(t *testing.T) {
	t.Parallel()
	var order []string
	m := &mockPDS{
		getRecord: func(collection, _ string, out any) (string, error) {
			if collection == bskyNSID {
				return "", auth.ErrRecordNotFound
			}
			*(out.(*map[string]any)) = map[string]any{"crafts": []any{}}
			return "existing-craftsky-cid", nil
		},
		putRecord: func(string, string, any) error { return nil },
	}
	projector := &recordingCraftskyProfileProjector{
		order: &order,
		err:   errors.New("projection unavailable"),
	}

	err := auth.InitializeProfileAndIdentityCache(
		context.Background(), m, loginAttempt("did:plc:projection-failure"),
		testOnboardingProfileWriter{}, nil, projector,
		orderedIdentityCacheUpdater{order: &order},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		orderedRepositoryTracker{order: &order},
	)
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Fatalf("error = %v; want ErrProfileInitFailed", err)
	}
	if got, want := order, []string{"craftsky-project"}; !slices.Equal(got, want) {
		t.Fatalf("effect order = %v; want %v", got, want)
	}
}

func TestInitializeProfile_ReturningUserBothPresent(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			case cskyNSID:
				*(out.(*map[string]any)) = map[string]any{
					"$type":  cskyNSID,
					"crafts": []any{"sewing"},
				}
				return "", nil
			}
			t.Fatalf("unexpected get collection %q", coll)
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error {
			t.Fatalf("PutRecord should not be called for returning user")
			return nil
		},
	}
	if err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:a")), testOnboardingProfileWriter{}); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
	if len(m.getCalls) != 2 {
		t.Errorf("getCalls = %d, want 2", len(m.getCalls))
	}
	if len(m.putCalls) != 0 {
		t.Errorf("putCalls = %d, want 0", len(m.putCalls))
	}
}

func TestInitializeProfile_NewUserWritesEmptyCraftsky(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			case cskyNSID:
				return "", auth.ErrRecordNotFound
			}
			return "", nil
		},
		putRecord: func(coll, rkey string, record any) error {
			if coll != cskyNSID {
				t.Errorf("put collection = %q, want %q", coll, cskyNSID)
			}
			if rkey != "self" {
				t.Errorf("put rkey = %q, want self", rkey)
			}
			body, _ := record.(map[string]any)
			if body["$type"] != cskyNSID {
				t.Errorf("put $type = %v", body["$type"])
			}
			c, _ := body["crafts"].([]string)
			if len(c) != 0 {
				t.Errorf("put crafts = %v, want empty", c)
			}
			return nil
		},
	}
	if err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:b")), testOnboardingProfileWriter{}); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
	if len(m.putCalls) != 1 {
		t.Errorf("putCalls = %d, want 1", len(m.putCalls))
	}
}

func TestInitializeProfileAndIdentityCacheUpsertsAfterSuccessfulInitialization(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			}
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	updater := &fakeIdentityCacheUpdater{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := auth.InitializeProfileAndIdentityCache(context.Background(), m, loginAttempt(syntax.DID("did:plc:new")), testOnboardingProfileWriter{}, nil, nil, updater, logger); err != nil {
		t.Fatalf("InitializeProfileAndIdentityCache: %v", err)
	}
	if len(updater.dids) != 1 || updater.dids[0].String() != "did:plc:new" {
		t.Fatalf("upserted DIDs = %v, want did:plc:new", updater.dids)
	}
}

func TestInitializeProfileAndIdentityCacheProjectsFetchedBlueskyProfileBeforeAuxiliaryEffects(t *testing.T) {
	t.Parallel()
	did := syntax.DID("did:plc:project-profile")
	cid := syntax.CID("bafyreiprofile")
	profile := map[string]any{"displayName": "Alice", "description": "Maker"}
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			switch coll {
			case bskyNSID:
				*(out.(*map[string]any)) = profile
				return cid.String(), nil
			case cskyNSID:
				*(out.(*map[string]any)) = map[string]any{"crafts": []any{"sewing"}}
				return "", nil
			default:
				t.Fatalf("unexpected collection %q", coll)
				return "", nil
			}
		},
		putRecord: func(string, string, any) error { return nil },
	}
	order := []string{}
	projector := &recordingBlueskyProfileProjector{order: &order}

	err := auth.InitializeProfileAndIdentityCache(
		context.Background(), m, loginAttempt(did), testOnboardingProfileWriter{},
		projector, nil, orderedIdentityCacheUpdater{order: &order},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		orderedRepositoryTracker{order: &order},
	)
	if err != nil {
		t.Fatalf("InitializeProfileAndIdentityCache: %v", err)
	}
	if projector.did != did || projector.cid != cid {
		t.Fatalf("projected identity = (%q, %q), want (%q, %q)", projector.did, projector.cid, did, cid)
	}
	if projector.record["displayName"] != "Alice" || projector.record["description"] != "Maker" {
		t.Fatalf("projected record = %#v", projector.record)
	}
	wantOrder := []string{"project", "track", "cache"}
	if len(order) != len(wantOrder) {
		t.Fatalf("effect order = %v, want %v", order, wantOrder)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("effect order = %v, want %v", order, wantOrder)
		}
	}
}

func TestInitializeProfileAndIdentityCacheProjectionIsOptionalAndBestEffort(t *testing.T) {
	t.Run("missing Bluesky profile skips projection", func(t *testing.T) {
		order := []string{}
		projector := &recordingBlueskyProfileProjector{order: &order}
		m := &mockPDS{
			getRecord: func(coll, _ string, out any) (string, error) {
				if coll == bskyNSID {
					return "", auth.ErrRecordNotFound
				}
				*(out.(*map[string]any)) = map[string]any{"crafts": []any{"sewing"}}
				return "", nil
			},
			putRecord: func(string, string, any) error { return nil },
		}

		err := auth.InitializeProfileAndIdentityCache(
			context.Background(), m, loginAttempt(syntax.DID("did:plc:no-bsky-profile")),
			testOnboardingProfileWriter{}, projector, nil, nil,
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		)
		if err != nil || projector.calls != 0 {
			t.Fatalf("missing profile err=%v projector calls=%d", err, projector.calls)
		}
	})

	t.Run("projection failure warns safely and continues", func(t *testing.T) {
		const profileSecret = "profile-secret-sentinel"
		const errorSecret = "projector-error-sentinel"
		order := []string{}
		projector := &recordingBlueskyProfileProjector{
			order: &order,
			err:   errors.New(errorSecret),
		}
		m := &mockPDS{
			getRecord: func(coll, _ string, out any) (string, error) {
				if coll == bskyNSID {
					*(out.(*map[string]any)) = map[string]any{"description": profileSecret}
					return "bafyreifailure", nil
				}
				*(out.(*map[string]any)) = map[string]any{"crafts": []any{"sewing"}}
				return "", nil
			},
			putRecord: func(string, string, any) error { return nil },
		}
		var logs bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logs, nil))

		err := auth.InitializeProfileAndIdentityCache(
			context.Background(), m, loginAttempt(syntax.DID("did:plc:projection-failure")),
			testOnboardingProfileWriter{}, projector, nil,
			orderedIdentityCacheUpdater{order: &order}, logger,
			orderedRepositoryTracker{order: &order},
		)
		if err != nil {
			t.Fatalf("projection failure should not fail initialization: %v", err)
		}
		if got := strings.Join(order, ","); got != "project,track,cache" {
			t.Fatalf("effect order = %q, want project,track,cache", got)
		}
		logged := logs.String()
		if !strings.Contains(logged, "profile_init.bluesky_projection") {
			t.Fatalf("projection warning missing operation: %s", logged)
		}
		if strings.Contains(logged, profileSecret) || strings.Contains(logged, errorSecret) ||
			strings.Contains(logged, "did:plc:projection-failure") || strings.Contains(logged, "bafyreifailure") {
			t.Fatalf("projection warning leaked sensitive details: %s", logged)
		}
	})
}

func TestInitializeProfileAndIdentityCacheLogsAndContinuesWhenUpsertFails(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			}
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	updater := &fakeIdentityCacheUpdater{err: errors.New("identity unavailable")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := auth.InitializeProfileAndIdentityCache(context.Background(), m, loginAttempt(syntax.DID("did:plc:new")), testOnboardingProfileWriter{}, nil, nil, updater, logger); err != nil {
		t.Fatalf("InitializeProfileAndIdentityCache should continue on updater failure: %v", err)
	}
	if len(updater.dids) != 1 {
		t.Fatalf("updater calls = %d, want 1", len(updater.dids))
	}
}

func TestInitializeProfileAndIdentityCacheRequestsRepositoryTrackingOnEverySuccess(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{"displayName": "Alice"}
				return "", nil
			}
			*(out.(*map[string]any)) = map[string]any{"crafts": []any{"sewing"}}
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	tracker := &fakeRepositoryTracker{err: errors.New("Tap temporarily unavailable")}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	did := syntax.DID("did:plc:returning")

	for i := 0; i < 2; i++ {
		if err := auth.InitializeProfileAndIdentityCache(context.Background(), m, loginAttempt(did), testOnboardingProfileWriter{}, nil, nil, nil, logger, tracker); err != nil {
			t.Fatalf("InitializeProfileAndIdentityCache retry %d: %v", i, err)
		}
	}
	if len(tracker.dids) != 2 || tracker.dids[0] != did || tracker.dids[1] != did {
		t.Fatalf("tracking requests = %v, want DID twice", tracker.dids)
	}
}

func TestOrdinaryOnboardingPreservesProfileAndAuxiliaryEffectSeverity(t *testing.T) {
	for _, purpose := range []auth.OAuthPurpose{auth.LoginOAuthPurpose, auth.RegistrationOAuthPurpose} {
		t.Run(string(purpose)+" profile failure is fatal", func(t *testing.T) {
			pds := &mockPDS{
				getRecord: func(string, string, any) (string, error) {
					return "", errors.New("profile unavailable")
				},
				putRecord: func(string, string, any) error { return nil },
			}
			attempt := loginAttempt(syntax.DID("did:plc:severity-fatal"))
			attempt.Purpose = purpose
			cache := &fakeIdentityCacheUpdater{}
			tracker := &fakeRepositoryTracker{}
			err := auth.InitializeProfileAndIdentityCache(
				context.Background(), pds, attempt, testOnboardingProfileWriter{}, nil, nil, cache,
				slog.New(slog.NewTextHandler(io.Discard, nil)), tracker,
			)
			if !errors.Is(err, auth.ErrProfileInitFailed) || len(cache.dids) != 0 || len(tracker.dids) != 0 {
				t.Fatalf("profile failure err=%v cache=%v tracker=%v", err, cache.dids, tracker.dids)
			}
		})
		t.Run(string(purpose)+" cache and tracker failures warn only", func(t *testing.T) {
			pds := &mockPDS{
				getRecord: func(collection, _ string, out any) (string, error) {
					if collection == bskyNSID {
						return "", auth.ErrRecordNotFound
					}
					*(out.(*map[string]any)) = map[string]any{"crafts": []any{"sewing"}}
					return "", nil
				},
				putRecord: func(string, string, any) error { return nil },
			}
			attempt := loginAttempt(syntax.DID("did:plc:severity-warning"))
			attempt.Purpose = purpose
			cache := &fakeIdentityCacheUpdater{err: errors.New("cache unavailable")}
			tracker := &fakeRepositoryTracker{err: errors.New("tracker unavailable")}
			err := auth.InitializeProfileAndIdentityCache(
				context.Background(), pds, attempt, testOnboardingProfileWriter{}, nil, nil, cache,
				slog.New(slog.NewTextHandler(io.Discard, nil)), tracker,
			)
			if err != nil || len(cache.dids) != 1 || len(tracker.dids) != 1 {
				t.Fatalf("warning effects err=%v cache=%v tracker=%v", err, cache.dids, tracker.dids)
			}
		})
	}
}

func TestInitializeProfile_NoBlueskyProfileIsOK(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, _ any) (string, error) {
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(coll, _ string, _ any) error {
			if coll != cskyNSID {
				t.Errorf("put collection = %q, want %q", coll, cskyNSID)
			}
			return nil
		},
	}
	if err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:c")), testOnboardingProfileWriter{}); err != nil {
		t.Fatalf("InitializeProfile: %v", err)
	}
}

func TestInitializeProfile_BlueskyReadErrorFails(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	m := &mockPDS{
		getRecord: func(coll, _ string, _ any) (string, error) {
			if coll == bskyNSID {
				return "", boom
			}
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:d")), testOnboardingProfileWriter{})
	if err == nil {
		t.Fatal("want error; got nil")
	}
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}

func TestInitializeProfile_CraftskyReadErrorFails(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return "", nil
			}
			return "", errors.New("boom")
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:e")), testOnboardingProfileWriter{})
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}

func TestInitializeProfile_MalformedCraftskyRecord(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return "", nil
			}
			// crafts expected to be []string; return wrong type.
			*(out.(*map[string]any)) = map[string]any{
				"$type":  cskyNSID,
				"crafts": "not an array",
			}
			return "", nil
		},
		putRecord: func(_, _ string, _ any) error { return nil },
	}
	err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:f")), testOnboardingProfileWriter{})
	if !errors.Is(err, auth.ErrProfileDataInvalid) {
		t.Errorf("want ErrProfileDataInvalid; got %v", err)
	}
}

func TestInitializeProfile_PutRecordFailure(t *testing.T) {
	t.Parallel()
	m := &mockPDS{
		getRecord: func(coll, _ string, out any) (string, error) {
			if coll == bskyNSID {
				*(out.(*map[string]any)) = map[string]any{}
				return "", nil
			}
			return "", auth.ErrRecordNotFound
		},
		putRecord: func(_, _ string, _ any) error { return errors.New("pds down") },
	}
	err := initializeProfileForTest(context.Background(), m, loginAttempt(syntax.DID("did:plc:g")), testOnboardingProfileWriter{})
	if !errors.Is(err, auth.ErrProfileInitFailed) {
		t.Errorf("want ErrProfileInitFailed; got %v", err)
	}
}
