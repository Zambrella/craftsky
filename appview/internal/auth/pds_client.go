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

// PDSClient is the minimal surface InitializeProfile uses against the
// user's PDS. In production it's an adapter over indigo's
// atclient.APIClient; in tests it's a hand-rolled mock.
//
// All record bodies are passed and returned as already-decoded Go values
// (typically map[string]any) — the adapter handles JSON encoding.
type PDSClient interface {
	GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) error
	PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error
}
