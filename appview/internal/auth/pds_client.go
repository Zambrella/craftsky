// appview/internal/auth/pds_client.go
package auth

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ErrRecordNotFound is the canonical "getRecord returned 404" sentinel
// used across this package. PDSClient implementations wrap whatever
// their upstream library raises into this value.
var (
	ErrRecordNotFound               = errors.New("pds: record not found")
	ErrRecordSwapConflict           = errors.New("pds: record changed before conditional mutation")
	ErrConditionalPutUnsupported    = errors.New("pds: conditional record Put is unsupported")
	ErrConditionalDeleteUnsupported = errors.New("pds: conditional record delete is unsupported")
)

// PDSClient is the minimal surface users of this package exercise against
// the caller's PDS. In production it's an adapter over indigo's
// atclient.APIClient; in tests it's a hand-rolled mock.
//
// All record bodies are passed and returned as already-decoded Go values
// (typically map[string]any) — the adapter handles JSON encoding.
//
// GetRecord returns the record CID alongside the decoded value. cid is
// always populated on success and empty on error.
type PDSClient interface {
	GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) (cid string, err error)
	PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error
	CreateRecord(ctx context.Context, repo syntax.DID, collection string, record any) (uri syntax.ATURI, cid syntax.CID, err error)
	DeleteRecord(ctx context.Context, repo syntax.DID, collection string, rkey string) error
	UploadBlob(ctx context.Context, contentType string, body []byte) (*UploadedBlob, error)
}

// ConditionalPDSRecordDeleter is the compare-and-swap delete capability for
// ordinary record effects. Implementations must send expectedCID as
// com.atproto.repo.deleteRecord's swapRecord precondition and must never fall
// back to an unconditional delete.
type ConditionalPDSRecordDeleter interface {
	DeleteRecordWithSwap(
		ctx context.Context,
		repo syntax.DID,
		collection string,
		rkey string,
		expectedCID syntax.CID,
	) error
}

// ConditionalPDSRecordPutter is the compare-and-swap Put capability for
// ordinary record effects. Implementations must send expectedCID as
// com.atproto.repo.putRecord's swapRecord precondition and must never fall
// back to an unconditional Put.
type ConditionalPDSRecordPutter interface {
	PutRecordWithSwap(
		ctx context.Context,
		repo syntax.DID,
		collection string,
		rkey string,
		record any,
		expectedCID syntax.CID,
	) error
}

type PDSRecord struct {
	URI   syntax.ATURI
	CID   syntax.CID
	Value any
}

// PDSRecordLister is a narrow optional capability used by public block
// reconciliation when Tap has not projected a newly written record yet.
type PDSRecordLister interface {
	ListRecords(
		ctx context.Context,
		repo syntax.DID,
		collection string,
		cursor string,
		limit int,
	) (records []PDSRecord, nextCursor string, err error)
}

// DeletionPDSClient is the deliberately closed capability available to the
// CraftSky account-deletion worker. It cannot delete an AT Protocol account,
// mutate other records, upload/delete blobs, or read record bodies directly.
type DeletionPDSClient interface {
	PDSRecordLister
	DeleteRecord(ctx context.Context, repo syntax.DID, collection string, rkey string) error
}

// UploadedBlob is normalized metadata returned from com.atproto.repo.uploadBlob.
// Raw preserves the atproto blob object for pass-through into later record writes.
type UploadedBlob struct {
	Raw  map[string]any
	CID  string
	MIME string
	Size int64
}

// PDSClientFactory builds a PDSClient scoped to a caller's OAuth session.
// Handler factories and the OAuth callback both take one of these rather
// than build clients directly, so tests can supply a mock without standing
// up indigo.
type PDSClientFactory func(ctx context.Context, did syntax.DID, oauthSessionID string) (PDSClient, error)
