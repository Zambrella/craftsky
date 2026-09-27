package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const MaxAtomicWrites = 200

var ErrAtomicMutationTooLarge = errors.New("PDS atomic mutation exceeds 200 writes")

var (
	ErrRepositorySwapConflict  = errors.New("PDS repository changed before atomic mutation")
	ErrRepositoryConflict      = errors.New("PDS repository remained unstable after three attempts")
	ErrAtomicWritesUnsupported = errors.New("PDS atomic applyWrites is unsupported")
)

type AuthoritativeRecord struct {
	URI    syntax.ATURI
	CID    syntax.CID
	Record json.RawMessage
}

func InvalidSwapRetryDelay(completedAttempt int, jitterMillis func(maxInclusive int) int) (time.Duration, bool) {
	var maxMillis int
	switch completedAttempt {
	case 1:
		maxMillis = 25
	case 2:
		maxMillis = 50
	default:
		return 0, false
	}
	if jitterMillis == nil {
		return 0, true
	}
	millis := jitterMillis(maxMillis)
	if millis < 0 {
		millis = 0
	} else if millis > maxMillis {
		millis = maxMillis
	}
	return time.Duration(millis) * time.Millisecond, true
}

func PlanSetRemoval(records []AuthoritativeRecord, matches func(AuthoritativeRecord) bool) ([]DispatchStep, error) {
	matched := make([]AuthoritativeRecord, 0, len(records))
	for _, record := range records {
		if record.URI == "" || record.CID == "" || !json.Valid(record.Record) {
			return nil, errors.New("authoritative record is incomplete")
		}
		if _, err := syntax.ParseATURI(record.URI.String()); err != nil {
			return nil, err
		}
		if matches(record) {
			matched = append(matched, record)
		}
	}
	if len(matched) > MaxAtomicWrites {
		return nil, ErrAtomicMutationTooLarge
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].URI < matched[j].URI })
	steps := make([]DispatchStep, len(matched))
	for i, record := range matched {
		steps[i] = DispatchStep{Action: "delete", URI: record.URI, ExpectedCID: record.CID}
	}
	return steps, nil
}

type AuthoritativeSetTransport interface {
	AuthoritativeReaderTransport
	ApplyWrites(context.Context, syntax.DID, syntax.CID, []DispatchStep) error
	DeleteRecordWithRepositorySwap(context.Context, syntax.DID, syntax.ATURI, syntax.CID, syntax.CID) error
}

func ExecuteSetRemoval(
	ctx context.Context,
	transport AuthoritativeSetTransport,
	repo syntax.DID,
	collection syntax.NSID,
	matches func(AuthoritativeRecord) bool,
	sleep func(time.Duration),
	jitterMillis func(int) int,
) error {
	reader, err := NewAuthoritativeReader(transport)
	if err != nil {
		return err
	}
	for attempt := 1; attempt <= 3; attempt++ {
		head, records, err := reader.CompleteCollection(ctx, repo, collection)
		if err != nil {
			return err
		}
		steps, err := PlanSetRemoval(records, matches)
		if err != nil {
			return err
		}
		if len(steps) == 0 {
			return nil
		}
		err = transport.ApplyWrites(ctx, repo, head, steps)
		if errors.Is(err, ErrAtomicWritesUnsupported) {
			if len(steps) != 1 {
				return ErrAtomicWritesUnsupported
			}
			err = transport.DeleteRecordWithRepositorySwap(ctx, repo, steps[0].URI, head, steps[0].ExpectedCID)
		}
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrRepositorySwapConflict) {
			return err
		}
		delay, retry := InvalidSwapRetryDelay(attempt, jitterMillis)
		if !retry {
			return ErrRepositoryConflict
		}
		if sleep != nil {
			sleep(delay)
		}
	}
	return ErrRepositoryConflict
}
