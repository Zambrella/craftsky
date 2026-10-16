# ADR 018: Atmosphere operator login and browser sessions

Date: 2026-10-15

Status: Accepted for implementation under the approved moderation-console plan

## Decision

The standalone moderation console authenticates existing AT Protocol accounts
with an isolated Indigo OAuth client requesting only `atproto`. AppView checks
the verified account DID against `safety_operators` before issuing a local
operator session. The selected admin/mod/helper set and active assignments are
loaded on every admin request. Consumer membership and OAuth handoff flows do
not participate. Legacy human integration bearers do not authenticate the
console or its production admin routes.

Serve the React assets under `/admin/` on AppView's origin. Admin APIs use an
opaque cookie-backed session with a database verifier, eight-hour absolute and
30-minute idle expiry, logout and maintainer revocation. Cookies are host-only,
HttpOnly, SameSite=Lax and Secure in HTTPS deployments. The explicit development
exception permits HTTP only on exact `127.0.0.1` loopback origins.

Login initiation and unsafe admin requests require the exact configured Origin
and a browser-bound CSRF header. OAuth state is separately persisted with the
browser-secret digest and consumed once before code exchange. Recheck DID/PDS/
issuer authority and active registry membership at completion. No credentials
are returned in URLs or JSON. The identity-only OAuth tokens are discarded after
verification; no refresh or PDS resource operation is needed or performed.
Pending attempts expire after ten minutes and are reclaimed on admission, with
a database-wide capacity limit. Consumer lifecycle storage is untouched.

Operator discovery/callback paths are `/oauth/admin/client-metadata.json`,
`/oauth/admin/jwks.json` and `/oauth/admin/callback`. The confidential client has
a distinct client ID and callback while initially using the existing server
signing key configuration; key rotation must update both client documents.
Protocol HTTP calls use the existing hardened federated clients and fresh
identity directory. URL validation enumerates exact paths per client purpose.

Consumer membership is not required, but an existing globally terminal DID
tombstone denies login/session issuance and every later operator request.
Retain the operator DID/audit linkage as private security history; terminal
purge deletes pending operator attempts using a DID-leading keyset index.

## Consequences

The admin cookie is a scoped addition to the API bearer/device-ID convention.
Ordinary consumer routes retain that contract and cannot use operator sessions.
Registry administration initially uses privileged maintainer CLI/database access
and records a maintenance actor rather than claiming OAuth attribution. Browser
operator administration is a later console stage.

This adds local session handling and frontend build maintenance, while removing
legacy external-UI resource authentication and its per-user binding layer. Historic actor
references and append-only audits remain intact. Production provisioning,
route enablement and deployment require the existing operational controls.
