package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestLogListeningIncludesAppVersion(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))

	logListening(logger, "0.0.0.0:8080")

	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatalf("decode startup log: %v", err)
	}
	if event["msg"] != "listening" {
		t.Fatalf("msg = %v", event["msg"])
	}
	if event["addr"] != "0.0.0.0:8080" {
		t.Fatalf("addr = %v", event["addr"])
	}
	if event["app_version"] != "dev" {
		t.Fatalf("app_version = %v, want dev", event["app_version"])
	}
}
