package auth

import (
	"context"
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
