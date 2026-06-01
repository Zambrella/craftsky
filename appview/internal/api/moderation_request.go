// appview/internal/api/moderation_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type ModerationRequestConfig struct {
	DefaultSourceDID  string
	TrustedSourceDIDs []string
}

type syntheticModerationRequest struct {
	SourceDID      string                     `json:"sourceDid,omitempty"`
	Subject        syntheticModerationSubject `json:"subject"`
	Value          string                     `json:"value"`
	Action         string                     `json:"action"`
	InternalReason *string                    `json:"internalReason,omitempty"`
	ExpiresAt      *string                    `json:"expiresAt,omitempty"`
}

type syntheticModerationSubject struct {
	Type string `json:"type"`
	DID  string `json:"did"`
	Rkey string `json:"rkey,omitempty"`
}

// DecodeSyntheticModerationRequest validates the dev-only synthetic moderation
// request and translates it into the store input shape. It accepts exactly one
// output object per request; arrays/batches are rejected.
func DecodeSyntheticModerationRequest(body io.Reader, cfg ModerationRequestConfig) (ModerationOutputInput, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return ModerationOutputInput{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": err.Error()}}
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] == '[' {
		return ModerationOutputInput{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "expected single object"}}
	}
	var req syntheticModerationRequest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return ModerationOutputInput{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": err.Error()}}
	}

	fields := map[string]string{}
	sourceDID := strings.TrimSpace(req.SourceDID)
	if sourceDID == "" {
		sourceDID = strings.TrimSpace(cfg.DefaultSourceDID)
	}
	if _, err := syntax.ParseDID(sourceDID); err != nil {
		fields["sourceDid"] = fmt.Sprintf("not a valid DID: %s", err)
	} else if !trustedSourceDID(sourceDID, cfg.TrustedSourceDIDs) {
		return ModerationOutputInput{}, &FieldError{Code: "untrusted_moderation_source", Fields: map[string]string{"sourceDid": "not trusted"}}
	}

	subjectDID, didErr := syntax.ParseDID(req.Subject.DID)
	if didErr != nil {
		fields["subject.did"] = fmt.Sprintf("not a valid DID: %s", didErr)
	}

	var subjectType ModerationSubjectType
	var collection, rkey, subjectURI *string
	switch req.Subject.Type {
	case string(ModerationSubjectPost):
		subjectType = ModerationSubjectPost
		parsedRkey, err := syntax.ParseRecordKey(req.Subject.Rkey)
		if err != nil {
			fields["subject.rkey"] = fmt.Sprintf("not a valid record key: %s", err)
		} else if didErr == nil {
			collectionValue := craftskyPostNSID
			rkeyValue := parsedRkey.String()
			uriValue := "at://" + subjectDID.String() + "/" + craftskyPostNSID + "/" + rkeyValue
			collection = &collectionValue
			rkey = &rkeyValue
			subjectURI = &uriValue
		}
	case string(ModerationSubjectAccount):
		subjectType = ModerationSubjectAccount
		if strings.TrimSpace(req.Subject.Rkey) != "" {
			fields["subject.rkey"] = "must be omitted for account subjects"
		}
	default:
		fields["subject.type"] = "must be post or account"
	}

	value := ModerationValue(req.Value)
	switch value {
	case ModerationValueHide, ModerationValueTakedown, ModerationValueWarn:
	default:
		fields["value"] = "must be hide, takedown, or warn"
	}

	action := ModerationAction(req.Action)
	switch action {
	case ModerationActionApply, ModerationActionNegate:
	default:
		fields["action"] = "must be apply or negate"
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			fields["expiresAt"] = "must be RFC3339 timestamp"
		} else {
			expiresAt = &parsed
		}
	}

	if len(fields) > 0 {
		return ModerationOutputInput{}, &FieldError{Code: "validation_failed", Fields: fields}
	}
	return ModerationOutputInput{
		SourceDID:         sourceDID,
		SubjectType:       subjectType,
		SubjectDID:        subjectDID.String(),
		SubjectCollection: collection,
		SubjectRkey:       rkey,
		SubjectURI:        subjectURI,
		Value:             value,
		Action:            action,
		InternalReason:    req.InternalReason,
		ExpiresAt:         expiresAt,
	}, nil
}

func trustedSourceDID(source string, trusted []string) bool {
	for _, did := range trusted {
		if strings.TrimSpace(did) == source {
			return true
		}
	}
	return false
}
