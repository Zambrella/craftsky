package moderation

import (
	"context"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type recordingCommander struct {
	command TrustedCommand
	calls   int
}

func (r *recordingCommander) ResolveCase(_ context.Context, command TrustedCommand) (CommandResult, error) {
	r.command = command
	r.calls++
	return CommandResult{CaseID: command.CaseID, Revision: command.ExpectedRevision + 1}, nil
}

func TestTrustedAdapterNormalizesWithoutInvokingCommander(t *testing.T) {
	commander := &recordingCommander{}
	adapter := NewTrustedInputAdapter(commander)
	caseID := uuid.New()
	request := ResolveRequest{
		CaseID: caseID, ExpectedRevision: 4,
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectStrike}},
		ActorID:  "spoofed", SourceSystem: "spoofed", SourceDID: syntax.DID("did:plc:spoofed"),
	}
	sources := []AuthenticatedSource{
		{SourceSystem: "admin-api", ActorID: "trusted-moderator", SourceDID: syntax.DID("did:plc:labeler")},
		{SourceSystem: "future-adapter", ActorID: "trusted-ingest", SourceDID: syntax.DID("did:plc:labeler")},
	}
	for _, authenticated := range sources {
		command, err := adapter.Normalize(authenticated, "adapter-replay-0001", request)
		if err != nil {
			t.Fatalf("source %s: %v", authenticated.SourceSystem, err)
		}
		if commander.calls != 0 {
			t.Fatalf("normalization invoked commander %d times", commander.calls)
		}
		if command.ActorID != authenticated.ActorID || command.SourceSystem != authenticated.SourceSystem ||
			command.SourceDID != authenticated.SourceDID || command.ReplayID != "adapter-replay-0001" ||
			command.CaseID != caseID || command.ExpectedRevision != 4 || !reflect.DeepEqual(command.Decision, request.Decision) {
			t.Fatalf("source %s normalized command = %+v", authenticated.SourceSystem, command)
		}
	}
}

func TestTrustedAdapterRejectsMissingAuthenticatedSourceIdentity(t *testing.T) {
	adapter := NewTrustedInputAdapter(&recordingCommander{})
	valid := AuthenticatedSource{SourceSystem: "admin-api", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler")}
	tests := []struct {
		name     string
		source   AuthenticatedSource
		replayID string
	}{
		{name: "source system", source: AuthenticatedSource{ActorID: valid.ActorID, SourceDID: valid.SourceDID}, replayID: "replay-1"},
		{name: "actor", source: AuthenticatedSource{SourceSystem: valid.SourceSystem, SourceDID: valid.SourceDID}, replayID: "replay-1"},
		{name: "source DID", source: AuthenticatedSource{SourceSystem: valid.SourceSystem, ActorID: valid.ActorID}, replayID: "replay-1"},
		{name: "replay identity", source: valid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := adapter.Normalize(test.source, test.replayID, ResolveRequest{}); err != ErrInvalidCommand {
				t.Fatalf("Normalize() error = %v, want %v", err, ErrInvalidCommand)
			}
		})
	}
}

func TestTrustedAdapterInvokesAdjudicationCommander(t *testing.T) {
	commander := &recordingCommander{}
	adapter := NewTrustedInputAdapter(commander)
	command, err := adapter.Normalize(
		AuthenticatedSource{SourceSystem: "future-adapter", ActorID: "trusted-ingest", SourceDID: syntax.DID("did:plc:labeler")},
		"stable-source-replay-1",
		ResolveRequest{
			CaseID: uuid.New(), ExpectedRevision: 2,
			Decision: Decision{Disposition: DispositionNoAction, Evidence: "not substantiated"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Invoke(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if commander.calls != 1 || !reflect.DeepEqual(commander.command, command) || result.Revision != 3 {
		t.Fatalf("command = %+v, calls = %d, result = %+v", commander.command, commander.calls, result)
	}
}
