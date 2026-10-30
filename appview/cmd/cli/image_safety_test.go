package main

import (
	"strings"
	"testing"
)

func TestImageSafetyCatchUpRefusesProductionBeforeConnecting(t *testing.T) {
	previous := envFlag
	envFlag = "prod"
	t.Cleanup(func() { envFlag = previous })
	command, _, err := rootCmd.Find([]string{"image-safety", "dev-catch-up"})
	if err != nil {
		t.Fatal(err)
	}
	if err := command.RunE(command, nil); err == nil || !strings.Contains(err.Error(), "dev-only") {
		t.Fatalf("production catch-up error=%v", err)
	}
}
