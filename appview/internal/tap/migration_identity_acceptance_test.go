package tap_test

import (
	"os"
	"strings"
	"testing"
)

func TestSupportedTapConfigurationResumesFromDurableCursor(t *testing.T) {
	compose, err := os.ReadFile("../../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	configuration := string(compose)
	if !strings.Contains(configuration, "image: ghcr.io/bluesky-social/indigo/tap:0.1.10") {
		t.Fatal("supported Tap configuration must retain the reviewed 0.1.10 pin")
	}
	if strings.Contains(configuration, "TAP_NO_REPLAY") {
		t.Fatal("supported Tap configuration disables durable-cursor replay")
	}
}
