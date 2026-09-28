package pdscommands

import (
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestShouldCompactOnlyExpiredTerminalCommands(t *testing.T) {
	deadline := time.Date(2026, time.September, 23, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name  string
		state CommandState
		now   time.Time
		want  bool
	}{
		{name: "accepted at deadline", state: CommandAccepted, now: deadline, want: true},
		{name: "rejected after deadline", state: CommandRejected, now: deadline.Add(time.Second), want: true},
		{name: "accepted before deadline", state: CommandAccepted, now: deadline.Add(-time.Nanosecond)},
		{name: "prepared after deadline", state: CommandPrepared, now: deadline.Add(24 * time.Hour)},
		{name: "dispatching after deadline", state: CommandDispatching, now: deadline.Add(24 * time.Hour)},
		{name: "ambiguous after deadline", state: CommandAmbiguous, now: deadline.Add(24 * time.Hour)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ShouldCompact(test.state, deadline, test.now); got != test.want {
				t.Fatalf("ShouldCompact(%q, deadline, now) = %t, want %t", test.state, got, test.want)
			}
		})
	}
}

func TestNewTombstoneRetainsOnlyMinimalScopedIdentity(t *testing.T) {
	var scopedKeyHash [32]byte
	for index := range scopedKeyHash {
		scopedKeyHash[index] = byte(index)
	}
	compactedAt := time.Date(2026, time.September, 23, 12, 0, 0, 0, time.UTC)
	want := Tombstone{
		OwnerDID:      syntax.DID("did:plc:retention"),
		OperationKind: "like.set",
		ScopedKeyHash: scopedKeyHash,
		CompactedAt:   compactedAt,
	}
	got := NewTombstone(want.OwnerDID, want.OperationKind, want.ScopedKeyHash, compactedAt)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NewTombstone() = %#v, want %#v", got, want)
	}

	typeOfTombstone := reflect.TypeOf(got)
	wantFields := []string{"OwnerDID", "OperationKind", "ScopedKeyHash", "CompactedAt"}
	if typeOfTombstone.NumField() != len(wantFields) {
		t.Fatalf("Tombstone has %d fields, want exactly %d", typeOfTombstone.NumField(), len(wantFields))
	}
	for index, wantField := range wantFields {
		if gotField := typeOfTombstone.Field(index).Name; gotField != wantField {
			t.Fatalf("Tombstone field %d = %q, want %q", index, gotField, wantField)
		}
	}
}

func TestReplayExpiresAtIsExactlyTwentyFourHoursAfterTerminalCompletion(t *testing.T) {
	terminalAt := time.Date(2026, time.September, 23, 12, 0, 0, 123, time.UTC)
	want := terminalAt.Add(24 * time.Hour)
	if got := ReplayExpiresAt(terminalAt); !got.Equal(want) {
		t.Fatalf("ReplayExpiresAt() = %s, want %s", got, want)
	}
}
