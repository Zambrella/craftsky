package main

import (
	"strings"
	"testing"
)

func TestConfigCheckWithholdsInvalidProtectedInput(t *testing.T) {
	previous := envFlag
	envFlag = "dev"
	t.Cleanup(func() { envFlag = previous })
	t.Setenv("VIDEO_PROVIDER", "protected-input-canary")
	command, _, err := rootCmd.Find([]string{"config-check"})
	if err != nil || command == rootCmd {
		t.Fatal("config-check not registered")
	}
	err = command.RunE(command, nil)
	if err == nil || strings.Contains(err.Error(), "protected-input-canary") {
		t.Fatal("config-check must fail without exposing invalid input")
	}
}
