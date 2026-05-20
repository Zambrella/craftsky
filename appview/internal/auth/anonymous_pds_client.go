// appview/internal/auth/anonymous_pds_client.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// AnonymousPDSClient is a read-only PDSClient that resolves each caller's
// PDS URL from their DID doc and talks to it via an unauthenticated
// atclient.APIClient. com.atproto.repo.getRecord is defined as public in
// the atproto lexicon; no DPoP or OAuth session is required.
//
// Used by the Bluesky backfill path in internal/index: when
// CraftskyProfile.Handle commits a new membership row we fetch the user's
// app.bsky.actor.profile record here and feed it back through
// BlueskyProfile.Handle as a synthesised tap.Event.
type AnonymousPDSClient struct {
	dir     identity.Directory
	timeout time.Duration
}

var _ PDSClient = (*AnonymousPDSClient)(nil)

// NewAnonymousPDSClient returns a client that honours the given per-request
// HTTP timeout. Tap's ACK timeout is ~10s; values in the 2–5s range keep
// backfill from wedging the pipeline on a slow PDS.
func NewAnonymousPDSClient(dir identity.Directory, timeout time.Duration) *AnonymousPDSClient {
	return &AnonymousPDSClient{dir: dir, timeout: timeout}
}

// ErrReadOnlyPDSClient is returned when a caller tries to write through
// the anonymous client. The interface satisfies PDSClient for convenience
// (single dependency type) but writes have no meaning here.
var ErrReadOnlyPDSClient = errors.New("pds: read-only client")

// GetRecord resolves the caller's PDS URL from their DID doc, then calls
// com.atproto.repo.getRecord. RecordNotFound errors are translated to
// ErrRecordNotFound via the shared translateGetRecordError helper.
func (c *AnonymousPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (string, error) {
	ident, err := c.dir.LookupDID(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("resolve did %s: %w", repo, err)
	}
	host := ident.PDSEndpoint()
	if host == "" {
		return "", fmt.Errorf("did %s: no atproto_pds service endpoint in DID doc", repo)
	}

	api := atclient.NewAPIClient(host)
	// NewAPIClient assigns `http.DefaultClient` to `api.Client`. Setting
	// `api.Client.Timeout = c.timeout` (as spec §4.2 step 3 prescribes)
	// would mutate the process-wide default. Replace the whole client
	// with our own so the short per-request timeout applies only here.
	// This is an intentional, narrower deviation from the spec wording.
	api.Client = &http.Client{Timeout: c.timeout}

	nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
	if err != nil {
		return "", fmt.Errorf("parse nsid: %w", err)
	}
	var resp struct {
		URI   string `json:"uri"`
		CID   string `json:"cid"`
		Value any    `json:"value"`
	}
	params := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"rkey":       rkey,
	}
	if err := api.Get(ctx, nsid, params, &resp); err != nil {
		return "", translateGetRecordError(err)
	}
	if resp.CID == "" {
		return "", fmt.Errorf("getRecord: PDS returned empty cid for %s/%s", collection, rkey)
	}
	if m, ok := out.(*map[string]any); ok {
		if v, ok := resp.Value.(map[string]any); ok {
			*m = v
			return resp.CID, nil
		}
		return "", fmt.Errorf("getRecord value has unexpected type %T", resp.Value)
	}
	return "", fmt.Errorf("unsupported out type %T", out)
}

// PutRecord is not supported by the anonymous client.
func (c *AnonymousPDSClient) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
	return ErrReadOnlyPDSClient
}

// CreateRecord is not supported by the anonymous client.
func (c *AnonymousPDSClient) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	return "", "", ErrReadOnlyPDSClient
}

// DeleteRecord is not supported by the anonymous client.
func (c *AnonymousPDSClient) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	return ErrReadOnlyPDSClient
}

// UploadBlob is not supported by the anonymous client.
func (c *AnonymousPDSClient) UploadBlob(_ context.Context, _ string, _ []byte) (*UploadedBlob, error) {
	return nil, ErrReadOnlyPDSClient
}
