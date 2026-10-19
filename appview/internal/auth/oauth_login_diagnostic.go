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
	return "OAuth login failed at " + e.stage
}

func loginDiagnostic(stage string, cause error) error {
	return &OAuthLoginDiagnostic{stage: stage, cause: cause}
}
