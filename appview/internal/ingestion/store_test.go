package ingestion

import (
	"encoding/json"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestSelectSourceVersion(t *testing.T) {
	current := sourceVersion{
		revision: "3aaaaaaaaaaa2",
		cid:      cidPointer("bafycurrent"),
		action:   "create",
		record:   json.RawMessage(`{"value":"current"}`),
	}
	for _, test := range []struct {
		name     string
		current  *sourceVersion
		incoming sourceVersion
		want     sourceVersionDecision
	}{
		{name: "first version", incoming: current, want: sourceVersionNewer},
		{name: "newer revision", current: &current, incoming: sourceVersion{
			revision: "3aaaaaaaaaaa3", cid: cidPointer("bafynew"), action: "update", record: json.RawMessage(`{"value":"new"}`),
		}, want: sourceVersionNewer},
		{name: "stale revision", current: &current, incoming: sourceVersion{
			revision: "3aaaaaaaaaaa1", cid: cidPointer("bafystale"), action: "update", record: json.RawMessage(`{"value":"stale"}`),
		}, want: sourceVersionStale},
		{name: "exact duplicate", current: &current, incoming: current, want: sourceVersionDuplicate},
		{name: "equal revision changed cid", current: &current, incoming: sourceVersion{
			revision: current.revision, cid: cidPointer("bafyconflict"), action: current.action, record: current.record,
		}, want: sourceVersionConflict},
		{name: "equal revision changed action", current: &current, incoming: sourceVersion{
			revision: current.revision, action: "delete",
		}, want: sourceVersionConflict},
		{name: "equal revision changed record", current: &current, incoming: sourceVersion{
			revision: current.revision, cid: current.cid, action: current.action, record: json.RawMessage(`{"value":"conflict"}`),
		}, want: sourceVersionConflict},
		{name: "duplicate delete", current: &sourceVersion{
			revision: current.revision, action: "delete",
		}, incoming: sourceVersion{
			revision: current.revision, action: "delete",
		}, want: sourceVersionDuplicate},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := selectSourceVersion(test.current, test.incoming); got != test.want {
				t.Fatalf("selectSourceVersion() = %d, want %d", got, test.want)
			}
		})
	}
}

func cidPointer(value syntax.CID) *string {
	raw := value.String()
	return &raw
}
