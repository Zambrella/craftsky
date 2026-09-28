package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestPlanSetRemovalFiltersInvalidMatchesSortsURIsAndEnforcesAtomicCap(t *testing.T) {
	for _, count := range []int{0, 1, 2, 200, 201} {
		t.Run(fmt.Sprintf("%d valid matches", count), func(t *testing.T) {
			records := make([]AuthoritativeRecord, 0, count+1)
			for i := count - 1; i >= 0; i-- {
				rkey := syntax.NewTIDFromInteger(uint64(i + 1000)).String()
				records = append(records, AuthoritativeRecord{
					URI:    syntax.ATURI("at://did:plc:actor/app.bsky.graph.follow/" + rkey),
					CID:    syntax.CID(fmt.Sprintf("bafy-%03d", i)),
					Record: json.RawMessage(`{"subject":"did:plc:target","createdAt":"2026-09-23T20:00:00Z"}`),
				})
			}
			records = append(records, AuthoritativeRecord{
				URI:    "at://did:plc:actor/app.bsky.graph.follow/3aaaaaaaaaaaz",
				CID:    "bafy-invalid",
				Record: json.RawMessage(`{"subject":"not-a-did","createdAt":"2026-09-23T20:00:00Z"}`),
			})
			steps, err := PlanSetRemoval(records, func(record AuthoritativeRecord) bool {
				var body struct {
					Subject string `json:"subject"`
				}
				return json.Unmarshal(record.Record, &body) == nil && body.Subject == "did:plc:target"
			})
			if count == 201 {
				if !errors.Is(err, ErrAtomicMutationTooLarge) || len(steps) != 0 {
					t.Fatalf("201-write plan steps=%d err=%v", len(steps), err)
				}
				return
			}
			if err != nil || len(steps) != count {
				t.Fatalf("plan steps=%d err=%v, want %d", len(steps), err, count)
			}
			for i := 1; i < len(steps); i++ {
				if steps[i-1].URI >= steps[i].URI {
					t.Fatalf("steps not URI sorted at %d: %s >= %s", i, steps[i-1].URI, steps[i].URI)
				}
			}
		})
	}
}

func TestExecuteSetRemovalRetriesInvalidSwapAndUsesGuardedSingleWriteFallback(t *testing.T) {
	record := AuthoritativeRecord{
		URI: "at://did:plc:actor/app.bsky.graph.follow/3aaaaaaaaaaa2", CID: "bafy-record",
		Record: json.RawMessage(`{"subject":"did:plc:target"}`),
	}
	transport := &setCommandTransportStub{
		heads:       []syntax.CID{"bafy-head-1", "bafy-head-1", "bafy-head-2", "bafy-head-2"},
		pages:       []AuthoritativeRecordPage{{Records: []AuthoritativeRecord{record}}, {Records: []AuthoritativeRecord{record}}},
		applyErrors: []error{ErrRepositorySwapConflict, ErrAtomicWritesUnsupported},
	}
	var delays []time.Duration
	err := ExecuteSetRemoval(
		context.Background(),
		transport,
		"did:plc:actor",
		"app.bsky.graph.follow",
		func(AuthoritativeRecord) bool { return true },
		func(delay time.Duration) { delays = append(delays, delay) },
		func(maxInclusive int) int { return maxInclusive },
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(transport.applyHeads) != 2 || transport.applyHeads[0] != "bafy-head-1" || transport.applyHeads[1] != "bafy-head-2" {
		t.Fatalf("apply heads = %v", transport.applyHeads)
	}
	if len(delays) != 1 || delays[0] != 25*time.Millisecond {
		t.Fatalf("retry delays = %v", delays)
	}
	if transport.fallbackHead != "bafy-head-2" || transport.fallbackRecord != "bafy-record" || transport.fallbackURI != record.URI {
		t.Fatalf("fallback guards = %q %q %q", transport.fallbackHead, transport.fallbackRecord, transport.fallbackURI)
	}
}

func TestExecuteSetRemovalRejectsUnsupportedMultiWriteAndConflictExhaustion(t *testing.T) {
	record := func(rkey string) AuthoritativeRecord {
		return AuthoritativeRecord{
			URI: syntax.ATURI("at://did:plc:actor/app.bsky.graph.follow/" + rkey), CID: syntax.CID("bafy-" + rkey),
			Record: json.RawMessage(`{"subject":"did:plc:target"}`),
		}
	}
	t.Run("unsupported multi-write", func(t *testing.T) {
		transport := &setCommandTransportStub{
			heads:       []syntax.CID{"bafy-head", "bafy-head"},
			pages:       []AuthoritativeRecordPage{{Records: []AuthoritativeRecord{record("first"), record("second")}}},
			applyErrors: []error{ErrAtomicWritesUnsupported},
		}
		err := ExecuteSetRemoval(context.Background(), transport, "did:plc:actor", "app.bsky.graph.follow", func(AuthoritativeRecord) bool { return true }, nil, nil)
		if !errors.Is(err, ErrAtomicWritesUnsupported) || transport.fallbackURI != "" {
			t.Fatalf("error/fallback = %v/%q", err, transport.fallbackURI)
		}
	})
	t.Run("third conflict is terminal", func(t *testing.T) {
		transport := &setCommandTransportStub{
			heads:       []syntax.CID{"h1", "h1", "h2", "h2", "h3", "h3"},
			pages:       []AuthoritativeRecordPage{{}, {}, {}},
			applyErrors: []error{ErrRepositorySwapConflict, ErrRepositorySwapConflict, ErrRepositorySwapConflict},
		}
		// Keep one write in every plan so applyWrites is exercised.
		for i := range transport.pages {
			transport.pages[i].Records = []AuthoritativeRecord{record("only")}
		}
		err := ExecuteSetRemoval(context.Background(), transport, "did:plc:actor", "app.bsky.graph.follow", func(AuthoritativeRecord) bool { return true }, func(time.Duration) {}, func(int) int { return 0 })
		if !errors.Is(err, ErrRepositoryConflict) || len(transport.applyHeads) != 3 {
			t.Fatalf("error/attempts = %v/%d", err, len(transport.applyHeads))
		}
	})
}

type setCommandTransportStub struct {
	heads          []syntax.CID
	headIndex      int
	pages          []AuthoritativeRecordPage
	pageIndex      int
	applyErrors    []error
	applyHeads     []syntax.CID
	fallbackHead   syntax.CID
	fallbackRecord syntax.CID
	fallbackURI    syntax.ATURI
}

func (stub *setCommandTransportStub) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	head := stub.heads[stub.headIndex]
	stub.headIndex++
	return head, nil
}

func (stub *setCommandTransportStub) ListRecords(context.Context, syntax.DID, syntax.NSID, string, int) (AuthoritativeRecordPage, error) {
	page := stub.pages[stub.pageIndex]
	stub.pageIndex++
	return page, nil
}

func (stub *setCommandTransportStub) ApplyWrites(_ context.Context, _ syntax.DID, head syntax.CID, _ []DispatchStep) error {
	stub.applyHeads = append(stub.applyHeads, head)
	return stub.applyErrors[len(stub.applyHeads)-1]
}

func (stub *setCommandTransportStub) DeleteRecordWithRepositorySwap(_ context.Context, _ syntax.DID, uri syntax.ATURI, head, record syntax.CID) error {
	stub.fallbackURI, stub.fallbackHead, stub.fallbackRecord = uri, head, record
	return nil
}

func TestInvalidSwapRetryBudgetUsesExactJitterWindows(t *testing.T) {
	tests := []struct {
		completedAttempt int
		maxMillis        int
		wantRetry        bool
	}{
		{completedAttempt: 1, maxMillis: 25, wantRetry: true},
		{completedAttempt: 2, maxMillis: 50, wantRetry: true},
		{completedAttempt: 3, wantRetry: false},
	}
	for _, test := range tests {
		observedMax := -1
		delay, retry := InvalidSwapRetryDelay(test.completedAttempt, func(maxInclusive int) int {
			observedMax = maxInclusive
			return maxInclusive
		})
		if retry != test.wantRetry {
			t.Fatalf("attempt %d retry=%t, want %t", test.completedAttempt, retry, test.wantRetry)
		}
		if !retry {
			if delay != 0 || observedMax != -1 {
				t.Fatalf("exhausted attempt delay=%s observedMax=%d", delay, observedMax)
			}
			continue
		}
		if observedMax != test.maxMillis || delay != time.Duration(test.maxMillis)*time.Millisecond {
			t.Fatalf("attempt %d delay=%s max=%d", test.completedAttempt, delay, observedMax)
		}
	}
}
