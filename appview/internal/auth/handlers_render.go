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
	Code            string
	Failure         RegistrationFailureCode
	DeepLinkURL     string
	VerifiedLinkURL string
	LoopbackURI     string
	Nonce           string
}

func renderCallbackHTML(w http.ResponseWriter, data callbackPageData) error {
	if data.Failure != "" && (!data.Failure.valid() || data.Code != "") {
		return ErrOAuthFlowInvalid
	}
	if data.DeepLinkURL != "" && data.LoopbackURI != "" {
		return fmt.Errorf("callback cannot use verified-link and loopback handoffs together")
	}
	if data.DeepLinkURL != "" {
		parsed, err := url.Parse(data.DeepLinkURL)
		if err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil {
			data.VerifiedLinkURL = data.DeepLinkURL
		}
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
	styleSource := "'none'"
	if nonce != "" {
		scriptSource = "'nonce-" + nonce + "'"
		styleSource = scriptSource
	}
	if connectSource == "" {
		connectSource = "'none'"
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src "+scriptSource+"; style-src "+styleSource+"; connect-src "+connectSource+"; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
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
<html lang="en"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="referrer" content="no-referrer">
<meta name="theme-color" content="#f5efe4">
<title>CraftSky — {{if .Failure}}return to the app{{else}}signed in{{end}}</title>
<style nonce="{{.Nonce}}">
:root {
    color-scheme: light;
    font-family: ui-rounded, "SF Pro Rounded", "Avenir Next", system-ui, sans-serif;
    --paper: #f5efe4;
    --paper-raised: #ffffff;
    --ink: #161210;
    --ink-muted: #3e3733;
    --cobalt: #1535d6;
    --cobalt-deep: #0c1f8c;
    --butter: #f7d46a;
}
* { box-sizing: border-box; }
body {
    display: grid;
    min-height: 100vh;
    min-height: 100svh;
    margin: 0;
    padding: 24px;
    place-items: center;
    color: var(--ink);
    background: var(--paper);
}
main {
    width: min(100%, 460px);
}
.card {
    padding: clamp(28px, 7vw, 42px);
    border: 1px solid #ded5c8;
    border-radius: 18px;
    background: var(--paper-raised);
    box-shadow: 0 14px 36px rgb(22 18 16 / 8%);
}
.brand {
    display: flex;
    align-items: center;
    gap: 9px;
    margin-bottom: 34px;
    font-size: 14px;
    font-weight: 800;
    letter-spacing: 0.04em;
}
.brand-mark {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--cobalt);
    font-size: 0;
}
.eyebrow {
    margin: 0 0 8px;
    color: var(--cobalt);
    font-size: 12px;
    font-weight: 800;
    letter-spacing: 0.09em;
    text-transform: uppercase;
}
h1 {
    margin: 0;
    font-size: clamp(32px, 8vw, 40px);
    line-height: 1.08;
    letter-spacing: -0.035em;
}
.message {
    margin: 18px 0 28px;
    color: var(--ink-muted);
    font-size: 17px;
    line-height: 1.5;
}
.cta {
    display: flex;
    width: 100%;
    min-height: 54px;
    align-items: center;
    justify-content: center;
    padding: 14px 24px;
    border: 0;
    border-radius: 12px;
    color: white;
    background: var(--cobalt);
    box-shadow: 0 6px 16px rgb(21 53 214 / 20%);
    font-size: 16px;
    font-weight: 800;
    text-align: center;
    text-decoration: none;
    transition: background 120ms ease, transform 120ms ease;
}
.cta:hover { background: var(--cobalt-deep); }
.cta:active { transform: translateY(1px); }
.cta:focus-visible { outline: 3px solid var(--butter); outline-offset: 3px; }
.note {
    margin: 18px 0 0;
    color: var(--ink-muted);
    font-size: 14px;
    line-height: 1.45;
    text-align: center;
}
@media (max-width: 420px) {
    body { padding: 16px; }
    .card { border-radius: 16px; }
}
@media (prefers-reduced-motion: reduce) {
    .cta { transition: none; }
}
</style>
</head><body>
<main>
<section class="card" aria-labelledby="page-title">
<div class="brand"><span class="brand-mark" aria-hidden="true">CS</span><span>CraftSky</span></div>
<p class="eyebrow">{{if .Failure}}One more step{{else}}All stitched up{{end}}</p>
<h1 id="page-title">{{if .Failure}}Return to CraftSky.{{else}}You're signed in.{{end}}</h1>
<p class="message">{{if .Failure}}Registration did not finish. Open CraftSky to continue.{{else if .VerifiedLinkURL}}Your session is ready. Open the app to continue.{{else if .DeepLinkURL}}Returning you to the app.{{else}}Authentication is complete. You can close this tab.{{end}}</p>
{{if .VerifiedLinkURL}}<a class="cta" href="{{.VerifiedLinkURL}}">Open CraftSky</a>{{end}}
{{if .VerifiedLinkURL}}<p class="note">You can close this tab after CraftSky opens.</p>{{end}}
</section>
</main>
<script nonce="{{.Nonce}}">
{{if .VerifiedLinkURL}}
{{else if .DeepLinkURL}}
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
