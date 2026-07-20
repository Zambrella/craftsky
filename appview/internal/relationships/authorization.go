package relationships

type Operation uint8

const (
	OperationFollowCreate Operation = iota
	OperationLikeCreate
	OperationRepostCreate
	OperationReplyCreate
	OperationQuoteCreate
	OperationMentionCreate
	OperationFollowDelete
	OperationLikeDelete
	OperationRepostDelete
	OperationContentDelete
	OperationReport
	OperationBlockCreate
	OperationBlockDelete
	OperationMuteCreate
	OperationMuteDelete
)

type Denial uint8

const (
	DenialNone Denial = iota
	DenialInteractionBlocked
	DenialNotOwner
)

type Authorization struct {
	Allowed bool
	Denial  Denial
}

func Authorize(operation Operation, state State, ownsResource bool) Authorization {
	switch operation {
	case OperationFollowCreate, OperationLikeCreate, OperationRepostCreate,
		OperationReplyCreate, OperationQuoteCreate, OperationMentionCreate:
		if state.HasBlock() {
			return Authorization{Denial: DenialInteractionBlocked}
		}
	case OperationFollowDelete, OperationLikeDelete, OperationRepostDelete, OperationContentDelete:
		if !ownsResource {
			return Authorization{Denial: DenialNotOwner}
		}
	case OperationBlockDelete:
		if !ownsResource || !state.Blocking {
			return Authorization{Denial: DenialNotOwner}
		}
	}
	return Authorization{Allowed: true}
}
