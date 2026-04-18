package index

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/tap"
)

func TestNotImplementedHandleErrors(t *testing.T) {
	t.Parallel()
	err := NotImplemented{}.Handle(context.Background(), tap.Event{})
	if err == nil {
		t.Fatal("expected error from NotImplemented.Handle, got nil")
	}
}
