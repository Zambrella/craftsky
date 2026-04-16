package firehose

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNotImplemented_ReplayErrors(t *testing.T) {
	var s Subscriber = NotImplemented{}
	err := s.Replay(context.Background(), time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "firehose") || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("err = %q, want containing 'firehose' and 'not yet implemented'", err.Error())
	}
}
