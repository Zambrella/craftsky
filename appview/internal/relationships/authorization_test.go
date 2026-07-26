package relationships

import "testing"

func TestAuthorizeDirectedWritesCleanupReportsAndRelationshipOwnership(t *testing.T) {
	for _, operation := range []Operation{
		OperationFollowCreate, OperationLikeCreate, OperationRepostCreate,
		OperationReplyCreate, OperationQuoteCreate, OperationMentionCreate,
	} {
		for _, state := range []State{{Blocking: true}, {BlockedBy: true}, {Blocking: true, BlockedBy: true}} {
			decision := Authorize(operation, state, false)
			if decision.Allowed || decision.Denial != DenialInteractionBlocked {
				t.Fatalf("Authorize(%v, %+v) = %+v, want interaction blocked", operation, state, decision)
			}
		}
		if decision := Authorize(operation, State{Muted: true}, false); !decision.Allowed {
			t.Fatalf("mute denied operation %v: %+v", operation, decision)
		}
	}

	for _, operation := range []Operation{
		OperationFollowDelete, OperationLikeDelete, OperationRepostDelete, OperationContentDelete,
	} {
		if decision := Authorize(operation, State{BlockedBy: true}, true); !decision.Allowed {
			t.Fatalf("owned cleanup %v denied: %+v", operation, decision)
		}
		if decision := Authorize(operation, State{}, false); decision.Allowed || decision.Denial != DenialNotOwner {
			t.Fatalf("foreign cleanup %v = %+v, want not owner", operation, decision)
		}
	}

	for _, operation := range []Operation{OperationReport, OperationBlockCreate} {
		if decision := Authorize(operation, State{BlockedBy: true}, false); !decision.Allowed {
			t.Fatalf("safety operation %v denied: %+v", operation, decision)
		}
	}
	if decision := Authorize(OperationBlockDelete, State{BlockedBy: true}, false); decision.Allowed || decision.Denial != DenialNotOwner {
		t.Fatalf("foreign unblock = %+v, want not owner", decision)
	}
	if decision := Authorize(OperationBlockDelete, State{Blocking: true, BlockedBy: true}, true); !decision.Allowed {
		t.Fatalf("owned unblock denied: %+v", decision)
	}
}
