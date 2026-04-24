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
func (i *IndigoPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) error {
	nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
	if err != nil {
		return fmt.Errorf("parse nsid: %w", err)
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
		return translateGetRecordError(err)
	}
	if m, ok := out.(*map[string]any); ok {
		if v, ok := resp.Value.(map[string]any); ok {
			*m = v
			return nil
		}
		return fmt.Errorf("getRecord value has unexpected type %T", resp.Value)
	}
	return fmt.Errorf("unsupported out type %T", out)
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
