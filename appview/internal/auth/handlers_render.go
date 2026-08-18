package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
)

// renderErrorHTML shows a minimal HTML error page. Used by the OAuth
// callback since it's loaded in a browser, not by a programmatic client.
func renderErrorHTML(w http.ResponseWriter, status int, userMessage string) {
	setCallbackSecurityHeaders(w, "")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = errorPageTmpl.Execute(w, errorPageData{Message: userMessage})
}

type errorPageData struct{ Message string }

var errorPageTmpl = template.Must(template.New("err").Parse(`<!doctype html>
<html><head><title>Craftsky — error</title></head><body>
<h1>Sign-in failed</h1>
<p>{{.Message}}</p>
</body></html>`))

// callbackPageData drives the post-OAuth callback HTML. Filled by
// CallbackHandler before rendering. Either DeepLinkURL OR LoopbackURI
// is set, never both.
type callbackPageData struct {
	Code        string
	DeepLinkURL string
	LoopbackURI string
	Nonce       string
}

func renderCallbackHTML(w http.ResponseWriter, data callbackPageData) error {
	nonceBytes := make([]byte, 18)
	if _, err := rand.Read(nonceBytes); err != nil {
		return fmt.Errorf("generate callback CSP nonce: %w", err)
	}
	data.Nonce = base64.RawURLEncoding.EncodeToString(nonceBytes)
	setCallbackSecurityHeaders(w, data.Nonce)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return callbackTmpl.Execute(w, data)
}

func setCallbackSecurityHeaders(w http.ResponseWriter, nonce string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	scriptSource := "'none'"
	if nonce != "" {
		scriptSource = "'nonce-" + nonce + "'"
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src "+scriptSource+"; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
}

// callbackTmpl renders the post-OAuth landing page. Uses html/template's
// contextual escaping so LoopbackURI inside the <script> body gets
// JavaScript-string-context escaping automatically. The template's
// double layer of safety (this + the regex check at ingress) is
// intentional: belt-and-braces against a malicious loopback_redirect_uri.
//
// SECURITY: do NOT swap "html/template" for "text/template" — the
// contextual escaping is load-bearing. Without it, a malicious
// loopback_redirect_uri could break out of the JS string literal even
// when ingress validation lets it through. The TestCallbackTemplate_*
// tests in handlers_test.go are regression tests against this swap.
var callbackTmpl = template.Must(template.New("cb").Parse(`<!doctype html>
<html><head><title>Craftsky — signed in</title></head><body>
<p>Signed in. {{if .DeepLinkURL}}Return to the Craftsky app.{{else}}You can close this tab.{{end}}</p>
<script nonce="{{.Nonce}}">
{{if .DeepLinkURL}}
window.location.replace({{.DeepLinkURL}});
{{else if .LoopbackURI}}
fetch({{.LoopbackURI}}, {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({code: {{.Code}}})
}).finally(function(){ document.body.insertAdjacentHTML("beforeend", "<p>Done.</p>"); });
{{end}}
</script>
</body></html>`))

// loopbackRedirectPattern matches the only URI shape our CLI uses:
// http://127.0.0.1:<port>/<path>. Reject anything else at ingress
// (e.g. https://evil.example/, javascript:..., mailto:...).
var loopbackRedirectPattern = regexp.MustCompile(`^http://127\.0\.0\.1:\d{1,5}(/[A-Za-z0-9._~\-/]*)?$`)
