package accountdeletion

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type fakeExpiredIntentSource struct {
	intents []ExpiredIntent
	err     error
	limit   int
}

func (source *fakeExpiredIntentSource) ListExpiredIntents(_ context.Context, limit int) ([]ExpiredIntent, error) {
	source.limit = limit
	return source.intents, source.err
}

type fakeIntentExpirer struct {
	errors map[uuid.UUID]error
	seen   []ExpiredIntent
}

func (expirer *fakeIntentExpirer) ExpireIntent(_ context.Context, jobID uuid.UUID, owner syntax.DID) error {
	expirer.seen = append(expirer.seen, ExpiredIntent{JobID: jobID, Owner: owner})
	return expirer.errors[jobID]
}

func TestIntentExpiryProcessorIsBoundedAndTreatsAConcurrentAcceptanceAsComplete(t *testing.T) {
	first := ExpiredIntent{JobID: uuid.New(), Owner: "did:plc:expiry-one"}
	second := ExpiredIntent{JobID: uuid.New(), Owner: "did:plc:expiry-two"}
	source := &fakeExpiredIntentSource{intents: []ExpiredIntent{first, second}}
	expirer := &fakeIntentExpirer{errors: map[uuid.UUID]error{second.JobID: ErrPointOfNoReturn}}
	processor, err := NewIntentExpiryProcessor(IntentExpiryProcessorOptions{
		Source: source, Expirer: expirer, BatchSize: 17,
	})
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 2 || source.limit != 17 || len(expirer.seen) != 2 {
		t.Fatalf("processed=%d limit=%d seen=%v", processed, source.limit, expirer.seen)
	}
}

func TestIntentExpiryProcessorStopsOnInfrastructureFailure(t *testing.T) {
	first := ExpiredIntent{JobID: uuid.New(), Owner: "did:plc:expiry-failure"}
	want := errors.New("database unavailable")
	processor, err := NewIntentExpiryProcessor(IntentExpiryProcessorOptions{
		Source:    &fakeExpiredIntentSource{intents: []ExpiredIntent{first}},
		Expirer:   &fakeIntentExpirer{errors: map[uuid.UUID]error{first.JobID: want}},
		BatchSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := processor.RunOnce(context.Background())
	if processed != 0 || !errors.Is(err, want) {
		t.Fatalf("processed=%d error=%v", processed, err)
	}
}
