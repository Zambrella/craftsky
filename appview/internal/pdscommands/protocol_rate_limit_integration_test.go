package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestRateLimitedSetCommandResumesSameKeyAfterCooldown(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 23, 21, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:set-command-owner")
	target := syntax.DID("did:plc:set-command-target")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',3,1,'test',$3,$3,$3),($2,'active',5,1,'test',$3,$3,$3)
	`, owner, target, now); err != nil {
		t.Fatal(err)
	}
	commandStore, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	boundary := &setCommandBoundary{client: pds}
	service, err := NewSetCommandService(SetCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) { return boundary, nil },
		Now:         func() time.Time { return now }, Sleep: func(time.Duration) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	subject := "at://did:plc:set-command-target/social.craftsky.feed.post/one"
	base := SetCommandRequest{
		Owner: owner, OwnerGeneration: 3, Target: target, TargetGeneration: 5, SessionID: "session",
		Collection: "social.craftsky.feed.like", Intent: json.RawMessage(`{"subject":"` + subject + `"}`),
		Matches: func(record AuthoritativeRecord) bool {
			var value struct {
				Subject string `json:"subject"`
			}
			return json.Unmarshal(record.Record, &value) == nil && value.Subject == subject
		},
		CreateRecord: func(createdAt time.Time) (json.RawMessage, error) {
			return json.Marshal(map[string]any{"subject": subject, "createdAt": createdAt.Format(time.RFC3339Nano)})
		},
		AcceptedPresent: func(record AuthoritativeRecord, created bool) (TerminalResult, error) {
			status := 200
			if created {
				status = 201
			}
			body, _ := json.Marshal(map[string]string{"uri": record.URI.String(), "cid": record.CID.String()})
			return TerminalResult{State: CommandAccepted, HTTPStatus: status, ResponseBody: body}, nil
		},
		AcceptedAbsent: func() TerminalResult { return TerminalResult{State: CommandAccepted, HTTPStatus: 204} },
		Rejected:       setCommandRejectedResult,
	}

	request := base
	request.OperationKind = "like"
	request.OperationKey = uuid.New()
	request.SelectedRkey = "3aaaaaaaaaaa1"
	request.DesiredActive = true
	pds.applyErr = &auth.RateLimitError{RetryAfter: 120 * time.Second}
	_, err = service.Execute(ctx, request)
	var limited *auth.RateLimitError
	if !errors.As(err, &limited) || limited.RetrySeconds() != 120 || pds.applyCalls != 1 || len(pds.records) != 0 {
		t.Fatalf("initial throttle: error=%v calls=%d", err, pds.applyCalls)
	}
	_, err = service.Execute(ctx, request)
	if !errors.As(err, &limited) || pds.applyCalls != 1 {
		t.Fatalf("cooldown replay: error=%v calls=%d", err, pds.applyCalls)
	}
	now = now.Add(121 * time.Second)
	pds.applyErr = nil
	accepted, err := service.Execute(ctx, request)
	if err != nil || accepted.State != CommandAccepted || pds.applyCalls != 2 || len(pds.records) != 1 {
		t.Fatalf("resumption: result=%+v error=%v calls=%d", accepted, err, pds.applyCalls)
	}
	replay, err := service.Execute(ctx, request)
	if err != nil || replay.State != CommandAccepted || pds.applyCalls != 2 {
		t.Fatalf("accepted replay: result=%+v error=%v calls=%d", replay, err, pds.applyCalls)
	}
	var commands, throttled, acceptedDispatches int
	err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM pds_commands), (SELECT count(*) FROM pds_command_dispatches WHERE outcome='rate_limited'),(SELECT count(*) FROM pds_command_dispatches WHERE outcome='accepted')`).Scan(&commands, &throttled, &acceptedDispatches)
	if err != nil || commands != 1 || throttled != 1 || acceptedDispatches != 1 {
		t.Fatalf("durability: commands=%d throttled=%d accepted=%d error=%v", commands, throttled, acceptedDispatches, err)
	}
}
