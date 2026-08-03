package scheduledposts

type DiagnosticOperation string

const (
	DiagnosticOperationCreate  DiagnosticOperation = "create"
	DiagnosticOperationEdit    DiagnosticOperation = "edit"
	DiagnosticOperationDelete  DiagnosticOperation = "delete"
	DiagnosticOperationClaim   DiagnosticOperation = "claim"
	DiagnosticOperationPublish DiagnosticOperation = "publish"
	DiagnosticOperationRetry   DiagnosticOperation = "retry"
	DiagnosticOperationCleanup DiagnosticOperation = "cleanup"
	DiagnosticOperationRecover DiagnosticOperation = "recover"
)

type DiagnosticResult string

const (
	DiagnosticResultSuccess DiagnosticResult = "success"
	DiagnosticResultFailure DiagnosticResult = "failure"
	DiagnosticResultStale   DiagnosticResult = "stale"
)

var safeDiagnosticErrorClasses = map[string]struct{}{
	"none":                   {},
	"auth_unavailable":       {},
	"object_unavailable":     {},
	"pds_unavailable":        {},
	"dependency_unavailable": {},
	"policy_invalid":         {},
	"media_invalid":          {},
	"record_conflict":        {},
	"lease_expired":          {},
	"stale_worker":           {},
}

func SafeDiagnosticFields(
	operation DiagnosticOperation,
	result DiagnosticResult,
	errorClass string,
) map[string]string {
	if !operation.safe() {
		operation = "unknown"
	}
	if !result.safe() {
		result = "unknown"
	}
	if _, ok := safeDiagnosticErrorClasses[errorClass]; !ok {
		errorClass = "unknown"
	}
	return map[string]string{
		"component":  "scheduled_posts",
		"operation":  string(operation),
		"result":     string(result),
		"errorClass": errorClass,
	}
}

func (operation DiagnosticOperation) safe() bool {
	switch operation {
	case DiagnosticOperationCreate,
		DiagnosticOperationEdit,
		DiagnosticOperationDelete,
		DiagnosticOperationClaim,
		DiagnosticOperationPublish,
		DiagnosticOperationRetry,
		DiagnosticOperationCleanup,
		DiagnosticOperationRecover:
		return true
	default:
		return false
	}
}

func (result DiagnosticResult) safe() bool {
	return result == DiagnosticResultSuccess ||
		result == DiagnosticResultFailure ||
		result == DiagnosticResultStale
}
