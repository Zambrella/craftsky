package moderation

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type SubjectType string

const (
	SubjectPost    SubjectType = "post"
	SubjectAccount SubjectType = "account"
	SubjectEvent   SubjectType = "event"
)

type AcceptedReport struct {
	ID                      string
	SubjectType             SubjectType
	SubjectDID              syntax.DID
	SubjectCollection       string
	SubjectRkey             string
	SubjectURI              syntax.ATURI
	SubjectCIDSnapshot      string
	SubmittedHandleSnapshot string
	CreatedAt               time.Time
}

type Case struct {
	ID         uuid.UUID
	SubjectKey string
	State      string
	Revision   int64
	CreatedAt  time.Time
}
