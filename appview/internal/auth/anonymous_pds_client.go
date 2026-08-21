// appview/internal/auth/anonymous_pds_client.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

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
	client  *http.Client
	origins PDSOriginValidator
}

// PDSOriginValidator is implemented by the process-wide federated HTTP
// boundary. A DID document is untrusted input, so its atproto_pds service URL
// is revalidated immediately before an XRPC client is constructed.
type PDSOriginValidator interface {
	ValidateOrigin(context.Context, string) (*url.URL, error)
}

var _ PDSClient = (*AnonymousPDSClient)(nil)

// NewAnonymousPDSClient requires the process-wide hardened PDS client and
// destination policy. It deliberately has no default-client fallback.
func NewAnonymousPDSClient(
	dir identity.Directory,
	client *http.Client,
	origins PDSOriginValidator,
) (*AnonymousPDSClient, error) {
	if dir == nil || client == nil || client.Transport == nil || origins == nil {
		return nil, errors.New("pds: anonymous client requires hardened identity, transport, and origin policy")
	}
	return &AnonymousPDSClient{dir: dir, client: client, origins: origins}, nil
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
	validatedHost, err := c.origins.ValidateOrigin(ctx, host)
	if err != nil {
		return "", err
	}

	api := atclient.NewAPIClient(validatedHost.String())
	api.Client = c.client

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
