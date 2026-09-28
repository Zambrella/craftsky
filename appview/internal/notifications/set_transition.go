package notifications

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type SetTransition string

const (
	SetTransitionUnchanged   SetTransition = "unchanged"
	SetTransitionActivated   SetTransition = "activated"
	SetTransitionDeactivated SetTransition = "deactivated"
)

func ApplySetTransition(
	ctx context.Context,
	tx pgx.Tx,
	lifecycle Lifecycle,
	transition SetTransition,
	activation Activation,
	retraction Retraction,
) error {
	switch transition {
	case SetTransitionActivated:
		return lifecycle.Activate(ctx, tx, activation)
	case SetTransitionDeactivated:
		return lifecycle.Retract(ctx, tx, retraction)
	case SetTransitionUnchanged:
		return nil
	default:
		return fmt.Errorf("unsupported set transition %q", transition)
	}
}
