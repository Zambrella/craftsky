package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

func TestBusinessRecordsPermanentDeletionIsOrderedRetrySafeAndScoped(t *testing.T) {
	owner := syntax.DID("did:plc:business-delete-owner")
	for _, test := range []struct {
		name              string
		interruptAfter    string
		membershipPresent bool
	}{
		{name: "events", interruptAfter: "events", membershipPresent: true},
		{name: "declaration", interruptAfter: "declaration", membershipPresent: true},
		{name: "membership", interruptAfter: "membership", membershipPresent: true},
		{name: "membership already absent", membershipPresent: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pds := newBusinessDeletionPDS(owner, test.membershipPresent, test.interruptAfter)
			deleter := NewPDSDeleter(pds, 1)

			_, err := deleter.DeleteAll(context.Background(), owner)
			if test.interruptAfter != "" {
				if err == nil {
					t.Fatal("first deletion unexpectedly survived injected interruption")
				}
				if _, err := deleter.DeleteAll(context.Background(), owner); err != nil {
					t.Fatalf("retry deletion: %v", err)
				}
			} else if err != nil {
				t.Fatalf("delete with absent membership: %v", err)
			}

			wantEffects := []string{"existing/post", "event/first", "event/second", "profile/self"}
			if test.membershipPresent {
				wantEffects = append(wantEffects, "membership/self")
			}
			if !reflect.DeepEqual(pds.effects, wantEffects) {
				t.Fatalf("deletion effects = %v, want %v", pds.effects, wantEffects)
			}
			remaining := 0
			for _, records := range pds.records {
				remaining += len(records)
			}
			if remaining != 0 {
				t.Fatalf("business records remain: %v", pds.records)
			}
			if !pds.finalRegistryScanObserved {
				t.Fatal("complete registry was not scanned after membership stage")
			}
			if !reflect.DeepEqual(pds.foreignRecords, []syntax.ATURI{
				"at://did:plc:business-delete-owner/app.bsky.feed.post/keep",
			}) || pds.blobs != 2 || !pds.didActive || !pds.pdsAccountActive {
				t.Fatalf(
					"out-of-scope state changed: foreign=%v blobs=%d did=%t pds=%t",
					pds.foreignRecords, pds.blobs, pds.didActive, pds.pdsAccountActive,
				)
			}
		})
	}
}

type businessDeletionPDS struct {
	owner                     syntax.DID
	records                   map[syntax.NSID][]auth.PDSRecord
	effects                   []string
	interruptAfter            string
	interrupted               bool
	membershipStageComplete   bool
	finalRegistryScanObserved bool
	foreignRecords            []syntax.ATURI
	blobs                     int
	didActive                 bool
	pdsAccountActive          bool
}

func newBusinessDeletionPDS(owner syntax.DID, membershipPresent bool, interruptAfter string) *businessDeletionPDS {
	records := map[syntax.NSID][]auth.PDSRecord{
		"social.craftsky.feed.post": {
			{URI: syntax.ATURI(fmt.Sprintf("at://%s/social.craftsky.feed.post/post", owner))},
		},
		"social.craftsky.business.event": {
			{URI: syntax.ATURI(fmt.Sprintf("at://%s/social.craftsky.business.event/first", owner))},
			{URI: syntax.ATURI(fmt.Sprintf("at://%s/social.craftsky.business.event/second", owner))},
		},
		"social.craftsky.business.profile": {
			{URI: syntax.ATURI(fmt.Sprintf("at://%s/social.craftsky.business.profile/self", owner))},
		},
	}
	if membershipPresent {
		records["social.craftsky.actor.profile"] = []auth.PDSRecord{
			{URI: syntax.ATURI(fmt.Sprintf("at://%s/social.craftsky.actor.profile/self", owner))},
		}
	}
	return &businessDeletionPDS{
		owner: owner, records: records, interruptAfter: interruptAfter,
		foreignRecords: []syntax.ATURI{syntax.ATURI(fmt.Sprintf("at://%s/app.bsky.feed.post/keep", owner))},
		blobs:          2, didActive: true, pdsAccountActive: true,
	}
}

func (pds *businessDeletionPDS) ListRecords(
	_ context.Context,
	repo syntax.DID,
	collection string,
	_ string,
	limit int,
) ([]auth.PDSRecord, string, error) {
	if repo != pds.owner || !isDeletionCollection(syntax.NSID(collection)) {
		return nil, "", errors.New("list escaped deletion scope")
	}
	if !pds.interrupted {
		switch {
		case pds.interruptAfter == "events" && collection == "social.craftsky.business.profile" && len(pds.records["social.craftsky.business.event"]) == 0:
			pds.interrupted = true
			return nil, "", errors.New("interrupted after events")
		case pds.interruptAfter == "declaration" && collection == "social.craftsky.actor.profile" && len(pds.records["social.craftsky.business.profile"]) == 0:
			pds.interrupted = true
			return nil, "", errors.New("interrupted after declaration")
		case pds.interruptAfter == "membership" && pds.membershipStageComplete:
			pds.interrupted = true
			return nil, "", errors.New("interrupted after membership")
		}
	}
	if pds.membershipStageComplete || (collection == "social.craftsky.feed.post" && len(pds.records["social.craftsky.business.profile"]) == 0 && len(pds.records["social.craftsky.business.event"]) == 0) {
		pds.finalRegistryScanObserved = true
	}
	records := pds.records[syntax.NSID(collection)]
	if len(records) > limit {
		records = records[:limit]
	}
	return append([]auth.PDSRecord(nil), records...), "", nil
}

func (pds *businessDeletionPDS) DeleteRecord(
	_ context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
) error {
	if repo != pds.owner || !isDeletionCollection(syntax.NSID(collection)) {
		return errors.New("delete escaped deletion scope")
	}
	nsid := syntax.NSID(collection)
	for index, record := range pds.records[nsid] {
		if record.URI.RecordKey().String() != rkey {
			continue
		}
		pds.records[nsid] = append(pds.records[nsid][:index], pds.records[nsid][index+1:]...)
		switch nsid {
		case "social.craftsky.feed.post":
			pds.effects = append(pds.effects, "existing/"+rkey)
		case "social.craftsky.business.event":
			pds.effects = append(pds.effects, "event/"+rkey)
		case "social.craftsky.business.profile":
			pds.effects = append(pds.effects, "profile/"+rkey)
		case "social.craftsky.actor.profile":
			pds.effects = append(pds.effects, "membership/"+rkey)
			pds.membershipStageComplete = true
		}
		return nil
	}
	return auth.ErrRecordNotFound
}

var _ auth.DeletionPDSClient = (*businessDeletionPDS)(nil)
