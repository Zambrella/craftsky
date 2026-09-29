package ingestion

import (
	"errors"
	"slices"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrRepositorySnapshotUnverified = errors.New("repository snapshot is not verified")

// RepositorySnapshotRecord is one record decoded from a complete repository.
type RepositorySnapshotRecord struct {
	URI    syntax.ATURI
	CID    syntax.CID
	Record []byte
}

func (snapshot VerifiedRepositorySnapshot) Collection(
	did syntax.DID,
	collection syntax.NSID,
) (syntax.CID, []RepositorySnapshotRecord, error) {
	if snapshot.verified == nil || snapshot.verified.did != did {
		return "", nil, ErrRepositorySnapshotUnverified
	}
	records := make([]RepositorySnapshotRecord, 0)
	for _, record := range snapshot.verified.records {
		if record.URI.Collection() != collection {
			continue
		}
		record.Record = slices.Clone(record.Record)
		records = append(records, record)
	}
	return snapshot.verified.root, records, nil
}

// VerifiedRepositorySnapshot can only receive its verification marker from the
// concrete bounded repository fetcher in this package.
type VerifiedRepositorySnapshot struct {
	verified *verifiedRepositorySnapshot
}

type verifiedRepositorySnapshot struct {
	did      syntax.DID
	revision syntax.TID
	root     syntax.CID
	records  []RepositorySnapshotRecord
}

func newVerifiedRepositorySnapshot(
	did syntax.DID,
	revision syntax.TID,
	root syntax.CID,
	records []RepositorySnapshotRecord,
) VerifiedRepositorySnapshot {
	cloned := make([]RepositorySnapshotRecord, len(records))
	for i, record := range records {
		record.Record = append([]byte(nil), record.Record...)
		cloned[i] = record
	}
	return VerifiedRepositorySnapshot{verified: &verifiedRepositorySnapshot{
		did: did, revision: revision, root: root, records: cloned,
	}}
}
