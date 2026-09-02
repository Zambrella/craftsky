package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// renderErrorHTML shows a minimal HTML error page. Used by the OAuth
// callback since it's loaded in a browser, not by a programmatic client.
func renderErrorHTML(w http.ResponseWriter, status int, userMessage string) {
	setCallbackSecurityHeaders(w, "", "")
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
	Failure     RegistrationFailureCode
	DeepLinkURL string
	LoopbackURI string
	Nonce       string
}

func renderCallbackHTML(w http.ResponseWriter, data callbackPageData) error {
	if data.Failure != "" && (!data.Failure.valid() || data.Code != "") {
		return ErrOAuthFlowInvalid
	}
	if data.DeepLinkURL != "" && data.LoopbackURI != "" {
		return fmt.Errorf("callback cannot use verified-link and loopback handoffs together")
	}
	connectSource := ""
	if data.LoopbackURI != "" {
		var err error
		connectSource, err = exactLoopbackOrigin(data.LoopbackURI)
		if err != nil {
			return err
		}
	}
	nonceBytes := make([]byte, 18)
	if _, err := rand.Read(nonceBytes); err != nil {
		return fmt.Errorf("generate callback CSP nonce: %w", err)
	}
	data.Nonce = base64.RawURLEncoding.EncodeToString(nonceBytes)
	setCallbackSecurityHeaders(w, data.Nonce, connectSource)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return callbackTmpl.Execute(w, data)
}

func trustedRegistrationFailurePageData(
	metadata AuthRequestMetadata,
	code RegistrationFailureCode,
	verifiedURL string,
	allowDev bool,
	_ url.Values,
	now time.Time,
) (callbackPageData, error) {
	if !code.valid() || !metadata.valid() || metadata.Purpose != RegistrationOAuthPurpose ||
		metadata.RequestState != AuthRequestReady || metadata.ExpiresAt.IsZero() || !now.Before(metadata.ExpiresAt) {
		return callbackPageData{}, ErrOAuthFlowInvalid
	}
	data := callbackPageData{Failure: code}
	var err error
	switch metadata.HandoffMode {
	case HandoffVerifiedLink:
		data.DeepLinkURL, err = verifiedCompletionURL(verifiedURL, url.Values{"error": {string(code)}})
	case HandoffLoopback:
		if !loopbackRedirectPattern.MatchString(metadata.LoopbackURI) {
			return callbackPageData{}, ErrOAuthFlowInvalid
		}
		data.LoopbackURI = metadata.LoopbackURI
	case HandoffDevScheme:
		if !allowDev {
			return callbackPageData{}, ErrOAuthFlowInvalid
		}
		data.DeepLinkURL, err = devSchemeCompletionURL("/auth/complete", url.Values{"error": {string(code)}})
	default:
		err = ErrOAuthFlowInvalid
	}
	if err != nil {
		return callbackPageData{}, err
	}
	return data, nil
}

func setCallbackSecurityHeaders(w http.ResponseWriter, nonce, connectSource string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	scriptSource := "'none'"
	if nonce != "" {
		scriptSource = "'nonce-" + nonce + "'"
	}
	if connectSource == "" {
		connectSource = "'none'"
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src "+scriptSource+"; connect-src "+connectSource+"; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
}

func exactLoopbackOrigin(raw string) (string, error) {
	if !loopbackRedirectPattern.MatchString(raw) {
		return "", fmt.Errorf("invalid loopback callback URI")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" {
		return "", fmt.Errorf("invalid loopback callback URI")
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("invalid loopback callback port")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
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
<p>{{if .Failure}}Registration did not complete.{{else}}Signed in.{{end}} {{if .DeepLinkURL}}Return to the Craftsky app.{{else}}You can close this tab.{{end}}</p>
<script nonce="{{.Nonce}}">
{{if .DeepLinkURL}}
window.location.replace({{.DeepLinkURL}});
{{else if .LoopbackURI}}
fetch({{.LoopbackURI}}, {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({{if .Failure}}{error: {{.Failure}}}{{else}}{code: {{.Code}}}{{end}})
}).finally(function(){ document.body.insertAdjacentHTML("beforeend", "<p>Done.</p>"); });
{{end}}
</script>
</body></html>`))

// loopbackRedirectPattern matches the only URI shape our CLI uses:
// http://127.0.0.1:<port>/<path>. Reject anything else at ingress
// (e.g. https://evil.example/, javascript:..., mailto:...).
var loopbackRedirectPattern = regexp.MustCompile(`^http://127\.0\.0\.1:\d{1,5}(/[A-Za-z0-9._~\-/]*)?$`)
