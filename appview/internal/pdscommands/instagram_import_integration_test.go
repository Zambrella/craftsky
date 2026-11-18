package pdscommands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/instagram"
)

func TestInstagramImportCreateOnlyIdentityAndConfirmedHistory(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	addressed, pds, base := postEditCommandFixture(t, &now)
	service, err := NewAppendCommandService(AppendCommandServiceConfig{Store: addressed.store, Lifecycles: addressed.lifecycles, NewBoundary: addressed.newBoundary, NewRecordKey: func() (syntax.RecordKey, error) { t.Fatal("import allocated ordinary identity"); return "", nil }, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	const sourceKey = "abcdef0123456789abcdef0123456789"
	rkey, err := instagram.PostImportRecordKey(sourceKey, time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	history := instagram.NewPostImportStore(addressed.store.pool)
	request := AppendCommandRequest{Owner: base.Owner, OwnerGeneration: 1, SessionID: base.SessionID, OperationKind: "post.create", OperationKey: uuid.New(), Collection: "social.craftsky.feed.post", SelectedRkey: rkey, Intent: json.RawMessage(`{"text":"historical caption"}`),
		BuildRecord: func(time.Time) (json.RawMessage, error) {
			return json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"historical caption","createdAt":"2020-06-15T12:00:00Z","externalImport":{"source":"instagram"},"sponsored":false}`), nil
		},
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 201, ResponseBody: record.Record}, nil
		}, Rejected: setCommandRejectedResult,
	}
	request.SelectRecordKey = func(ctx context.Context, tx pgx.Tx) (syntax.RecordKey, error) {
		return instagram.ReservePostImport(ctx, tx, base.Owner, sourceKey, rkey)
	}
	request.FinalizeAccepted = func(ctx context.Context, tx pgx.Tx, command PreparedCommand, record AuthoritativeRecord) error {
		return instagram.FinalizePostImport(ctx, tx, base.Owner, instagram.PostImport{SourceKey: sourceKey, URI: record.URI, Kind: "project", OperationKey: request.OperationKey})
	}
	pds.loseNextResponse = true
	result, err := service.Execute(ctx, request)
	if err != nil || result.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("ambiguous=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	items, err := history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 0 {
		t.Fatalf("unconfirmed import marked posted: %+v %v", items, err)
	}
	result, err = service.Execute(ctx, request)
	if err != nil || result.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("recovery=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	items, err = history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 1 || items[0].Kind != "project" {
		t.Fatalf("confirmed history=%+v %v", items, err)
	}
	other, err := history.Lookup(ctx, "did:plc:other-owner", []string{sourceKey})
	if err != nil || len(other) != 0 {
		t.Fatalf("history crossed accounts=%+v %v", other, err)
	}

	// Another draft or browser can only address the same create-only URI.
	// Changed captions cannot replace it or create a second post.
	changed := request
	changed.OperationKey = uuid.New()
	changed.SelectRecordKey = nil
	changed.Intent = json.RawMessage(`{"text":"edited caption"}`)
	changed.BuildRecord = func(time.Time) (json.RawMessage, error) {
		return json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"edited caption","createdAt":"2020-06-15T12:00:00Z"}`), nil
	}
	result, err = service.Execute(ctx, changed)
	if err != nil || result.State != CommandRejected || pds.applyCalls != 1 {
		t.Fatalf("duplicate changed source=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	for _, record := range pds.records {
		encoded, _ := json.Marshal(record.value)
		if strings.Contains(string(encoded), "edited caption") {
			t.Fatal("duplicate replaced imported record")
		}
	}
	// Private source history survives replay-body compaction.
	now = now.Add(31 * 24 * time.Hour)
	if _, err := addressed.store.CompactExpired(ctx, 20); err != nil {
		t.Fatal(err)
	}
	items, err = history.Lookup(ctx, base.Owner, []string{sourceKey})
	if err != nil || len(items) != 1 {
		t.Fatalf("history lost at compaction=%+v %v", items, err)
	}
}
