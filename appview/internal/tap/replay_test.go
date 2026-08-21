package tap_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/tap"
)

func TestReplayEnvelopeReclassifiesWithoutAcknowledgingOrRequarantining(t *testing.T) {
	ingestor := &replayIngestor{}
	outcome, err := tap.ReplayEnvelope(context.Background(), []byte(`{
		"id":71,"type":"record","record":{
			"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:replay",
			"collection":"social.craftsky.feed.post","rkey":"one",
			"action":"create","cid":"bafy-replay","record":{"text":"fixed"}
		}
	}`), ingestor)
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
	if ingestor.records != 1 || ingestor.quarantines != 0 {
		t.Fatalf("records=%d quarantines=%d", ingestor.records, ingestor.quarantines)
	}
}

func TestReplayEnvelopeLeavesStillInvalidInputPending(t *testing.T) {
	ingestor := &replayIngestor{}
	outcome, err := tap.ReplayEnvelope(context.Background(), []byte(`{
		"id":72,"type":"record","record":{
			"live":true,"rev":"3aaaaaaaaaaa2","did":"not-a-did",
			"collection":"social.craftsky.feed.post","rkey":"one",
			"action":"create","cid":"bafy-replay","record":{"text":"fixed"}
		}
	}`), ingestor)
	if err == nil || outcome.Acknowledgable() {
		t.Fatalf("outcome=%+v err=%v", outcome, err)
	}
	if outcome.Reason != tap.ReasonInvalidDID || ingestor.records != 0 || ingestor.quarantines != 0 {
		t.Fatalf("outcome=%+v records=%d quarantines=%d", outcome, ingestor.records, ingestor.quarantines)
	}
}

type replayIngestor struct {
	records     int
	identities  int
	quarantines int
}

func (ingestor *replayIngestor) IngestRecord(context.Context, tap.Event) (tap.Outcome, error) {
	ingestor.records++
	return tap.Applied(), nil
}

func (ingestor *replayIngestor) IngestIdentity(context.Context, tap.IdentityEvent) (tap.Outcome, error) {
	ingestor.identities++
	return tap.Applied(), nil
}

func (ingestor *replayIngestor) Quarantine(context.Context, tap.InvalidEvent) (tap.Outcome, error) {
	ingestor.quarantines++
	return tap.PermanentInvalid(tap.ReasonInvalidEnvelope), nil
}

var _ tap.DurableIngestor = (*replayIngestor)(nil)
