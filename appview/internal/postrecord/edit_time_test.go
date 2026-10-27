package postrecord_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"social.craftsky/appview/internal/postrecord"
)

// IT-010 / IT-014: discovery of a first snapshot does not prove an edit.
func TestObserveEditFirstSnapshotHasNoInferredEdit(t *testing.T) {
	observed := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	state, err := postrecord.ObserveEdit(nil, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"first","createdAt":"2020-01-01T00:00:00Z","sponsored":false}`), "first-cid", "3aaaaaaaaaaaa", observed)
	if err != nil || state.CID != "first-cid" || state.Revision != "3aaaaaaaaaaaa" || state.ContentDigest == [32]byte{} || state.EditedAt != nil || state.TimeSource != "" {
		t.Fatalf("first snapshot state=%+v, %v", state, err)
	}
}

// IT-010 / IT-014: portable author declarations are reported, not confirmed.
func TestObserveEditFirstDeclaredSnapshotUsesReportedTime(t *testing.T) {
	observed := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	declared := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	state, err := postrecord.ObserveEdit(nil, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"first","createdAt":"2020-01-01T00:00:00Z","sponsored":false,"editedAt":"2026-10-25T13:00:00+01:00"}`), "first-cid", "3aaaaaaaaaaaa", observed)
	if err != nil || state.EditedAt == nil || !state.EditedAt.Equal(declared) || state.TimeSource != postrecord.EditReported {
		t.Fatalf("declared snapshot state=%+v, %v", state, err)
	}
}

// IT-010: absent or future declarations on a known content transition use the
// observation instant and explicitly avoid claiming the author's exact time.
func TestObserveEditKnownChangeUsesObservedFallback(t *testing.T) {
	for _, tc := range []struct{ name, oldMetadata, newMetadata string }{
		{"without declaration", "", ""},
		{"removed declaration", `,"editedAt":"2026-10-24T12:00:00Z"`, ""},
		{"future declaration", "", `,"editedAt":"2099-01-01T00:00:00Z"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
			observed := first.Add(24 * time.Hour)
			previous, err := postrecord.ObserveEdit(nil, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2020-01-01T00:00:00Z"`+tc.oldMetadata+`}`), "old-cid", "3aaaaaaaaaaaa", first)
			if err != nil {
				t.Fatal(err)
			}
			state, err := postrecord.ObserveEdit(&previous, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"after","sponsored":false,"createdAt":"2020-01-01T00:00:00Z"`+tc.newMetadata+`}`), "new-cid", "3aaaaaaaaaaab", observed)
			if err != nil || state.CID != "new-cid" || state.Revision != "3aaaaaaaaaaab" || state.EditedAt == nil || !state.EditedAt.Equal(observed) || state.TimeSource != postrecord.EditObserved {
				t.Fatalf("observed transition state=%+v, %v", state, err)
			}
		})
	}
}

// IT-010 / UT-003: metadata-only changes carry the known content edit forward.
func TestObserveEditMetadataOnlyKeepsKnownTime(t *testing.T) {
	for _, tc := range []struct {
		source   postrecord.EditTimeSource
		metadata string
	}{
		{postrecord.EditObserved, ""},
		{postrecord.EditReported, `,"editedAt":"2026-10-25T12:00:00Z"`},
		{postrecord.EditConfirmed, `,"editedAt":"2026-10-25T12:00:00Z"`},
	} {
		t.Run(string(tc.source), func(t *testing.T) {
			first := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
			previous, err := postrecord.ObserveEdit(nil, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"same","sponsored":false,"createdAt":"2020-01-01T00:00:00Z","future":{"integer":9007199254740993},"editedAt":"2026-10-24T12:00:00Z"}`), "old-cid", "3aaaaaaaaaaaa", first)
			if err != nil {
				t.Fatal(err)
			}
			previous.TimeSource = tc.source
			state, err := postrecord.ObserveEdit(&previous, json.RawMessage(`{"future":{"integer":9007199254740993},"sponsored":false,"text":"same","createdAt":"2020-01-01T00:00:00Z","$type":"social.craftsky.feed.post"`+tc.metadata+`}`), "new-cid", "3aaaaaaaaaaab", first.Add(48*time.Hour))
			if err != nil || state.CID != "new-cid" || state.Revision != "3aaaaaaaaaaab" || state.EditedAt == nil || !state.EditedAt.Equal(first) || state.TimeSource != tc.source || state.ContentDigest != previous.ContentDigest {
				t.Fatalf("metadata-only state=%+v, %v", state, err)
			}
		})
	}
}

// IT-010: source revision, never CID spelling or later delivery time, orders state.
func TestObserveEditStaleEventCannotRollBackCurrentState(t *testing.T) {
	currentAt := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	previous, err := postrecord.ObserveEdit(nil, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"newest","sponsored":false,"createdAt":"2020-01-01T00:00:00Z","editedAt":"2026-10-26T12:00:00Z"}`), "a-current-cid", "3aaaaaaaaaaac", currentAt)
	if err != nil {
		t.Fatal(err)
	}
	previous.TimeSource = postrecord.EditConfirmed
	state, err := postrecord.ObserveEdit(&previous, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"older","sponsored":false,"createdAt":"2020-01-01T00:00:00Z","editedAt":"2026-10-25T12:00:00Z"}`), "z-older-cid", "3aaaaaaaaaaab", currentAt.Add(time.Hour))
	if err != nil || !reflect.DeepEqual(state, previous) {
		t.Fatalf("stale event changed current state: got=%+v want=%+v error=%v", state, previous, err)
	}
}
