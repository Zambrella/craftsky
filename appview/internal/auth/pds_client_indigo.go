// appview/internal/auth/pds_client_indigo.go
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// IndigoPDSClient adapts indigo's *atclient.APIClient to our PDSClient
// interface.
type IndigoPDSClient struct {
	Client *atclient.APIClient
}

var _ PDSClient = (*IndigoPDSClient)(nil)

// GetRecord calls com.atproto.repo.getRecord on the user's PDS. A
// "record missing" response is translated to ErrRecordNotFound so callers
// can switch on presence; see translateGetRecordError for the detection
// rules.
func (i *IndigoPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (string, error) {
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
	if err := i.Client.Get(ctx, nsid, params, &resp); err != nil {
		return "", translateGetRecordError(err)
	}
	// Downstream callers (notably the Bluesky backfiller) write resp.CID
	// into NOT NULL columns. Fail loudly here rather than silently
	// propagate an empty string from a malformed PDS response.
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

// translateGetRecordError maps the PDS response for a missing record into
// ErrRecordNotFound. atproto PDSes signal a missing record with HTTP 400
// + XRPC error name "RecordNotFound" (NOT HTTP 404); we also accept a
// plain HTTP 404 as a fallback in case an upstream variant uses it.
// Returns the original error otherwise (including nil in → nil out).
func translateGetRecordError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *atclient.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	if apiErr.Name == "RecordNotFound" || apiErr.StatusCode == 404 {
		return ErrRecordNotFound
	}
	return err
}

// PutRecord calls com.atproto.repo.putRecord on the user's PDS.
func (i *IndigoPDSClient) PutRecord(ctx context.Context, repo syntax.DID, collection, rkey string, record any) error {
	nsid, err := syntax.ParseNSID("com.atproto.repo.putRecord")
	if err != nil {
		return fmt.Errorf("parse nsid: %w", err)
	}
	body := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"rkey":       rkey,
		"record":     record,
	}
	var resp any
	return i.Client.Post(ctx, nsid, body, &resp)
}

// CreateRecord calls com.atproto.repo.createRecord on the user's PDS.
// Returns the AT-URI and CID assigned by the PDS. The PDS stamps the
// rkey on TID-keyed collections.
func (i *IndigoPDSClient) CreateRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	record any,
) (syntax.ATURI, syntax.CID, error) {
	nsid, err := syntax.ParseNSID("com.atproto.repo.createRecord")
	if err != nil {
		return "", "", fmt.Errorf("parse nsid: %w", err)
	}
	body := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"record":     record,
	}
	var resp struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}
	if err := i.Client.Post(ctx, nsid, body, &resp); err != nil {
		return "", "", err
	}
	if resp.URI == "" || resp.CID == "" {
		return "", "", fmt.Errorf("createRecord: PDS returned empty uri or cid")
	}
	return syntax.ATURI(resp.URI), syntax.CID(resp.CID), nil
}

// DeleteRecord calls com.atproto.repo.deleteRecord on the user's PDS.
// "Record not found" responses are translated to ErrRecordNotFound so
// callers can treat delete-of-already-deleted as idempotent success.
func (i *IndigoPDSClient) DeleteRecord(
	ctx context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
) error {
	nsid, err := syntax.ParseNSID("com.atproto.repo.deleteRecord")
	if err != nil {
		return fmt.Errorf("parse nsid: %w", err)
	}
	body := map[string]any{
		"repo":       repo.String(),
		"collection": collection,
		"rkey":       rkey,
	}
	var resp any
	if err := i.Client.Post(ctx, nsid, body, &resp); err != nil {
		// translateGetRecordError also handles deleteRecord's "RecordNotFound" shape; reused deliberately.
		return translateGetRecordError(err)
	}
	return nil
}
