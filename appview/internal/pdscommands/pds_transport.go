package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/recordjson"
)

type PDSProtocol interface {
	auth.PDSRecordLister
	auth.RepositoryCommandPDSClient
	GetRecord(context.Context, syntax.DID, string, string, any) (string, error)
}

func (transport *PDSTransport) GetRecord(
	ctx context.Context,
	repo syntax.DID,
	uri syntax.ATURI,
) (AuthoritativeRecord, error) {
	collection, rkey, err := uriParts(uri)
	if err != nil {
		return AuthoritativeRecord{}, err
	}
	var value any
	cid, err := transport.client.GetRecord(ctx, repo, collection.String(), rkey.String(), &value)
	if err != nil {
		return AuthoritativeRecord{}, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return AuthoritativeRecord{}, fmt.Errorf("marshal authoritative record %s: %w", uri, err)
	}
	return AuthoritativeRecord{URI: uri, CID: syntax.CID(cid), Record: raw}, nil
}

type PDSTransport struct {
	client PDSProtocol
}

func NewPDSTransport(client PDSProtocol) (*PDSTransport, error) {
	if client == nil {
		return nil, errors.New("PDS command transport requires client")
	}
	return &PDSTransport{client: client}, nil
}

func (transport *PDSTransport) LatestCommit(ctx context.Context, repo syntax.DID) (syntax.CID, error) {
	return transport.client.LatestCommit(ctx, repo)
}

func (transport *PDSTransport) LatestCommitMetadata(ctx context.Context, repo syntax.DID) (auth.RepositoryCommit, error) {
	client, ok := transport.client.(auth.RepositoryHeadMetadataClient)
	if !ok {
		return auth.RepositoryCommit{}, auth.ErrCommitMetadataUnavailable
	}
	commit, err := client.LatestCommitMetadata(ctx, repo)
	if err != nil {
		return auth.RepositoryCommit{}, err
	}
	if commit.CID == "" || commit.Revision == "" {
		return auth.RepositoryCommit{}, auth.ErrCommitMetadataUnavailable
	}
	return commit, nil
}

func (transport *PDSTransport) hasWriteReceipts() bool {
	_, ok := transport.client.(auth.RepositoryWriteMetadataClient)
	return ok
}

// observeRecord brackets a read with an unchanged repository head. Its revision
// is an observation bound, never evidence of the record's original write commit.
func (transport *PDSTransport) observeRecord(ctx context.Context, repo syntax.DID, uri syntax.ATURI) (AuthoritativeRecord, error) {
	for attempt := 0; attempt < 3; attempt++ {
		before, err := transport.LatestCommitMetadata(ctx, repo)
		if err != nil {
			return AuthoritativeRecord{}, err
		}
		record, err := transport.GetRecord(ctx, repo, uri)
		if err != nil {
			return AuthoritativeRecord{}, err
		}
		after, err := transport.LatestCommitMetadata(ctx, repo)
		if err != nil {
			return AuthoritativeRecord{}, err
		}
		if before == after {
			record.ObservationCID = after.CID
			record.ObservationRevision = after.Revision
			return record, nil
		}
	}
	return AuthoritativeRecord{}, ErrRepositoryConflict
}

func (transport *PDSTransport) ApplyWritesWithCommit(ctx context.Context, repo syntax.DID, head syntax.CID, steps []DispatchStep) (auth.RepositoryWriteReceipt, error) {
	client, ok := transport.client.(auth.RepositoryWriteMetadataClient)
	if !ok {
		return auth.RepositoryWriteReceipt{}, auth.ErrCommitMetadataUnavailable
	}
	writes, err := repositoryWrites(steps)
	if err != nil {
		return auth.RepositoryWriteReceipt{}, err
	}
	receipt, err := client.ApplyWritesWithCommit(ctx, repo, head, writes)
	return receipt, translateProtocolError(err)
}

func (transport *PDSTransport) ListRecords(
	ctx context.Context,
	repo syntax.DID,
	collection syntax.NSID,
	cursor string,
	limit int,
) (AuthoritativeRecordPage, error) {
	records, next, err := transport.client.ListRecords(ctx, repo, collection.String(), cursor, limit)
	if err != nil {
		return AuthoritativeRecordPage{}, err
	}
	page := AuthoritativeRecordPage{Records: make([]AuthoritativeRecord, len(records)), NextCursor: next}
	for index, record := range records {
		raw, err := json.Marshal(record.Value)
		if err != nil {
			return AuthoritativeRecordPage{}, fmt.Errorf("marshal authoritative record %s: %w", record.URI, err)
		}
		page.Records[index] = AuthoritativeRecord{URI: record.URI, CID: record.CID, Record: raw}
	}
	return page, nil
}

func (transport *PDSTransport) ApplyWrites(
	ctx context.Context,
	repo syntax.DID,
	head syntax.CID,
	steps []DispatchStep,
) error {
	writes, err := repositoryWrites(steps)
	if err != nil {
		return err
	}
	return translateProtocolError(transport.client.ApplyWrites(ctx, repo, head, writes))
}

func repositoryWrites(steps []DispatchStep) ([]auth.RepositoryWrite, error) {
	writes := make([]auth.RepositoryWrite, len(steps))
	for index, step := range steps {
		collection, rkey, err := uriParts(step.URI)
		if err != nil {
			return nil, err
		}
		write := auth.RepositoryWrite{Action: step.Action, Collection: collection, RKey: rkey, ExpectedCID: step.ExpectedCID}
		if len(step.Body) > 0 {
			var value any
			if err := recordjson.Unmarshal(step.Body, &value); err != nil {
				return nil, fmt.Errorf("decode dispatch record: %w", err)
			}
			write.Record = value
		}
		writes[index] = write
	}
	return writes, nil
}

func (transport *PDSTransport) DeleteRecordWithRepositorySwap(
	ctx context.Context,
	repo syntax.DID,
	uri syntax.ATURI,
	head syntax.CID,
	record syntax.CID,
) error {
	collection, rkey, err := uriParts(uri)
	if err != nil {
		return err
	}
	err = transport.client.DeleteRecordWithRepositorySwap(ctx, repo, collection, rkey, head, record)
	return translateProtocolError(err)
}

func uriParts(uri syntax.ATURI) (syntax.NSID, syntax.RecordKey, error) {
	parsed, err := syntax.ParseATURI(uri.String())
	if err != nil {
		return "", "", err
	}
	return parsed.Collection(), parsed.RecordKey(), nil
}

func translateProtocolError(err error) error {
	var apiErr *atclient.APIError
	var limited *auth.RateLimitError
	switch {
	case errors.As(err, &limited):
		return err
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests:
		return &auth.RateLimitError{RetryAfter: auth.RetryAfter("", time.Now()), Cause: err}
	case errors.Is(err, auth.ErrRepositorySwapConflict):
		return errors.Join(ErrRepositorySwapConflict, err)
	case errors.Is(err, auth.ErrApplyWritesUnsupported):
		return errors.Join(ErrAtomicWritesUnsupported, err)
	case errors.As(err, &apiErr) && apiErr.StatusCode >= http.StatusBadRequest && apiErr.StatusCode < http.StatusInternalServerError:
		return errors.Join(ErrDispatchRejected, err)
	default:
		return err
	}
}
