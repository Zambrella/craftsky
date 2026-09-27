package pdscommands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

const (
	requestFingerprintDomain  = "craftsky-pds-command-request-v1\n"
	dispatchFingerprintDomain = "craftsky-pds-command-dispatch-v1\n"
)

type ScopedCommandKey struct {
	Owner         syntax.DID
	OperationKind string
	OperationKey  uuid.UUID
}

type BlobReference struct {
	CID      syntax.CID `json:"cid"`
	MIMEType string     `json:"mimeType"`
	Size     int64      `json:"size"`
}

type RequestFingerprintInput struct {
	AlgorithmVersion int             `json:"algorithmVersion"`
	Owner            syntax.DID      `json:"owner"`
	OperationKind    string          `json:"operationKind"`
	Intent           json.RawMessage `json:"intent"`
	Conditions       json.RawMessage `json:"conditions,omitempty"`
	Blobs            []BlobReference `json:"blobs,omitempty"`
}

type DispatchStep struct {
	Action      string          `json:"action"`
	URI         syntax.ATURI    `json:"uri"`
	Body        json.RawMessage `json:"body,omitempty"`
	ExpectedCID syntax.CID      `json:"expectedCid,omitempty"`
}

type DispatchFingerprintInput struct {
	AlgorithmVersion   int             `json:"algorithmVersion"`
	RequestFingerprint [32]byte        `json:"-"`
	RepositoryHead     syntax.CID      `json:"repositoryHead"`
	Generated          json.RawMessage `json:"generated,omitempty"`
	Steps              []DispatchStep  `json:"steps"`
}

func RequestFingerprint(input RequestFingerprintInput) ([32]byte, error) {
	canonical, err := canonicalJSON(input)
	if err != nil {
		return [32]byte{}, err
	}
	return domainSeparatedFingerprint(requestFingerprintDomain, canonical), nil
}

func DispatchFingerprint(input DispatchFingerprintInput) ([32]byte, error) {
	payload := struct {
		AlgorithmVersion   int             `json:"algorithmVersion"`
		RequestFingerprint string          `json:"requestFingerprint"`
		RepositoryHead     syntax.CID      `json:"repositoryHead"`
		Generated          json.RawMessage `json:"generated,omitempty"`
		Steps              []DispatchStep  `json:"steps"`
	}{
		AlgorithmVersion:   input.AlgorithmVersion,
		RequestFingerprint: hex.EncodeToString(input.RequestFingerprint[:]),
		RepositoryHead:     input.RepositoryHead,
		Generated:          input.Generated,
		Steps:              input.Steps,
	}
	canonical, err := canonicalJSON(payload)
	if err != nil {
		return [32]byte{}, err
	}
	return domainSeparatedFingerprint(dispatchFingerprintDomain, canonical), nil
}

func domainSeparatedFingerprint(domain string, canonical []byte) [32]byte {
	input := make([]byte, 0, len(domain)+len(canonical))
	input = append(input, domain...)
	input = append(input, canonical...)
	return sha256.Sum256(input)
}
