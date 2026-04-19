package auth

import (
	"context"
	"errors"
	"testing"
)

func TestDevDIDRoundTrip(t *testing.T) {
	ctx := context.Background()

	got, ok := DevDIDFromContext(ctx)
	if ok {
		t.Errorf("empty ctx: ok=true, did=%q, want ok=false", got)
	}

	ctx = WithDevDID(ctx, "did:plc:abc")
	got, ok = DevDIDFromContext(ctx)
	if !ok {
		t.Fatal("after WithDevDID: ok=false, want true")
	}
	if got != "did:plc:abc" {
		t.Errorf("did = %q, want did:plc:abc", got)
	}
}

func TestMockAuthService_FallsBackToDefaultDID(t *testing.T) {
	m := &MockAuthService{DefaultDID: "did:plc:default"}
	got, err := m.Authenticate(context.Background(), "any-token")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.DID != "did:plc:default" {
		t.Errorf("did = %q, want did:plc:default", got.DID)
	}
}

func TestMockAuthService_PrefersDevDIDFromContext(t *testing.T) {
	m := &MockAuthService{DefaultDID: "did:plc:default"}
	ctx := WithDevDID(context.Background(), "did:plc:override")
	got, err := m.Authenticate(ctx, "any-token")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.DID != "did:plc:override" {
		t.Errorf("did = %q, want did:plc:override", got.DID)
	}
}

func TestCraftskyAuthService_EmptyTokenInPackage(t *testing.T) {
	// CraftskyAuthService with nil Store still rejects the empty token before
	// touching the store.
	svc := &CraftskyAuthService{Store: nil}
	_, err := svc.Authenticate(context.Background(), "")
	if !errors.Is(err, ErrAuthTokenInvalid) {
		t.Fatalf("want ErrAuthTokenInvalid, got %v", err)
	}
}
