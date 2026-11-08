package pdscommands

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/guestvisibility"
	"social.craftsky/appview/internal/guestvisibilitydb"
	"social.craftsky/appview/internal/ownerlifecycle"
)

var ErrGuestVisibilityUnknown = errors.New("profile visibility could not be established")

type GuestVisibilityPending struct {
	OperationKey      uuid.UUID  `json:"operationKey"`
	ExpectedCID       syntax.CID `json:"expectedCid"`
	HideFromLoggedOut bool       `json:"hideFromLoggedOut"`
}

type GuestVisibilitySetting struct {
	Pending           *GuestVisibilityPending `json:"pending,omitempty"`
	HideFromLoggedOut bool                    `json:"hideFromLoggedOut"`
	ExpectedCID       syntax.CID              `json:"expectedCid"`
}

type GuestVisibilityPutRequest struct {
	Owner             syntax.DID
	OwnerGeneration   int64
	SessionID         string
	ExpectedCID       syntax.CID
	HideFromLoggedOut bool
	OperationKey      uuid.UUID
	Rejected          func(error) TerminalResult
}

func guestProfileURI(owner syntax.DID) syntax.ATURI {
	return syntax.ATURI("at://" + owner.String() + "/app.bsky.actor.profile/self")
}

// ReadGuestVisibility is an explicit authenticated owner operation, never a
// guest-read repair. Stable observation evidence is persisted under the fence.
func (service *AddressedCommandService) ReadGuestVisibility(ctx context.Context, owner syntax.DID, generation int64, session string) (GuestVisibilitySetting, error) {
	if owner == "" || generation < 1 || session == "" {
		return GuestVisibilitySetting{}, ErrMalformedCommand
	}
	boundary, err := service.newBoundary(ctx, owner, session)
	if err != nil {
		return GuestVisibilitySetting{}, err
	}
	var setting GuestVisibilitySetting
	err = boundary.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: generation}}, func(ctx context.Context, client auth.PDSClient) error {
		protocol, ok := client.(PDSProtocol)
		if !ok {
			return ErrDispatchUnavailable
		}
		transport, err := NewPDSTransport(protocol)
		if err != nil {
			return err
		}
		record, err := observeGuestProfile(ctx, transport, owner)
		if err != nil {
			return err
		}
		preference := guestProfilePreference(record.Record)
		accepted := false
		var pending *GuestVisibilityPending
		unresolvedTerminal := false
		err = service.store.withTransaction(ctx, func(tx pgx.Tx) error {
			queries := guestvisibilitydb.New(tx)
			held, heldErr := queries.LockHeldGuestVisibilityCommand(ctx, owner.String())
			if heldErr != nil && !errors.Is(heldErr, pgx.ErrNoRows) {
				return heldErr
			}
			if heldErr == nil && held.State != string(CommandAccepted) && held.State != string(CommandRejected) {
				if held.OwnerGeneration != generation {
					return ownerlifecycle.ErrGenerationChanged
				}
				var intent struct {
					Hide        *bool       `json:"hideFromLoggedOut"`
					ExpectedCID *syntax.CID `json:"expectedCid"`
				}
				if err := json.Unmarshal(held.ImmutableRequest, &intent); err != nil || intent.Hide == nil || intent.ExpectedCID == nil {
					return ErrDispatchUnavailable
				}
				pending = &GuestVisibilityPending{OperationKey: held.OperationKey.Bytes, ExpectedCID: *intent.ExpectedCID, HideFromLoggedOut: *intent.Hide}
			}
			var err error
			accepted, err = guestvisibility.ApplyTx(ctx, tx, owner, preference, guestvisibility.Evidence{Revision: record.ObservationRevision, CID: record.CID, Kind: guestvisibility.Observation})
			if err != nil {
				return err
			}
			if accepted && preference != guestvisibility.Unknown && heldErr == nil && (held.State == string(CommandAccepted) || held.State == string(CommandRejected)) {
				released, err := queries.ReleaseTerminalGuestVisibilityHold(ctx, guestvisibilitydb.ReleaseTerminalGuestVisibilityHoldParams{Did: owner.String(), GuestVisibilityHoldKey: held.ID})
				if err != nil {
					return err
				}
				unresolvedTerminal = released == 0
			}
			return nil
		})
		if err != nil {
			return err
		}
		if preference == guestvisibility.Unknown {
			return ErrGuestVisibilityUnknown
		}
		if !accepted {
			return ErrRepositoryConflict
		}
		if unresolvedTerminal {
			return ErrDispatchUnavailable
		}
		setting = GuestVisibilitySetting{HideFromLoggedOut: preference == guestvisibility.Hidden, ExpectedCID: record.CID, Pending: pending}
		return nil
	})
	return setting, err
}

func (service *AddressedCommandService) PutGuestVisibility(ctx context.Context, input GuestVisibilityPutRequest) (CommandResult, error) {
	intent, err := json.Marshal(struct {
		Hide        bool       `json:"hideFromLoggedOut"`
		ExpectedCID syntax.CID `json:"expectedCid"`
	}{input.HideFromLoggedOut, input.ExpectedCID})
	if err != nil {
		return CommandResult{}, err
	}
	var commandID uuid.UUID
	finalize := func(ctx context.Context, tx pgx.Tx, command PreparedCommand, record AuthoritativeRecord) error {
		preference := guestProfilePreference(record.Record)
		if preference == guestvisibility.Unknown {
			return ErrGuestVisibilityUnknown
		}
		evidence := guestvisibility.Evidence{Revision: record.Revision, CID: record.CID, Kind: guestvisibility.WriteReceipt}
		if evidence.Revision == "" {
			evidence.Revision = record.ObservationRevision
			evidence.Kind = guestvisibility.Observation
		}
		accepted, err := guestvisibility.ApplyTx(ctx, tx, input.Owner, preference, evidence)
		if err != nil {
			return err
		}
		if !accepted {
			return ErrRepositoryConflict
		}
		return guestvisibilitydb.New(tx).ReleaseGuestVisibilityHold(ctx, guestvisibilitydb.ReleaseGuestVisibilityHoldParams{Did: input.Owner.String(), GuestVisibilityHoldKey: pgtype.UUID{Bytes: command.ID, Valid: true}})
	}
	expectedCID := input.ExpectedCID
	if expectedCID == "" {
		expectedCID = "*"
	}
	result, err := service.Put(ctx, AddressedPutCommandRequest{
		Owner: input.Owner, OwnerGeneration: input.OwnerGeneration, SessionID: input.SessionID,
		OperationKind: "profile.guest_visibility", OperationKey: input.OperationKey, URI: guestProfileURI(input.Owner), ExpectedCID: expectedCID, StrictExpectedCID: true, recoverGuestVisibility: true, allowAbsentProfile: input.ExpectedCID == "", Intent: intent,
		BuildRecord: func(record AuthoritativeRecord) (json.RawMessage, error) {
			if len(record.Record) > 0 && guestProfilePreference(record.Record) == guestvisibility.Unknown {
				return nil, ErrGuestVisibilityUnknown
			}
			return guestvisibility.SetPreference(record.Record, input.HideFromLoggedOut)
		},
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			body, err := json.Marshal(GuestVisibilitySetting{HideFromLoggedOut: guestProfilePreference(record.Record) == guestvisibility.Hidden, ExpectedCID: record.CID})
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200, ResponseBody: body, ResponseHeaders: json.RawMessage(`{"Content-Type":"application/json"}`)}, err
		},
		Rejected: input.Rejected, FinalizeAccepted: finalize, FinalizeObserved: finalize,
		PrepareLocal: func(ctx context.Context, tx pgx.Tx, command PreparedCommand) error {
			commandID = command.ID
			queries := guestvisibilitydb.New(tx)
			state, err := queries.LockGuestVisibilityCommand(ctx, guestvisibilitydb.LockGuestVisibilityCommandParams{ID: pgtype.UUID{Bytes: command.ID, Valid: true}, OwnerDid: input.Owner.String()})
			if err != nil {
				return err
			}
			if state == string(CommandAccepted) || state == string(CommandRejected) {
				return nil
			}
			if err := queries.EnsureGuestVisibilityRow(ctx, guestvisibilitydb.EnsureGuestVisibilityRowParams{Did: input.Owner.String()}); err != nil {
				return err
			}
			_, err = queries.HoldGuestVisibility(ctx, guestvisibilitydb.HoldGuestVisibilityParams{Did: input.Owner.String(), GuestVisibilityHoldKey: pgtype.UUID{Bytes: command.ID, Valid: true}})
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRecordConflict
			}
			return err
		},
	})
	if err == nil && result.State == CommandRejected && commandID != uuid.Nil {
		// A rejected CAS may have observed a newer external opt-out. Retain
		// that authoritative preference before allowing anonymous reads again.
		_, err = service.ReadGuestVisibility(ctx, input.Owner, input.OwnerGeneration, input.SessionID)
	}
	return result, err
}

// Observe absence only when bracketed by the same authoritative repository
// head. This bounded owner read is not used by guest reads or background jobs.
func observeGuestProfile(ctx context.Context, transport *PDSTransport, owner syntax.DID) (AuthoritativeRecord, error) {
	uri := guestProfileURI(owner)
	for attempt := 0; attempt < 3; attempt++ {
		before, err := transport.LatestCommitMetadata(ctx, owner)
		if err != nil {
			return AuthoritativeRecord{}, err
		}
		record, err := transport.GetRecord(ctx, owner, uri)
		if errors.Is(err, auth.ErrRecordNotFound) {
			record = AuthoritativeRecord{URI: uri, Record: json.RawMessage(`{"$type":"app.bsky.actor.profile"}`)}
		} else if err != nil {
			return AuthoritativeRecord{}, err
		}
		after, err := transport.LatestCommitMetadata(ctx, owner)
		if err != nil {
			return AuthoritativeRecord{}, err
		}
		if before == after {
			record.ObservationCID, record.ObservationRevision = after.CID, after.Revision
			return record, nil
		}
	}
	return AuthoritativeRecord{}, ErrRepositoryConflict
}

func guestProfilePreference(raw json.RawMessage) guestvisibility.Preference {
	var recordType struct {
		Type string `json:"$type"`
	}
	if err := json.Unmarshal(raw, &recordType); err != nil || recordType.Type != "app.bsky.actor.profile" {
		return guestvisibility.Unknown
	}
	return guestvisibility.ParsePreference(raw)
}
