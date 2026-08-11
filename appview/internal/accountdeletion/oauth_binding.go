package accountdeletion

import (
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrBoundOAuthUnauthorized = errors.New("bound OAuth session unauthorized")

type OAuthBinding struct {
	JobID     string
	Owner     syntax.DID
	SessionID string
	Bound     bool
}

func BoundOAuthSessionForWorker(binding OAuthBinding, jobID string, owner syntax.DID) (string, error) {
	if !binding.Bound || binding.SessionID == "" || binding.JobID != jobID || binding.Owner != owner {
		return "", ErrBoundOAuthUnauthorized
	}
	return binding.SessionID, nil
}

func BoundOAuthSessionForDevice(OAuthBinding) (string, error) {
	return "", ErrBoundOAuthUnauthorized
}

func BindDeletionOAuthSession(existing OAuthBinding, jobID string, owner syntax.DID, sessionID string) (OAuthBinding, error) {
	if jobID == "" || owner == "" || sessionID == "" {
		return OAuthBinding{}, ErrBoundOAuthUnauthorized
	}
	if existing.Bound {
		if existing.JobID == jobID && existing.Owner == owner && existing.SessionID == sessionID {
			return existing, nil
		}
		return OAuthBinding{}, ErrBoundOAuthUnauthorized
	}
	return OAuthBinding{JobID: jobID, Owner: owner, SessionID: sessionID, Bound: true}, nil
}

func ReplaceDeletionOAuthSession(existing OAuthBinding, jobID string, owner syntax.DID, sessionID string, freshProof bool) (OAuthBinding, error) {
	if !freshProof || !existing.Bound || existing.JobID != jobID || existing.Owner != owner || sessionID == "" {
		return OAuthBinding{}, ErrBoundOAuthUnauthorized
	}
	return OAuthBinding{JobID: jobID, Owner: owner, SessionID: sessionID, Bound: true}, nil
}
