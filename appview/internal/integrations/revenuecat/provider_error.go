package revenuecat

// ProviderError retains native causes in memory while its ordinary text never
// includes a request URL, customer, credential or provider response payload.
// StatusCode is selected independently from any AppView response status.
type ProviderError struct {
	stage      string
	statusCode int
	cause      error
}

func (e *ProviderError) Error() string   { return e.DiagnosticMessage() }
func (e *ProviderError) Unwrap() error   { return e.cause }
func (e *ProviderError) StatusCode() int { return e.statusCode }
func (e *ProviderError) DiagnosticMessage() string {
	switch e.stage {
	case "create":
		return "create RevenueCat request failed"
	case "request":
		return "RevenueCat request failed"
	case "response":
		return "RevenueCat response rejected"
	case "read":
		return "read RevenueCat response failed"
	case "decode":
		return "decode RevenueCat response failed"
	case "oversized":
		return "RevenueCat response exceeds configured limit"
	default:
		return "RevenueCat provider operation failed"
	}
}
