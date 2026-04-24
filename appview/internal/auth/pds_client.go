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
var ErrRecordNotFound = errors.New("pds: record not found")

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
}

// PDSClientFactory builds a PDSClient scoped to a caller's OAuth session.
// Handler factories and the OAuth callback both take one of these rather
// than build clients directly, so tests can supply a mock without standing
// up indigo.
type PDSClientFactory func(ctx context.Context, did syntax.DID, oauthSessionID string) (PDSClient, error)
