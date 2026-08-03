package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/scheduledposts"
)

// ScheduledPublicationPolicy is the current membership and directed-
// interaction policy required immediately before a delayed public write.
type ScheduledPublicationPolicy interface {
	RequireCurrentMember(context.Context, syntax.DID) error
	DirectedInteractionAuthorizer
}

// ValidateScheduledPublication reapplies the current post, membership, mention,
// and block rules without changing the member-authored scheduled payload.
func ValidateScheduledPublication(
	ctx context.Context,
	policy ScheduledPublicationPolicy,
	owner syntax.DID,
	payload scheduledposts.Payload,
	limits MediaLimits,
) error {
	if policy == nil {
		return fmt.Errorf("scheduled publication policy is required")
	}
	if err := policy.RequireCurrentMember(ctx, owner); err != nil {
		return err
	}
	request := PostCreateRequest{
		Text:   payload.Text,
		Facets: payload.Facets,
		Langs:  payload.Langs,
	}
	if len(payload.Project) > 0 {
		if err := json.Unmarshal(payload.Project, &request.Project); err != nil {
			return err
		}
	}
	for _, media := range payload.Media {
		image := PostImage{
			Image: map[string]any{
				"ref":      map[string]any{"$link": "bafk-scheduled-placeholder"},
				"mimeType": "image/jpeg",
				"size":     1,
			},
			Alt: media.Alt,
		}
		if media.Width != 0 || media.Height != 0 {
			image.AspectRatio = &PostImageAspectRatio{
				Width:  media.Width,
				Height: media.Height,
			}
		}
		request.Images = append(request.Images, image)
	}
	if err := ValidatePostCreateWithLimits(request, limits); err != nil {
		return err
	}
	mentioned, err := mentionedDIDs(request)
	if err != nil {
		return err
	}
	for _, subject := range mentioned {
		if err := policy.AuthorizeDirectedInteraction(
			ctx,
			owner,
			subject,
			relationships.OperationMentionCreate,
		); err != nil {
			return err
		}
	}
	return nil
}
