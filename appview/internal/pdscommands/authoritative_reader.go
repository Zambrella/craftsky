package pdscommands

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ingestion"
)

var (
	ErrRepositoryChanged    = errors.New("PDS repository changed during authoritative read")
	ErrUnsafeRepositoryRead = errors.New("PDS repository collection read is unsafe")
)

const authoritativePageSize = 100

type AuthoritativeReaderLimits struct {
	MaxPages   int
	MaxRecords int
	MaxBytes   int64
}

var defaultAuthoritativeReaderLimits = AuthoritativeReaderLimits{
	MaxPages: 1000, MaxRecords: 100_000, MaxBytes: 64 << 20,
}

type AuthoritativeRecordPage struct {
	Records    []AuthoritativeRecord
	NextCursor string
}

type AuthoritativeReaderTransport interface {
	LatestCommit(context.Context, syntax.DID) (syntax.CID, error)
	ListRecords(context.Context, syntax.DID, syntax.NSID, string, int) (AuthoritativeRecordPage, error)
}

type AuthoritativeReader struct {
	transport AuthoritativeReaderTransport
	limits    AuthoritativeReaderLimits
	fallback  AuthoritativeSnapshotFallback
}

type AuthoritativeSnapshotFallback interface {
	FetchCollection(context.Context, syntax.DID, syntax.NSID) (ingestion.VerifiedRepositorySnapshot, error)
}

func NewAuthoritativeReader(transport AuthoritativeReaderTransport) (*AuthoritativeReader, error) {
	if transport == nil {
		return nil, errors.New("authoritative reader requires transport")
	}
	return NewAuthoritativeReaderWithLimits(transport, defaultAuthoritativeReaderLimits)
}

func NewAuthoritativeReaderWithLimits(
	transport AuthoritativeReaderTransport,
	limits AuthoritativeReaderLimits,
) (*AuthoritativeReader, error) {
	if transport == nil {
		return nil, errors.New("authoritative reader requires transport")
	}
	if limits.MaxPages <= 0 || limits.MaxRecords <= 0 || limits.MaxBytes <= 0 {
		return nil, errors.New("authoritative reader limits must be positive")
	}
	return &AuthoritativeReader{transport: transport, limits: limits}, nil
}

func NewAuthoritativeReaderWithFallback(
	transport AuthoritativeReaderTransport,
	limits AuthoritativeReaderLimits,
	fallback AuthoritativeSnapshotFallback,
) (*AuthoritativeReader, error) {
	reader, err := NewAuthoritativeReaderWithLimits(transport, limits)
	if err != nil {
		return nil, err
	}
	if fallback == nil {
		return nil, errors.New("authoritative snapshot fallback is unavailable")
	}
	reader.fallback = fallback
	return reader, nil
}

func (reader *AuthoritativeReader) CompleteCollection(
	ctx context.Context,
	repo syntax.DID,
	collection syntax.NSID,
) (syntax.CID, []AuthoritativeRecord, error) {
	before, err := reader.transport.LatestCommit(ctx, repo)
	if err != nil {
		return "", nil, err
	}
	var records []AuthoritativeRecord
	cursor := ""
	seenCursors := map[string]struct{}{"": {}}
	seenURIs := make(map[syntax.ATURI]struct{})
	pagesRead := 0
	var bytesRead int64
	for {
		pagesRead++
		if pagesRead > reader.limits.MaxPages {
			return "", nil, ErrUnsafeRepositoryRead
		}
		page, err := reader.transport.ListRecords(ctx, repo, collection, cursor, authoritativePageSize)
		if err != nil {
			return "", nil, err
		}
		for _, record := range page.Records {
			bytesRead += int64(len(record.Record))
			if len(records)+1 > reader.limits.MaxRecords || bytesRead > reader.limits.MaxBytes {
				return "", nil, ErrUnsafeRepositoryRead
			}
			parsed, err := syntax.ParseATURI(record.URI.String())
			if err != nil || parsed.Authority().DID() != repo || parsed.Collection() != collection || record.CID == "" || !json.Valid(record.Record) {
				return "", nil, ErrUnsafeRepositoryRead
			}
			if _, duplicate := seenURIs[record.URI]; duplicate {
				return "", nil, ErrUnsafeRepositoryRead
			}
			seenURIs[record.URI] = struct{}{}
			records = append(records, record)
		}
		if page.NextCursor == "" {
			break
		}
		if _, loop := seenCursors[page.NextCursor]; loop {
			return "", nil, ErrUnsafeRepositoryRead
		}
		seenCursors[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
	}
	after, err := reader.transport.LatestCommit(ctx, repo)
	if err != nil {
		return "", nil, err
	}
	if before != after {
		if reader.fallback != nil {
			snapshot, fetchErr := reader.fallback.FetchCollection(ctx, repo, collection)
			if fetchErr != nil {
				return "", nil, fetchErr
			}
			root, snapshotRecords, verifyErr := snapshot.Collection(repo, collection)
			if verifyErr != nil {
				return "", nil, verifyErr
			}
			converted := make([]AuthoritativeRecord, len(snapshotRecords))
			for index, record := range snapshotRecords {
				converted[index] = AuthoritativeRecord{URI: record.URI, CID: record.CID, Record: record.Record}
			}
			return root, converted, nil
		}
		return "", nil, ErrRepositoryChanged
	}
	return before, records, nil
}
