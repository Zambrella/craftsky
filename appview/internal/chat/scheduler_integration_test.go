package chat

import (
	"context"
	"crypto/sha256"
	"errors"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type earlyFailureSessions struct {
	SessionRunner
	healthy syntax.DID
	calls   [2]atomic.Int32
	first   syntax.DID
	failed  chan struct{}
}

func (s *earlyFailureSessions) WithCapability(ctx context.Context, owner syntax.DID, parent string, required []string, operation auth.OAuthSessionOperation) error {
	if owner != s.healthy {
		index := 0
		if owner != s.first {
			index = 1
		}
		s.calls[index].Add(1)
		select {
		case s.failed <- struct{}{}:
		default:
		}
		return errors.New("synthetic authority resolution unavailable")
	}
	return s.SessionRunner.WithCapability(ctx, owner, parent, required, operation)
}

// IT-020 / AC-017, AC-022: failure before entering the session callback must
// release capacity and schedule backoff, rather than reselecting oldest owners.
func TestSchedulerBacksOffEarlyFailuresAndServesThirdOwner(t *testing.T) {
	healthy := syntax.DID("did:plc:healthy-third")
	seenHealthy := make(chan struct{}, 1)
	service, store, info := newServiceFixture(t, true, append(auth.DefaultScopes(), auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		switch {
		case strings.HasSuffix(r.URL.Path, "getLog"):
			select {
			case seenHealthy <- struct{}{}:
			default:
			}
			body = `{"cursor":"0001","logs":[]}`
		case strings.HasSuffix(r.URL.Path, "listConvos"):
			body = `{"convos":[]}`
		case strings.HasSuffix(r.URL.Path, "listConvoRequests"):
			body = `{"requests":[]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	second := syntax.DID("did:plc:early-two")
	for _, owner := range []syntax.DID{second, healthy} {
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, owner); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Connection(t.Context(), owner); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Connect(t.Context(), owner, 1, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at) VALUES($1,'synthetic-parent','{}','active',1,1,now()+interval '1 day')`, owner); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256([]byte(owner))
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,idle_expires_at) VALUES($1,$2,'synthetic-parent','active',1,now()+interval '1 day')`, hash[:], owner); err != nil {
			t.Fatal(err)
		}
	}
	for _, owner := range []syntax.DID{info.DID, second, healthy} {
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,phase,cursor) VALUES($1,'ready','saved-cursor')`, owner); err != nil {
			t.Fatal(err)
		}
	}
	early := &earlyFailureSessions{SessionRunner: service.sessions, healthy: healthy, first: info.DID, failed: make(chan struct{}, 2)}
	service.sessions = early
	now := time.Now().UTC().Truncate(time.Microsecond).Add(time.Hour)
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }, Pulses: make(chan time.Time), Jitter: func() float64 { return 0 }})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("scheduler did not stop")
		}
	})
	select {
	case <-seenHealthy:
	case <-time.After(2 * time.Second):
		t.Fatal("healthy third owner starved behind early failures")
	}
	// Healthy progress can win the race with the other admitted owner's
	// preflight. Let both failures enter before cancellation so shutdown does
	// not replace the behavior this test intends to inspect.
	for range 2 {
		select {
		case <-early.failed:
		case <-time.After(2 * time.Second):
			t.Fatal("early owner did not enter its failure callback")
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("shutdown=%v", err)
	}
	// Cleanup still needs a completion to consume.
	done <- context.Canceled
	for index, owner := range []syntax.DID{info.DID, second} {
		if early.calls[index].Load() != 1 {
			t.Fatalf("early owner retried without backoff: %d", early.calls[index].Load())
		}
		var due time.Time
		var cursor, state string
		var failures int
		var released bool
		if err := store.pool.QueryRow(t.Context(), `SELECT next_due_at,cursor,safe_state,failure_count,claim_token IS NULL FROM chat_sync_state WHERE owner_did=$1`, owner).Scan(&due, &cursor, &state, &failures, &released); err != nil {
			t.Fatal(err)
		}
		if due.Sub(now) != time.Second || cursor != "saved-cursor" || state != "unavailable" || failures != 1 || !released {
			t.Fatalf("early failure state: delay=%s cursor=%s state=%s failures=%d released=%t", due.Sub(now), cursor, state, failures, released)
		}
	}
}

// IT-020 / NFR-004: capacity is a database claim invariant, including
// independent synchronizer instances; one slow owner cannot occupy all work.
func TestPollClaimsBoundConcurrentAccountsAcrossWorkerInstances(t *testing.T) {
	started := make(chan struct{}, 3)
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"chatDisabled":false}`
		if strings.HasSuffix(r.URL.Path, "getLog") {
			started <- struct{}{}
			select {
			case <-release:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			body = `{"cursor":"0001","logs":[]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}))
	owners := []syntax.DID{info.DID, "did:plc:parallel-two", "did:plc:parallel-three"}
	for _, owner := range owners[1:] {
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, owner); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Connection(t.Context(), owner); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Connect(t.Context(), owner, 1, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at) VALUES($1,'synthetic-parent','{}','active',1,1,now()+interval '1 day')`, owner); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256([]byte(owner))
		if _, err := store.pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,idle_expires_at) VALUES($1,$2,'synthetic-parent','active',1,now()+interval '1 day')`, hash[:], owner); err != nil {
			t.Fatal(err)
		}
	}
	results := make(chan error, 3)
	now := time.Now().Add(time.Hour)
	for _, owner := range owners {
		worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		go func() { results <- worker.Poll(t.Context(), owner) }()
	}
	completed, polling := 0, 0
	for polling < 2 {
		select {
		case <-started:
			polling++
		case err := <-results:
			if !errors.Is(err, ErrPollNotDue) || completed != 0 {
				t.Fatalf("capacity admission=%v", err)
			}
			completed++
		case <-time.After(3 * time.Second):
			t.Fatal("poll did not start")
		}
	}
	if completed == 0 {
		select {
		case <-started:
			t.Fatal("more than two accounts polled concurrently")
		case err := <-results:
			if !errors.Is(err, ErrPollNotDue) {
				t.Fatalf("third claim=%v", err)
			}
			completed++
		case <-time.After(3 * time.Second):
			t.Fatal("third claim did not yield promptly")
		}
	}
	once.Do(func() { close(release) })
	for completed < 3 {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
			completed++
		case <-time.After(3 * time.Second):
			t.Fatal("held account did not finish")
		}
	}
}

// IT-020 / NFR-004, NFR-005: one held owner leaves capacity for healthy
// owners; view wakeups cannot overlap that owner's poll; shutdown joins work.
func TestSchedulerServesHealthyOwnerWhileAnotherPollIsHeld(t *testing.T) {
	service, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unused initial transport"); return nil, nil }))
	healthy := syntax.DID("did:plc:healthy-scheduler")
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, healthy); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Connection(t.Context(), healthy); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Connect(t.Context(), healthy, 1, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO oauth_sessions(account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at) VALUES($1,'synthetic-parent','{}','active',1,1,now()+interval '1 day')`, healthy); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(healthy))
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,idle_expires_at) VALUES($1,$2,'synthetic-parent','active',1,now()+interval '1 day')`, hash[:], healthy); err != nil {
		t.Fatal(err)
	}
	started := make(chan syntax.DID, 10)
	service.transport = func(session *oauth.ClientSession) (*Transport, error) {
		owner := session.Data.AccountDID
		return NewTransport(&atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			body := `{"chatDisabled":false}`
			switch r.URL.Path {
			case "/xrpc/chat.bsky.convo.getLog":
				started <- owner
				if owner == info.DID {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				body = `{"cursor":"0001","logs":[]}`
			case "/xrpc/chat.bsky.convo.listConvos":
				body = `{"convos":[]}`
			case "/xrpc/chat.bsky.convo.listConvoRequests":
				body = `{"requests":[]}`
			}
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
		})}}), nil
	}
	now := time.Now().Add(time.Hour)
	pulses := make(chan time.Time, 10)
	worker, err := NewSynchronizer(service, backgroundSessionFunc(func(context.Context, syntax.DID, []string) (string, error) { return info.SessionID, nil }), SyncOptions{Now: func() time.Time { return now }, Pulses: pulses})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	seen := map[syntax.DID]int{}
	for len(seen) < 2 {
		select {
		case owner := <-started:
			seen[owner]++
		case <-time.After(3 * time.Second):
			t.Fatal("healthy account starved behind held account")
		}
	}
	for range 5 {
		pulses <- now
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("shutdown=%v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("scheduler did not join cancelled work")
	}
	for len(started) > 0 {
		seen[<-started]++
	}
	if seen[info.DID] != 1 {
		t.Fatalf("held owner overlapped: %d", seen[info.DID])
	}
}
