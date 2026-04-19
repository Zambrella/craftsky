package auth

import "io"

// RenderCallbackForTest is a test-only seam exposed via the standard
// internal_test package pattern (this file is excluded from normal
// builds because it ends in _test.go). Lets handlers_test.go drive the
// callback template directly without going through the full OAuth
// dance, so we can write XSS regression tests against the JS-string
// escaping in callbackTmpl without standing up a fake Authorization
// Server.
func RenderCallbackForTest(w io.Writer, token, loopbackURI string) error {
	return callbackTmpl.Execute(w, callbackPageData{Token: token, LoopbackURI: loopbackURI})
}
