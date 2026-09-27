package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestAuthoritativeReaderFailsClosedOnUnstableOrMalformedPagination(t *testing.T) {
	collection := syntax.NSID("app.bsky.graph.follow")
	record := AuthoritativeRecord{
		URI: "at://did:plc:reader/app.bsky.graph.follow/3aaaaaaaaaaa2", CID: "bafy-record",
		Record: json.RawMessage(`{"subject":"did:plc:target","createdAt":"2026-09-23T20:00:00Z"}`),
	}
	tests := []struct {
		name      string
		transport *readerTransportStub
		wantErr   error
		wantCount int
	}{
		{
			name: "stable pages",
			transport: &readerTransportStub{
				heads: []syntax.CID{"bafy-head", "bafy-head"},
				pages: map[string]AuthoritativeRecordPage{"": {Records: []AuthoritativeRecord{record}}},
			},
			wantCount: 1,
		},
		{
			name: "head changes",
			transport: &readerTransportStub{
				heads: []syntax.CID{"bafy-before", "bafy-after"},
				pages: map[string]AuthoritativeRecordPage{"": {Records: []AuthoritativeRecord{record}}},
			},
			wantErr: ErrRepositoryChanged,
		},
		{
			name: "cursor loops",
			transport: &readerTransportStub{
				heads: []syntax.CID{"bafy-head", "bafy-head"},
				pages: map[string]AuthoritativeRecordPage{"": {NextCursor: "again"}, "again": {NextCursor: "again"}},
			},
			wantErr: ErrUnsafeRepositoryRead,
		},
		{
			name: "duplicate URI",
			transport: &readerTransportStub{
				heads: []syntax.CID{"bafy-head", "bafy-head"},
				pages: map[string]AuthoritativeRecordPage{"": {Records: []AuthoritativeRecord{record, record}}},
			},
			wantErr: ErrUnsafeRepositoryRead,
		},
		{
			name: "out of scope collection",
			transport: &readerTransportStub{
				heads: []syntax.CID{"bafy-head", "bafy-head"},
				pages: map[string]AuthoritativeRecordPage{"": {Records: []AuthoritativeRecord{{URI: "at://did:plc:reader/app.bsky.graph.block/3aaaaaaaaaaa2", CID: "bafy-record", Record: record.Record}}}},
			},
			wantErr: ErrUnsafeRepositoryRead,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, err := NewAuthoritativeReader(test.transport)
			if err != nil {
				t.Fatal(err)
			}
			_, records, err := reader.CompleteCollection(context.Background(), "did:plc:reader", collection)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("read error = %v, want %v", err, test.wantErr)
			}
			if err == nil && len(records) != test.wantCount {
				t.Fatalf("record count = %d, want %d", len(records), test.wantCount)
			}
		})
	}
}

func TestAuthoritativeReaderEnforcesConfiguredPageRecordAndByteLimits(t *testing.T) {
	record := AuthoritativeRecord{
		URI: "at://did:plc:reader/app.bsky.graph.follow/3aaaaaaaaaaa2", CID: "bafy-record",
		Record: json.RawMessage(`{"subject":"did:plc:target"}`),
	}
	tests := []struct {
		name   string
		limits AuthoritativeReaderLimits
		pages  map[string]AuthoritativeRecordPage
	}{
		{name: "page limit", limits: AuthoritativeReaderLimits{MaxPages: 1, MaxRecords: 10, MaxBytes: 1000}, pages: map[string]AuthoritativeRecordPage{"": {NextCursor: "next"}, "next": {}}},
		{name: "record limit", limits: AuthoritativeReaderLimits{MaxPages: 2, MaxRecords: 1, MaxBytes: 1000}, pages: map[string]AuthoritativeRecordPage{"": {Records: []AuthoritativeRecord{record, {URI: "at://did:plc:reader/app.bsky.graph.follow/3aaaaaaaaaaa3", CID: "bafy-other", Record: record.Record}}}}},
		{name: "byte limit", limits: AuthoritativeReaderLimits{MaxPages: 2, MaxRecords: 10, MaxBytes: int64(len(record.Record) - 1)}, pages: map[string]AuthoritativeRecordPage{"": {Records: []AuthoritativeRecord{record}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transport := &readerTransportStub{heads: []syntax.CID{"head", "head"}, pages: test.pages}
			reader, err := NewAuthoritativeReaderWithLimits(transport, test.limits)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = reader.CompleteCollection(context.Background(), "did:plc:reader", "app.bsky.graph.follow")
			if !errors.Is(err, ErrUnsafeRepositoryRead) {
				t.Fatalf("error = %v, want ErrUnsafeRepositoryRead", err)
			}
		})
	}
}

type readerTransportStub struct {
	heads     []syntax.CID
	headIndex int
	pages     map[string]AuthoritativeRecordPage
}

func (stub *readerTransportStub) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	head := stub.heads[stub.headIndex]
	stub.headIndex++
	return head, nil
}

func (stub *readerTransportStub) ListRecords(_ context.Context, _ syntax.DID, _ syntax.NSID, cursor string, _ int) (AuthoritativeRecordPage, error) {
	return stub.pages[cursor], nil
}
