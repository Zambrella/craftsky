package moderation

import (
	"encoding/json"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// SubjectSnapshot is the complete subject data permitted across moderation
// presentation boundaries. Persistence JSON must never be returned directly.
type SubjectSnapshot struct {
	Type            SubjectType  `json:"type"`
	DID             syntax.DID   `json:"did"`
	Collection      string       `json:"collection,omitempty"`
	Rkey            string       `json:"rkey,omitempty"`
	URI             syntax.ATURI `json:"uri,omitempty"`
	CID             string       `json:"cid,omitempty"`
	SubmittedHandle string       `json:"submittedHandle,omitempty"`
}

func presentSubjectSnapshot(raw json.RawMessage, fallback SubjectSnapshot) (SubjectSnapshot, error) {
	var snapshot SubjectSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return SubjectSnapshot{}, err
	}
	if snapshot.Type == "" || snapshot.DID == "" {
		return fallback, nil
	}
	if snapshot.Collection == "" {
		snapshot.Collection = fallback.Collection
	}
	if snapshot.Rkey == "" {
		snapshot.Rkey = fallback.Rkey
	}
	if snapshot.URI == "" {
		snapshot.URI = fallback.URI
	}
	if snapshot.CID == "" {
		snapshot.CID = fallback.CID
	}
	if snapshot.SubmittedHandle == "" {
		snapshot.SubmittedHandle = fallback.SubmittedHandle
	}
	return snapshot, nil
}

type DecisionPresentation struct {
	Reason          Reason
	Action          string
	AppealAvailable bool
	InternalDetail  string
}

type DecisionNotice struct {
	Reason          string `json:"reason"`
	Action          string `json:"action"`
	AppealAvailable bool   `json:"appealAvailable"`
}

func PresentDecisionNotice(presentation DecisionPresentation) DecisionNotice {
	reason := "Community standards violation"
	if presentation.Reason == ReasonChildSafety {
		reason = "Child safety violation"
	}
	return DecisionNotice{Reason: reason, Action: presentation.Action, AppealAvailable: presentation.AppealAvailable}
}
