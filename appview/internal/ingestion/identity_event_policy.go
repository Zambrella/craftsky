package ingestion

import (
	"fmt"

	"social.craftsky/appview/internal/tap"
)

type identityEventPolicy struct {
	enqueueRefresh bool
}

func classifyIdentityEvent(event tap.IdentityEvent) (identityEventPolicy, error) {
	switch event.Status {
	case "active", "deactivated", "suspended", "takendown", "deleted":
		return identityEventPolicy{enqueueRefresh: true}, nil
	default:
		return identityEventPolicy{}, fmt.Errorf("unsupported Tap identity status %q", event.Status)
	}
}
