package pdscommands

import (
	"context"
	"errors"
	"testing"

	"social.craftsky/appview/internal/ownerlifecycle"
)

type alreadyFencedCommandExecutorFunc func(
	context.Context,
	FencedAppendCommandRequest,
) (CommandResult, error)

func (execute alreadyFencedCommandExecutorFunc) ExecuteAppend(
	ctx context.Context,
	request FencedAppendCommandRequest,
) (CommandResult, error) {
	return execute(ctx, request)
}

func TestScopedAlreadyFencedCommandExecutorRejectsUseAfterCallback(t *testing.T) {
	calls := 0
	scoped, err := NewScopedAlreadyFencedCommandExecutor(
		alreadyFencedCommandExecutorFunc(func(
			context.Context,
			FencedAppendCommandRequest,
		) (CommandResult, error) {
			calls++
			return CommandResult{}, nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scoped.ExecuteAppend(context.Background(), FencedAppendCommandRequest{}); err != nil {
		t.Fatal(err)
	}
	scoped.CloseAndWait()
	if _, err := scoped.ExecuteAppend(context.Background(), FencedAppendCommandRequest{}); !errors.Is(err, ownerlifecycle.ErrFenceRequired) {
		t.Fatalf("post-callback execution error=%v, want %v", err, ownerlifecycle.ErrFenceRequired)
	}
	if calls != 1 {
		t.Fatalf("delegate calls=%d, want one", calls)
	}
}
