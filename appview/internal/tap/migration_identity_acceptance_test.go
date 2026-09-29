package tap_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestDevTapSkipsStaleCursorWhileProductionReplays(t *testing.T) {
	compose, err := os.ReadFile("../../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	configuration := string(compose)
	if !strings.Contains(configuration, "image: ghcr.io/bluesky-social/indigo/tap:0.1.10") {
		t.Fatal("supported Tap configuration must retain the reviewed 0.1.10 pin")
	}
	if !strings.Contains(configuration, "\n      TAP_NO_REPLAY: \"true\"\n") {
		t.Fatal("local Tap must skip a stale firehose cursor on restart")
	}

	production, err := os.ReadFile("../../../render.yaml")
	if err != nil {
		t.Fatalf("read render.yaml: %v", err)
	}
	if !regexp.MustCompile(`(?m)^[ \t]*- key: TAP_NO_REPLAY\n[ \t]*value: "false"[ \t]*$`).Match(production) {
		t.Fatal("production Tap must continue replaying from its durable cursor")
	}
}
