package auth

// OAuthLoginDiagnostic identifies the failed boundary without exporting OAuth
// inputs, endpoints, request data or provider prose. The original cause survives.
type OAuthLoginDiagnostic struct {
	stage string
	cause error
}

func (e *OAuthLoginDiagnostic) Error() string { return e.cause.Error() }
func (e *OAuthLoginDiagnostic) Unwrap() error { return e.cause }
func (e *OAuthLoginDiagnostic) Stage() string { return e.stage }
func (e *OAuthLoginDiagnostic) DiagnosticMessage() string {
	// The pinned Indigo directory returns these fixed internal failures without
	// typed causes. Select exact literals only; identity/provider prose is private.
	if e.stage == "identity" {
		switch e.cause.Error() {
		case "identity not found in cache after coalesce returned":
			return "OAuth login failed at identity; reason=cache_result_missing"
		case "unexpected control-flow error":
			return "OAuth login failed at identity; reason=directory_control_flow"
		case "at-identifier neither a Handle nor a DID":
			return "OAuth login failed at identity; reason=identifier_type_invalid"
		}
	}
	return "OAuth login failed at " + e.stage
}

func loginDiagnostic(stage string, cause error) error {
	return &OAuthLoginDiagnostic{stage: stage, cause: cause}
}
