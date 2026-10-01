package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

const onboardingProfileKind = "oauth_onboarding_profile"

var onboardingProfileBody = json.RawMessage(`{"$type":"social.craftsky.actor.profile","crafts":[]}`)

var ErrOnboardingProfileUnresolved = errors.New("onboarding profile write outcome is unresolved")
var ErrOnboardingProfileChanged = errors.New("onboarding profile changed before creation")

// OnboardingProfileService is the one pre-activation command writer. It uses the
// callback's existing exclusive owner fence and pending PDS capability, never
// the ordinary active-session command boundary.
type OnboardingProfileService struct {
	store      *Store
	lifecycles *ownerlifecycle.Store
	now        func() time.Time
}

func NewOnboardingProfileService(store *Store, lifecycles *ownerlifecycle.Store) (*OnboardingProfileService, error) {
	if store == nil || lifecycles == nil || store.lifecycles != lifecycles {
		return nil, errors.New("onboarding profile command requires shared command and owner stores")
	}
	return &OnboardingProfileService{store: store, lifecycles: lifecycles, now: time.Now}, nil
}

func (service *OnboardingProfileService) PutProfile(
	ctx context.Context, client auth.PDSClient, owner syntax.DID, generation int64, record any,
) (syntax.CID, error) {
	if service == nil || client == nil || owner == "" || generation < 1 {
		return "", ErrMalformedCommand
	}
	protocol, ok := client.(PDSProtocol)
	if !ok {
		return "", ErrDispatchUnavailable
	}
	body, err := json.Marshal(record)
	if err != nil {
		return "", errors.Join(ErrMalformedCommand, err)
	}
	matches, err := equalCanonicalJSON(body, onboardingProfileBody)
	if err != nil || !matches {
		return "", ErrMalformedCommand
	}
	var cid syntax.CID
	err = service.lifecycles.WithOnboardingEffect(ctx, owner, generation, func(effectCtx context.Context) error {
		var writeErr error
		cid, writeErr = service.putFenced(effectCtx, protocol, owner, generation)
		return writeErr
	})
	return cid, err
}

func (service *OnboardingProfileService) putFenced(
	ctx context.Context, client PDSProtocol, owner syntax.DID, generation int64,
) (syntax.CID, error) {
	uri := syntax.ATURI("at://" + owner.String() + "/social.craftsky.actor.profile/self")
	key := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s:%s:%d", onboardingProfileKind, owner, generation)))
	intent := json.RawMessage(`{"collection":"social.craftsky.actor.profile","rkey":"self"}`)
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1, Owner: owner, OperationKind: onboardingProfileKind,
		Intent: intent, Conditions: json.RawMessage(fmt.Sprintf(`{"ownerGeneration":%d}`, generation)),
	})
	if err != nil {
		return "", err
	}
	command, err := service.store.PrepareOrReplay(ctx, PrepareRequest{
		Owner: owner, OwnerGeneration: generation, OperationKind: onboardingProfileKind,
		OperationKey: key, FingerprintVersion: 1, Fingerprint: fingerprint,
		ImmutableRequest: intent, SelectedURI: uri, SelectedRkey: "self",
		ExpectedOwner: owner, ExpectedOwnerGeneration: generation,
	})
	if err != nil {
		return "", err
	}
	if command.State == CommandRejected {
		return "", ErrOnboardingProfileChanged
	}
	firstAttempt := 1
	if command.State != CommandAccepted {
		command, firstAttempt, err = service.store.ResumeKnownInvalidSwap(ctx, command)
		if err != nil {
			return "", err
		}
	}
	transport, err := NewPDSTransport(client)
	if err != nil {
		return "", err
	}
	// Always read the live record, including on replay: an accepted command is
	// not permission to project a profile subsequently removed by its owner.
	for attemptNumber := firstAttempt; attemptNumber <= 3; attemptNumber++ {
		record, readErr := transport.GetRecord(ctx, owner, uri)
		if readErr == nil {
			matching, err := equalCanonicalJSON(record.Record, onboardingProfileBody)
			if err != nil {
				return "", err
			}
			if !matching {
				if command.State != CommandPrepared {
					_, _ = service.store.UnresolvedResult(ctx, command)
					return "", ErrOnboardingProfileUnresolved
				}
				return "", ErrOnboardingProfileChanged
			}
			if command.State == CommandDispatching {
				if _, err := service.store.UnresolvedResult(ctx, command); err != nil {
					return "", err
				}
				command.State = CommandAmbiguous
			}
			if command.State == CommandPrepared || command.State == CommandAmbiguous {
				if err := service.store.CompleteCommand(ctx, command.ID, TerminalResult{
					State: CommandAccepted, HTTPStatus: http.StatusOK,
				}); err != nil {
					return "", err
				}
			}
			return record.CID, nil
		}
		if !errors.Is(readErr, auth.ErrRecordNotFound) {
			return "", errors.Join(ErrDispatchUnavailable, readErr)
		}
		if command.State == CommandAccepted {
			return "", ErrOnboardingProfileChanged
		}
		if command.State == CommandAmbiguous || command.State == CommandDispatching {
			_, _ = service.store.UnresolvedResult(ctx, command)
			return "", ErrOnboardingProfileUnresolved
		}
		head, err := transport.LatestCommit(ctx, owner)
		if err != nil {
			return "", errors.Join(ErrDispatchUnavailable, err)
		}
		step := DispatchStep{Action: "create", URI: uri, Body: onboardingProfileBody}
		dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
			AlgorithmVersion: 1, RequestFingerprint: fingerprint, RepositoryHead: head, Steps: []DispatchStep{step},
		})
		if err != nil {
			return "", err
		}
		attempt, err := service.store.BeginDispatch(ctx, command.ID, ExactPlan{
			Version: command.ActivePlanVersion + 1, FingerprintVersion: 1,
			Fingerprint: dispatchFingerprint, RepositoryCID: head,
			RemoteDeadline: service.now().UTC().Add(30 * time.Second), Steps: []DispatchStep{step},
		})
		if err != nil {
			return "", err
		}
		err = transport.ApplyWrites(ctx, owner, head, []DispatchStep{step})
		if errors.Is(err, ErrRepositorySwapConflict) {
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{
				Outcome: DispatchInvalidSwap, ErrorClass: "invalid_swap",
			}); completeErr != nil {
				return "", completeErr
			}
			command.ActivePlanVersion++
			continue
		}
		if err != nil {
			if errors.Is(err, ErrDispatchRejected) || errors.Is(err, ErrAtomicWritesUnsupported) {
				if completeErr := service.store.CompleteDispatchAndCommand(ctx, attempt.ID,
					DispatchCompletion{Outcome: CommandRejected, ErrorClass: "rejected"},
					TerminalResult{State: CommandRejected, HTTPStatus: http.StatusBadGateway},
				); completeErr != nil {
					return "", completeErr
				}
				return "", errors.Join(ErrDispatchUnavailable, err)
			}
			if completeErr := service.store.CompleteDispatch(ctx, attempt.ID, DispatchCompletion{
				Outcome: CommandAmbiguous, RetryAfterSeconds: 1, ErrorClass: "transport",
			}); completeErr != nil {
				return "", completeErr
			}
			return "", ErrOnboardingProfileUnresolved
		}
		// Resolve the created CID before publishing membership. A lost read
		// leaves a durable dispatching command for the next login to reconcile.
		created, err := transport.GetRecord(ctx, owner, uri)
		if err != nil {
			return "", errors.Join(ErrOnboardingProfileUnresolved, err)
		}
		matching, err := equalCanonicalJSON(created.Record, onboardingProfileBody)
		if err != nil || !matching {
			return "", ErrOnboardingProfileUnresolved
		}
		if err := service.store.CompleteDispatchAndCommand(ctx, attempt.ID,
			DispatchCompletion{Outcome: CommandAccepted},
			TerminalResult{State: CommandAccepted, HTTPStatus: http.StatusOK},
		); err != nil {
			return "", err
		}
		return created.CID, nil
	}
	return "", ErrOnboardingProfileUnresolved
}
