package pdseffects

import (
	"context"
	"errors"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// ReadRecordRequest describes a lifecycle-fenced, authenticated PDS record
// read. It deliberately exposes no raw PDS client to the caller.
type ReadRecordRequest struct {
	Owner           syntax.DID
	OwnerGeneration int64
	ExpectedOwners  []ownerlifecycle.ExpectedOwner
	Collection      syntax.NSID
	Rkey            syntax.RecordKey
}

// ReadRecord returns the authoritative record CID after reading inside the
// same owner/session boundary used by durable mutations. Reads do not create
// effect attempts because they cannot change remote state.
func (executor *Executor) ReadRecord(
	ctx context.Context,
	request ReadRecordRequest,
	out any,
) (syntax.CID, error) {
	if executor == nil || executor.attempts == nil || executor.boundary == nil {
		return "", errors.New("durable PDS effect executor is unavailable")
	}
	if err := validateReadScope(executor.owner, request); err != nil {
		return "", err
	}
	if out == nil {
		return "", errors.New("PDS record read output is unavailable")
	}

	var cid string
	err := executor.boundary.WithActiveEffects(
		ctx,
		request.ExpectedOwners,
		func(effectCtx context.Context, client auth.PDSClient) error {
			callCtx, cancel := context.WithTimeout(effectCtx, executor.timeout)
			defer cancel()
			var err error
			cid, err = client.GetRecord(
				callCtx,
				request.Owner,
				request.Collection.String(),
				request.Rkey.String(),
				out,
			)
			if err != nil {
				return err
			}
			if strings.TrimSpace(cid) == "" {
				return errors.New("authoritative PDS record has no CID")
			}
			return nil
		},
	)
	return syntax.CID(cid), err
}

func validateReadScope(executorOwner syntax.DID, request ReadRecordRequest) error {
	if executorOwner == "" || request.Owner != executorOwner {
		return ErrExecutorOwnerMismatch
	}
	if request.OwnerGeneration <= 0 || len(request.ExpectedOwners) == 0 ||
		request.Collection == "" || request.Rkey == "" {
		return errors.New("PDS record read identity is incomplete")
	}
	found := false
	for _, item := range request.ExpectedOwners {
		if item.Owner != request.Owner {
			continue
		}
		if item.AllowMissing || item.Generation != request.OwnerGeneration {
			return ownerlifecycle.ErrGenerationChanged
		}
		found = true
	}
	if !found {
		return ownerlifecycle.ErrFenceRequired
	}
	return nil
}
