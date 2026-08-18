package pdseffects

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func deterministicRecordURI(
	owner syntax.DID,
	collection syntax.NSID,
	rkey syntax.RecordKey,
) (syntax.ATURI, error) {
	if owner == "" || collection == "" || rkey == "" {
		return "", errors.New("PDS record identity is incomplete")
	}
	uri, err := syntax.ParseATURI(fmt.Sprintf(
		"at://%s/%s/%s",
		owner,
		collection,
		rkey,
	))
	if err != nil {
		return "", fmt.Errorf("build deterministic PDS record URI: %w", err)
	}
	return uri, nil
}

func canonicalPutBody(
	owner syntax.DID,
	collection syntax.NSID,
	rkey syntax.RecordKey,
	record any,
	expectedCID syntax.CID,
) ([]byte, [sha256.Size]byte, error) {
	if _, err := deterministicRecordURI(owner, collection, rkey); err != nil {
		return nil, [sha256.Size]byte{}, err
	}
	if record == nil {
		return nil, [sha256.Size]byte{}, errors.New("PDS record body is unavailable")
	}
	request := map[string]any{
		"repo":       owner.String(),
		"collection": collection.String(),
		"rkey":       rkey.String(),
		"record":     record,
	}
	if expectedCID != "" {
		request["swapRecord"] = expectedCID.String()
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, [sha256.Size]byte{}, fmt.Errorf("marshal canonical PDS put body: %w", err)
	}
	return body, sha256.Sum256(body), nil
}

func canonicalDeleteBody(
	owner syntax.DID,
	collection syntax.NSID,
	rkey syntax.RecordKey,
	expectedCID syntax.CID,
) ([]byte, [sha256.Size]byte, syntax.ATURI, error) {
	uri, err := deterministicRecordURI(owner, collection, rkey)
	if err != nil {
		return nil, [sha256.Size]byte{}, "", err
	}
	request := map[string]any{
		"repo":       owner.String(),
		"collection": collection.String(),
		"rkey":       rkey.String(),
	}
	if expectedCID != "" {
		request["swapRecord"] = expectedCID.String()
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, [sha256.Size]byte{}, "", fmt.Errorf("marshal canonical PDS delete body: %w", err)
	}
	return body, sha256.Sum256(body), uri, nil
}
