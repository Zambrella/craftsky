package observability

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIT014ContributorGuideAndCallableExamples(t *testing.T) {
	guide, err := os.ReadFile("../../../docs/development/logging-error-reporting.md")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := os.ReadFile("../../../AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(policy), "docs/development/logging-error-reporting.md") {
		t.Fatal("canonical mandatory policy link missing")
	}
	for _, topic := range []string{"CaptureDiagnostic", "LogDiagnostic", "ObservePrivateFailure", "DiagnosticMessage", "beforeSend", "requestId", "run_id", "retry", "credentials", "Verification", "checklist"} {
		if !strings.Contains(string(guide), topic) {
			t.Errorf("guide missing %s", topic)
		}
	}
	var output bytes.Buffer
	observer := New(Config{Logger: slog.New(slog.NewJSONHandler(&output, nil))})
	ctx := context.Background()
	input := DiagnosticInput{Error: &pgconn.PgError{Code: "08006", Message: "PRIVATE_ROW"}, Context: EventContext{"component": "api", "operation": "post.read", "failure_stage": "query", "result": "error"}, Workflow: PublicRecordContext{ActorDID: syntax.DID("did:plc:actor"), TargetDID: syntax.DID("did:plc:target"), URI: syntax.ATURI("at://did:plc:target/social.craftsky.feed.post/record")}}
	LogDiagnostic(ctx, observer.logger, input)
	observer.CaptureDiagnostic(ctx, input)
	observer.ObservePrivateFailure(ctx, errors.New("PRIVATE_DRAFT"), syntax.DID("did:plc:actor"), "40000000-0000-4000-8000-000000000001", "schedule.publish", "record_write", "retry", 2)
	raw, _ := url.Parse("/v1/saves/PRIVATE_FOLDER?access_token=PRIVATE_TOKEN")
	selected := ConservativeRequestDiagnostic(raw)
	if strings.Contains(selected.Path, "PRIVATE_") {
		t.Fatal("raw path/query emitted")
	}
	for _, value := range []string{"08006", "did:plc:target", "operation_account_did", "40000000-0000-4000-8000-000000000001"} {
		if !strings.Contains(output.String(), value) {
			t.Errorf("missing selected example field %s", value)
		}
	}
	if strings.Contains(output.String(), "PRIVATE_") {
		t.Fatal("incorrect-example raw data leaked")
	}
}
