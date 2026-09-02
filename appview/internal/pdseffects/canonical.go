package pdseffects

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"

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
	normalizedRecord, err := normalizeRecordJSON(record)
	if err != nil {
		return nil, [sha256.Size]byte{}, err
	}
	request := map[string]any{
		"repo":       owner.String(),
		"collection": collection.String(),
		"rkey":       rkey.String(),
		"record":     normalizedRecord,
	}
	if expectedCID == "*" {
		request["swapRecord"] = nil
	} else if expectedCID != "" {
		request["swapRecord"] = expectedCID.String()
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, [sha256.Size]byte{}, fmt.Errorf("marshal canonical PDS put body: %w", err)
	}
	return body, sha256.Sum256(body), nil
}

// RecordContentFingerprint is the normalized record-only join key shared by
// durable PDS Put attempts and Tap ingestion. Conditional-write metadata is
// deliberately excluded and remains in the request fingerprint/expected CID.
func RecordContentFingerprint(
	owner syntax.DID,
	collection syntax.NSID,
	rkey syntax.RecordKey,
	record any,
) ([sha256.Size]byte, error) {
	_, fingerprint, err := canonicalPutBody(owner, collection, rkey, record, "")
	return fingerprint, err
}

func normalizeRecordJSON(record any) (any, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal PDS record body: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("decode PDS record body: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("PDS record body contains multiple JSON values")
		}
		return nil, fmt.Errorf("finish PDS record body: %w", err)
	}
	return normalized, nil
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
