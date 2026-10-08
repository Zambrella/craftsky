package tap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/tap"
)

type diagnosticProjectorFunc func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error)

func (f diagnosticProjectorFunc) Project(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
	return f(ctx, tx, source)
}

func TestLexiconRejectionWarnsWithUsefulReasonWithoutIssue(t *testing.T) {
	cases := []struct {
		name        string
		nsid        syntax.NSID
		key         syntax.RecordKey
		record      json.RawMessage
		explanation string
	}{
		{"missing sponsorship", "social.craftsky.feed.post", "3aaaaaaaaaaa2", json.RawMessage(`{"text":"private-canary","createdAt":"2026-10-07T00:00:00Z"}`), "missing required field: sponsored"},
		{"oversized business image", "social.craftsky.business.profile", "self", json.RawMessage(`{"products":[{"title":"private-canary","uri":"https://shop.example/yarn","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":11189477}}}]}`), "lexicon size limit (11189477 bytes)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dispatcher := index.NewTransactionalDispatcher()
			action := ""
			dispatcher.Register(tc.nsid, diagnosticProjectorFunc(func(_ context.Context, _ pgx.Tx, event ingestion.SourceRecord) (tap.Outcome, error) {
				action = event.Action
				return tap.Applied(), nil
			}))
			event := tap.Event{URI: syntax.ATURI("at://did:plc:actor/" + tc.nsid.String() + "/" + tc.key.String()), DID: "did:plc:actor", Collection: tc.nsid, Rkey: tc.key, Action: "create", ID: 178, Record: tc.record}
			outcome, err := dispatcher.Project(context.Background(), nil, ingestion.SourceRecord{URI: event.URI, DID: event.DID, Collection: event.Collection, Rkey: event.Rkey, Action: event.Action, Record: event.Record, SourceEventID: event.ID})
			if err != nil || outcome.Kind != tap.OutcomePermanentInvalid || action != "delete" {
				t.Fatalf("processing changed: %+v %v action=%s", outcome, err, action)
			}
			var logs bytes.Buffer
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&logs, nil)))
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
			input := tap.RecordFailureDiagnostic(event, outcome, err)
			observability.LogDiagnostic(context.Background(), logger, input)
			observer.CaptureDiagnostic(context.Background(), input)
			observer.Flush(time.Second)
			for _, want := range []string{`"level":"WARN"`, "invalid_lexicon", tc.explanation, event.URI.String(), "178"} {
				if !strings.Contains(logs.String(), want) {
					t.Errorf("missing %s: %s", want, logs.String())
				}
			}
			if strings.Contains(logs.String(), "private-canary") {
				t.Error("record body leaked")
			}
			serialized, _ := json.Marshal(transport.Events())
			if strings.Contains(string(serialized), "private-canary") {
				t.Error("private payload reached SDK")
			}
			issues, logsCount := 0, 0
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 {
					issues++
				}
				for _, entry := range event.Logs {
					logsCount++
					if !strings.Contains(entry.Body, tc.explanation) || string(entry.Level) != "warn" {
						t.Errorf("unhelpful remote validation log: %+v", entry)
					}
				}
			}
			if issues != 0 || logsCount != 1 {
				t.Errorf("issues=%d logs=%d", issues, logsCount)
			}
		})
	}
}
