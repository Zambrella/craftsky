package ctxkeys_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/ctxkeys"
)

func TestDeviceID_RoundTrip(t *testing.T) {
	ctx := ctxkeys.WithDeviceID(context.Background(), "dev-abc")
	got, ok := ctxkeys.GetDeviceID(ctx)
	if !ok || got != "dev-abc" {
		t.Errorf("got (%q, %v), want (dev-abc, true)", got, ok)
	}
}

func TestDeviceID_AbsentReturnsFalse(t *testing.T) {
	_, ok := ctxkeys.GetDeviceID(context.Background())
	if ok {
		t.Error("ok should be false for empty context")
	}
}
