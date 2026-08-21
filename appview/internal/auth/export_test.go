package auth

import (
	"io"
	"net/http"
)

// RenderCallbackForTest is a test-only seam exposed via the standard
// internal_test package pattern (this file is excluded from normal
// builds because it ends in _test.go). Lets handlers_test.go drive the
// callback template directly without going through the full OAuth
// dance, so we can write XSS regression tests against the JS-string
// escaping in callbackTmpl without standing up a fake Authorization
// Server.
func RenderCallbackForTest(w io.Writer, code, loopbackURI string) error {
	return callbackTmpl.Execute(w, callbackPageData{Code: code, LoopbackURI: loopbackURI, Nonce: "test-nonce"})
}

// RenderSecureCallbackForTest exercises the production callback renderer,
// including its generated nonce and exact-origin CSP.
func RenderSecureCallbackForTest(w http.ResponseWriter, code, loopbackURI string) error {
	return renderCallbackHTML(w, callbackPageData{Code: code, LoopbackURI: loopbackURI})
}
