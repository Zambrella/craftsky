package app

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type recordingOAuthMetadataSource struct {
	mu                     sync.Mutex
	protectedResourceCalls int
	authorizationCalls     int
	issuerURL              string
	protectedResourceError error
	authorizationError     error
	issuerByPDS            map[string]string
}

type recordingOAuthMetadataCacheObserver struct {
	mu     sync.Mutex
	events [][2]string
}

func (observer *recordingOAuthMetadataCacheObserver) ObserveOAuthMetadataCache(_ context.Context, stage, result string) {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	observer.events = append(observer.events, [2]string{stage, result})
}

func (observer *recordingOAuthMetadataCacheObserver) Events() [][2]string {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	return append([][2]string(nil), observer.events...)
}

func (observer *recordingOAuthMetadataCacheObserver) Count(stage, result string) int {
	observer.mu.Lock()
	defer observer.mu.Unlock()
	count := 0
	for _, event := range observer.events {
		if event == [2]string{stage, result} {
			count++
		}
	}
	return count
}

func (source *recordingOAuthMetadataSource) ResolveAuthServerURL(_ context.Context, pds string) (string, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.protectedResourceCalls++
	if source.protectedResourceError != nil {
		return "", source.protectedResourceError
	}
	if issuer := source.issuerByPDS[pds]; issuer != "" {
		return issuer, nil
	}
	return source.issuerURL, nil
}

func (source *recordingOAuthMetadataSource) ResolveAuthServerMetadata(_ context.Context, issuer string) (*oauth.AuthServerMetadata, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	source.authorizationCalls++
	if source.authorizationError != nil {
		return nil, source.authorizationError
	}
	return &oauth.AuthServerMetadata{Issuer: issuer}, nil
}

func (source *recordingOAuthMetadataSource) calls() (int, int) {
	source.mu.Lock()
	defer source.mu.Unlock()
	return source.protectedResourceCalls, source.authorizationCalls
}

func TestOAuthAuthorityMetadataCacheDoesNotRetainFailures(t *testing.T) {
	source := &recordingOAuthMetadataSource{
		issuerURL:              "https://auth.example",
		protectedResourceError: errors.New("metadata unavailable"),
	}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := resolver.ResolveIssuer(context.Background(), "https://pds.example"); err == nil {
			t.Fatal("ResolveIssuer succeeded during metadata failure")
		}
	}
	if protected, authorization := source.calls(); protected != 2 || authorization != 0 {
		t.Fatalf("failure calls = protected:%d authorization:%d, want 2/0", protected, authorization)
	}

	source.mu.Lock()
	source.protectedResourceError = nil
	source.authorizationError = errors.New("issuer unavailable")
	source.mu.Unlock()
	for range 2 {
		if _, err := resolver.ResolveIssuer(context.Background(), "https://pds.example"); err == nil {
			t.Fatal("ResolveIssuer succeeded during issuer failure")
		}
	}
	if protected, authorization := source.calls(); protected != 3 || authorization != 2 {
		t.Fatalf("issuer failure calls = protected:%d authorization:%d, want 3/2", protected, authorization)
	}
}

func TestOAuthAuthorityMetadataCacheRejectsTTLAboveSecurityBound(t *testing.T) {
	_, err := newCachedOAuthAuthorityMetadataResolver(
		&recordingOAuthMetadataSource{},
		maxOAuthAuthorityMetadataCacheTTL+time.Nanosecond,
		10,
		time.Second,
		time.Now,
		nil,
	)
	if err == nil {
		t.Fatal("cache accepted TTL above five-minute security bound")
	}
}

func TestOAuthAuthorityMetadataCacheBoundsCapacityAcrossStages(t *testing.T) {
	source := &recordingOAuthMetadataSource{
		issuerURL: "https://auth-a.example",
		issuerByPDS: map[string]string{
			"https://pds-a.example": "https://auth-a.example",
			"https://pds-b.example": "https://auth-b.example",
		},
	}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 2, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, pds := range []string{"https://pds-a.example", "https://pds-b.example", "https://pds-a.example"} {
		if _, err := resolver.ResolveIssuer(context.Background(), pds); err != nil {
			t.Fatal(err)
		}
	}
	if protected, authorization := source.calls(); protected != 3 || authorization != 3 {
		t.Fatalf("capacity calls = protected:%d authorization:%d, want 3/3", protected, authorization)
	}
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	if len(resolver.entries) != 2 || resolver.order.Len() != 2 {
		t.Fatalf("cache size = %d/%d, want 2/2", len(resolver.entries), resolver.order.Len())
	}
}

type blockingOAuthMetadataSource struct {
	protectedCalls atomic.Int32
	authCalls      atomic.Int32
	started        chan struct{}
	release        chan struct{}
}

func (source *blockingOAuthMetadataSource) ResolveAuthServerURL(ctx context.Context, _ string) (string, error) {
	if source.protectedCalls.Add(1) == 1 {
		close(source.started)
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-source.release:
		return "https://auth.example", nil
	}
}

func (source *blockingOAuthMetadataSource) ResolveAuthServerMetadata(context.Context, string) (*oauth.AuthServerMetadata, error) {
	source.authCalls.Add(1)
	return &oauth.AuthServerMetadata{Issuer: "https://auth.example"}, nil
}

func TestOAuthAuthorityMetadataCacheCoalescesMissesAndLetsWaitersCancel(t *testing.T) {
	source := &blockingOAuthMetadataSource{started: make(chan struct{}), release: make(chan struct{})}
	observer := &recordingOAuthMetadataCacheObserver{}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, time.Now, observer,
	)
	if err != nil {
		t.Fatal(err)
	}

	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.ResolveIssuer(context.Background(), "https://pds.example")
		firstResult <- err
	}()
	<-source.started

	const waiters = 8
	waiterResults := make(chan error, waiters)
	canceledCtx, cancelWaiter := context.WithCancel(context.Background())
	cancelWaiter()
	if _, err := resolver.ResolveIssuer(canceledCtx, "https://PDS.example/"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter error = %v", err)
	}
	for range waiters {
		go func() {
			_, err := resolver.ResolveIssuer(context.Background(), "https://PDS.example/")
			waiterResults <- err
		}()
	}
	key := oauthMetadataCacheKey{stage: "protected_resource", origin: "https://pds.example"}
	deadline := time.Now().Add(time.Second)
	for {
		resolver.mu.Lock()
		joined := resolver.flights[key] != nil && resolver.flights[key].waiters == waiters+1
		resolver.mu.Unlock()
		if joined {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("waiters did not join the shared protected-resource flight")
		}
		time.Sleep(time.Millisecond)
	}
	close(source.release)
	if err := <-firstResult; err != nil {
		t.Fatal(err)
	}
	for range waiters {
		err := <-waiterResults
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := source.protectedCalls.Load(); got != 1 {
		t.Fatalf("protected-resource calls = %d, want 1", got)
	}
	if got := source.authCalls.Load(); got != 1 {
		t.Fatalf("authorization-server calls = %d, want 1", got)
	}
	if got := observer.Count("protected_resource", "miss"); got != 1 {
		t.Fatalf("protected-resource miss events = %d, want 1", got)
	}
	if got := observer.Count("protected_resource", "coalesced"); got != waiters {
		t.Fatalf("protected-resource coalesced events = %d, want %d", got, waiters)
	}
}

type cancellationAwareOAuthMetadataSource struct {
	calls   atomic.Int32
	started chan struct{}
}

type cancellationIgnoringOAuthMetadataSource struct {
	calls          atomic.Int32
	active         atomic.Int32
	maximumActive  atomic.Int32
	firstStarted   chan struct{}
	firstCanceled  chan struct{}
	releaseFirst   chan struct{}
	firstCompleted chan struct{}
}

func (source *cancellationIgnoringOAuthMetadataSource) ResolveAuthServerURL(ctx context.Context, _ string) (string, error) {
	call := source.calls.Add(1)
	active := source.active.Add(1)
	for {
		maximum := source.maximumActive.Load()
		if active <= maximum || source.maximumActive.CompareAndSwap(maximum, active) {
			break
		}
	}
	defer source.active.Add(-1)
	if call == 1 {
		close(source.firstStarted)
		<-ctx.Done()
		close(source.firstCanceled)
		<-source.releaseFirst
		close(source.firstCompleted)
		return "", ctx.Err()
	}
	return "https://auth.example", nil
}

func (*cancellationIgnoringOAuthMetadataSource) ResolveAuthServerMetadata(context.Context, string) (*oauth.AuthServerMetadata, error) {
	return &oauth.AuthServerMetadata{Issuer: "https://auth.example"}, nil
}

func (source *cancellationAwareOAuthMetadataSource) ResolveAuthServerURL(ctx context.Context, _ string) (string, error) {
	if source.calls.Add(1) == 1 {
		close(source.started)
		<-ctx.Done()
		return "", ctx.Err()
	}
	return "https://auth.example", nil
}

func (*cancellationAwareOAuthMetadataSource) ResolveAuthServerMetadata(context.Context, string) (*oauth.AuthServerMetadata, error) {
	return &oauth.AuthServerMetadata{Issuer: "https://auth.example"}, nil
}

func TestOAuthAuthorityMetadataCacheCancelsAbandonedFlightWithoutCaching(t *testing.T) {
	source := &cancellationAwareOAuthMetadataSource{started: make(chan struct{})}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.ResolveIssuer(ctx, "https://pds.example")
		firstResult <- err
	}()
	<-source.started
	cancel()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("abandoned flight error = %v", err)
	}

	if _, err := resolver.ResolveIssuer(context.Background(), "https://pds.example"); err != nil {
		t.Fatal(err)
	}
	if got := source.calls.Load(); got != 2 {
		t.Fatalf("protected-resource calls = %d, want 2", got)
	}
}

func TestOAuthAuthorityMetadataCacheCountsAbandonedLoaderUntilExit(t *testing.T) {
	source := &cancellationIgnoringOAuthMetadataSource{
		firstStarted: make(chan struct{}), firstCanceled: make(chan struct{}),
		releaseFirst: make(chan struct{}), firstCompleted: make(chan struct{}),
	}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 1, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.ResolveIssuer(ctx, "https://pds-a.example")
		firstResult <- err
	}()
	<-source.firstStarted
	cancel()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("abandoned flight error = %v", err)
	}
	<-source.firstCanceled

	if _, err := resolver.ResolveIssuer(context.Background(), "https://pds-b.example"); err == nil {
		t.Fatal("new loader started while abandoned loader was still active")
	}
	if got := source.calls.Load(); got != 1 {
		t.Fatalf("loader calls before abandoned exit = %d, want 1", got)
	}

	close(source.releaseFirst)
	<-source.firstCompleted
	deadline := time.Now().Add(time.Second)
	for {
		resolver.mu.Lock()
		activeLoads := resolver.activeLoads
		resolver.mu.Unlock()
		if activeLoads == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("abandoned loader did not release capacity")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := resolver.ResolveIssuer(context.Background(), "https://pds-b.example"); err != nil {
		t.Fatal(err)
	}
	if got := source.maximumActive.Load(); got != 1 {
		t.Fatalf("maximum active loaders = %d, want 1", got)
	}
}

func TestOAuthAuthorityMetadataCacheBoundsDistinctFlights(t *testing.T) {
	source := &blockingOAuthMetadataSource{started: make(chan struct{}), release: make(chan struct{})}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 1, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	firstResult := make(chan error, 1)
	go func() {
		_, err := resolver.ResolveIssuer(context.Background(), "https://pds-a.example")
		firstResult <- err
	}()
	<-source.started
	if _, err := resolver.ResolveIssuer(context.Background(), "https://pds-b.example"); err == nil {
		t.Fatal("distinct flight exceeded cache capacity")
	}
	close(source.release)
	if err := <-firstResult; err != nil {
		t.Fatal(err)
	}
	if got := source.protectedCalls.Load(); got != 1 {
		t.Fatalf("protected-resource calls = %d, want 1", got)
	}
}

type countingOAuthAuthorityDirectory struct {
	lookups atomic.Int32
	value   *identity.Identity
}

func (directory *countingOAuthAuthorityDirectory) LookupDID(context.Context, syntax.DID) (*identity.Identity, error) {
	directory.lookups.Add(1)
	copy := *directory.value
	return &copy, nil
}

func (directory *countingOAuthAuthorityDirectory) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	return nil, errors.New("handle lookup is unavailable")
}

func (directory *countingOAuthAuthorityDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return nil, errors.New("identifier lookup is unavailable")
}

func (*countingOAuthAuthorityDirectory) Purge(context.Context, syntax.AtIdentifier) error { return nil }

func TestAuthoritativeOAuthVerifierKeepsDIDFreshWhileMetadataIsWarm(t *testing.T) {
	did := syntax.DID("did:plc:metadata-cache")
	directory := &countingOAuthAuthorityDirectory{value: &identity.Identity{
		DID: did,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds.example"},
		},
	}}
	source := &recordingOAuthMetadataSource{issuerURL: "https://auth.example"}
	metadata, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := newAuthoritativeOAuthVerifierWithMetadata(directory, metadata)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		authority, err := verifier.ResolveCurrent(context.Background(), did)
		if err != nil {
			t.Fatal(err)
		}
		if authority.PDSOrigin != "https://pds.example" || authority.IssuerOrigin != "https://auth.example" {
			t.Fatalf("authority = %+v", authority)
		}
	}
	if got := directory.lookups.Load(); got != 2 {
		t.Fatalf("DID lookups = %d, want 2", got)
	}
	if protected, authorization := source.calls(); protected != 1 || authorization != 1 {
		t.Fatalf("metadata calls = protected:%d authorization:%d, want 1/1", protected, authorization)
	}
}

func TestOAuthAuthorityVerifierWiringCachesOnlyOrdinaryOperations(t *testing.T) {
	did := syntax.DID("did:plc:metadata-wiring")
	directory := &countingOAuthAuthorityDirectory{value: &identity.Identity{
		DID: did,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds.example"},
		},
	}}
	source := &recordingOAuthMetadataSource{issuerURL: "https://auth.example"}
	fresh, operations, err := newOAuthAuthorityVerifiers(
		directory, source, 5*time.Minute, 10, time.Second, nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if _, err := operations.ResolveCurrent(context.Background(), did); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if _, err := fresh.ResolveCurrent(context.Background(), did); err != nil {
			t.Fatal(err)
		}
	}
	if got := directory.lookups.Load(); got != 4 {
		t.Fatalf("DID lookups = %d, want 4", got)
	}
	if protected, authorization := source.calls(); protected != 3 || authorization != 3 {
		t.Fatalf("metadata calls = protected:%d authorization:%d, want 3/3", protected, authorization)
	}
}

func TestCachedOAuthAuthorityVerifierDetectsPDSMoveImmediately(t *testing.T) {
	now := time.Date(2026, 9, 4, 13, 0, 0, 0, time.UTC)
	did := syntax.DID("did:plc:metadata-move")
	directory := &countingOAuthAuthorityDirectory{value: &identity.Identity{
		DID: did,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds-a.example"},
		},
	}}
	source := &recordingOAuthMetadataSource{issuerByPDS: map[string]string{
		"https://pds-a.example": "https://auth-a.example",
		"https://pds-b.example": "https://auth-b.example",
	}}
	metadata, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, func() time.Time { return now }, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := newAuthoritativeOAuthVerifierWithMetadata(directory, metadata)
	if err != nil {
		t.Fatal(err)
	}

	first, err := verifier.ResolveCurrent(context.Background(), did)
	if err != nil {
		t.Fatal(err)
	}
	if first.PDSOrigin != "https://pds-a.example" || first.IssuerOrigin != "https://auth-a.example" {
		t.Fatalf("first authority = %+v", first)
	}
	directory.value.Services["atproto_pds"] = identity.ServiceEndpoint{
		Type: "AtprotoPersonalDataServer", URL: "https://pds-b.example",
	}
	second, err := verifier.ResolveCurrent(context.Background(), did)
	if err != nil {
		t.Fatal(err)
	}
	if second.PDSOrigin != "https://pds-b.example" || second.IssuerOrigin != "https://auth-b.example" {
		t.Fatalf("migrated authority = %+v", second)
	}
	if protected, authorization := source.calls(); protected != 2 || authorization != 2 {
		t.Fatalf("migration metadata calls = protected:%d authorization:%d, want 2/2", protected, authorization)
	}
}

func TestCachedOAuthAuthorityVerifierBoundsIssuerOnlyChangeByTTL(t *testing.T) {
	now := time.Date(2026, 9, 4, 14, 0, 0, 0, time.UTC)
	did := syntax.DID("did:plc:metadata-issuer")
	directory := &countingOAuthAuthorityDirectory{value: &identity.Identity{
		DID: did,
		Services: map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds.example"},
		},
	}}
	source := &recordingOAuthMetadataSource{issuerByPDS: map[string]string{
		"https://pds.example": "https://auth-a.example",
	}}
	metadata, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, func() time.Time { return now }, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := newAuthoritativeOAuthVerifierWithMetadata(directory, metadata)
	if err != nil {
		t.Fatal(err)
	}
	if authority, err := verifier.ResolveCurrent(context.Background(), did); err != nil || authority.IssuerOrigin != "https://auth-a.example" {
		t.Fatalf("initial authority = %+v, %v", authority, err)
	}

	source.mu.Lock()
	source.issuerByPDS["https://pds.example"] = "https://auth-b.example"
	source.mu.Unlock()
	now = now.Add(4*time.Minute + 59*time.Second)
	if authority, err := verifier.ResolveCurrent(context.Background(), did); err != nil || authority.IssuerOrigin != "https://auth-a.example" {
		t.Fatalf("warm authority = %+v, %v", authority, err)
	}

	now = now.Add(time.Second)
	if authority, err := verifier.ResolveCurrent(context.Background(), did); err != nil || authority.IssuerOrigin != "https://auth-b.example" {
		t.Fatalf("expired authority = %+v, %v", authority, err)
	}
}

func TestOAuthAuthorityMetadataCacheReusesValidatedResultsUntilFixedExpiry(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	source := &recordingOAuthMetadataSource{issuerURL: "https://AUTH.example/"}
	observer := &recordingOAuthMetadataCacheObserver{}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source,
		5*time.Minute,
		10,
		10*time.Second,
		func() time.Time { return now },
		observer,
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, pds := range []string{"https://PDS.example/", "https://pds.example"} {
		issuer, err := resolver.ResolveIssuer(context.Background(), pds)
		if err != nil {
			t.Fatalf("ResolveIssuer(%q): %v", pds, err)
		}
		if issuer != "https://auth.example" {
			t.Fatalf("issuer = %q", issuer)
		}
	}
	if protected, authorization := source.calls(); protected != 1 || authorization != 1 {
		t.Fatalf("warm calls = protected:%d authorization:%d, want 1/1", protected, authorization)
	}

	now = now.Add(4*time.Minute + 59*time.Second)
	if _, err := resolver.ResolveIssuer(context.Background(), "https://pds.example"); err != nil {
		t.Fatal(err)
	}
	if protected, authorization := source.calls(); protected != 1 || authorization != 1 {
		t.Fatalf("pre-expiry calls = protected:%d authorization:%d, want 1/1", protected, authorization)
	}

	now = now.Add(time.Second)
	if _, err := resolver.ResolveIssuer(context.Background(), "https://pds.example"); err != nil {
		t.Fatal(err)
	}
	if protected, authorization := source.calls(); protected != 2 || authorization != 2 {
		t.Fatalf("expiry calls = protected:%d authorization:%d, want 2/2", protected, authorization)
	}
	wantEvents := [][2]string{
		{"protected_resource", "miss"}, {"authorization_server", "miss"},
		{"protected_resource", "hit"}, {"authorization_server", "hit"},
		{"protected_resource", "hit"}, {"authorization_server", "hit"},
		{"protected_resource", "miss"}, {"authorization_server", "miss"},
	}
	if events := observer.Events(); !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("cache events = %#v, want %#v", events, wantEvents)
	}
}

func TestOAuthAuthorityMetadataCacheDoesNotRetainInvalidResolvedOrigin(t *testing.T) {
	source := &recordingOAuthMetadataSource{issuerURL: "https://auth.example/not-an-origin"}
	resolver, err := newCachedOAuthAuthorityMetadataResolver(
		source, 5*time.Minute, 10, time.Second, time.Now, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := resolver.ResolveIssuer(context.Background(), "https://pds.example"); err == nil {
			t.Fatal("ResolveIssuer accepted metadata URL with a path")
		}
	}
	if protected, authorization := source.calls(); protected != 2 || authorization != 0 {
		t.Fatalf("invalid-origin calls = protected:%d authorization:%d, want 2/0", protected, authorization)
	}
}
