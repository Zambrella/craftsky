package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"social.craftsky/appview/internal/observability"
)

type privateStartupError struct{}

func (privateStartupError) Error() string {
	return "private-payload-canary Authorization: Bearer credential-canary postgres://private-dsn-canary"
}

func TestStartupDiagnosticPreservesCauseTypeWithoutPrivateProse(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&output, nil)))
	logStartupFailure(logger, fmt.Errorf("private-wrapper-canary: %w", privateStartupError{}))
	for _, positive := range []string{"proof.start", "startup", "privateStartupError"} {
		if !strings.Contains(output.String(), positive) {
			t.Errorf("missing diagnostic field %s", positive)
		}
	}
	for _, secret := range []string{"private-payload-canary", "credential-canary", "private-dsn-canary", "private-wrapper-canary"} {
		if strings.Contains(output.String(), secret) {
			t.Errorf("private value leaked: %s", secret)
		}
	}
}
