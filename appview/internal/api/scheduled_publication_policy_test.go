package api

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledPolicyFake struct {
	currentErr error
	blockedDID syntax.DID
}

func (fake scheduledPolicyFake) RequireCurrentMember(context.Context, syntax.DID) error {
	return fake.currentErr
}

func (fake scheduledPolicyFake) AuthorizeDirectedInteraction(
	_ context.Context,
	_, subject syntax.DID,
	_ relationships.Operation,
) error {
	if subject == fake.blockedDID {
		return ErrInteractionBlocked
	}
	return nil
}

func TestValidateScheduledPublicationRechecksCurrentPolicy(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:owner")
	mentioned := syntax.DID("did:plc:mentioned")
	payload := scheduledposts.Payload{
		Kind: scheduledposts.PostKindStandard,
		Text: "hello",
		Facets: []byte(`[{
			"index":{"byteStart":0,"byteEnd":5},
			"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:mentioned"}]
		}]`),
	}

	if err := ValidateScheduledPublication(
		context.Background(),
		scheduledPolicyFake{},
		owner,
		payload,
		DefaultMediaLimits(),
	); err != nil {
		t.Fatalf("valid publication policy: %v", err)
	}

	if err := ValidateScheduledPublication(
		context.Background(),
		scheduledPolicyFake{blockedDID: mentioned},
		owner,
		payload,
		DefaultMediaLimits(),
	); !errors.Is(err, ErrInteractionBlocked) {
		t.Fatalf("blocked mention error = %v, want ErrInteractionBlocked", err)
	}

	if err := ValidateScheduledPublication(
		context.Background(),
		scheduledPolicyFake{currentErr: relationships.ErrProfileNotFound},
		owner,
		payload,
		DefaultMediaLimits(),
	); !errors.Is(err, relationships.ErrProfileNotFound) {
		t.Fatalf("former member error = %v, want ErrProfileNotFound", err)
	}

	invalid := payload
	invalid.Text = ""
	if err := ValidateScheduledPublication(
		context.Background(),
		scheduledPolicyFake{},
		owner,
		invalid,
		DefaultMediaLimits(),
	); err == nil {
		t.Fatal("invalid current post policy was accepted")
	}
}
