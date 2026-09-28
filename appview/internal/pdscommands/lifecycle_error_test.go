package pdscommands

import (
	"errors"
	"fmt"
	"testing"

	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestOnlyDefinitiveLifecycleFailuresRejectPreparedSetCommands(t *testing.T) {
	for _, cause := range []error{
		ownerlifecycle.ErrTerminalOwner,
		ownerlifecycle.ErrOwnerNotActive,
		ownerlifecycle.ErrGenerationChanged,
	} {
		if !isDefinitiveLifecycleError(fmt.Errorf("check lifecycle: %w", cause)) {
			t.Fatalf("%v should terminate the command", cause)
		}
	}
	if isDefinitiveLifecycleError(errors.New("database temporarily unavailable")) {
		t.Fatal("transient lifecycle lookup must leave the prepared command retryable")
	}
}
