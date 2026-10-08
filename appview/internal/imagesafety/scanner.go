package imagesafety

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrUnknownFixture = errors.New("unknown image scanner fixture")

type ScanInput struct {
	BlobCID  syntax.CID
	MIMEType string
	Size     int64
	Content  io.Reader
}

type ScanResult struct {
	State                      State
	ProviderReference          string
	IntegrityMetadataReference string
}

type MatchDetection struct {
	ResultID                   uuid.UUID
	ProviderReference          string
	IntegrityMetadataReference string
	DetectedAt                 time.Time
}

type MatchRecorder interface {
	RecordMatchTx(context.Context, pgx.Tx, MatchDetection) error
}

type Scanner interface {
	Scan(context.Context, ScanInput) (ScanResult, error)
}

type FixtureScanner struct {
	results map[syntax.CID]State
}

func NewFixtureScanner(results map[syntax.CID]State) *FixtureScanner {
	copy := make(map[syntax.CID]State, len(results))
	for cid, state := range results {
		copy[cid] = state
	}
	return &FixtureScanner{results: copy}
}

func (scanner *FixtureScanner) Scan(_ context.Context, input ScanInput) (ScanResult, error) {
	state, ok := scanner.results[input.BlobCID]
	if !ok || !state.Valid() || state == StatePending {
		return ScanResult{State: StateError}, ErrUnknownFixture
	}
	return ScanResult{State: state}, nil
}
