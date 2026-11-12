# Account provider settings

Mobile and web account settings show the handle, DID and hosting provider, with
copy actions for account details and links to provider account management.
The copy explains that provider changes affect the shared AT Protocol account.
The separate CraftSky section labels deletion as “Delete CraftSky account” and
explains that this does not delete the wider account.

`GET /v1/whoami` adds optional `pdsUrl`, `hostingProvider`,
`accountManagementUrl` and `passwordChangeUrl` fields, using the existing
server-side identity directory lookup. No new OAuth permissions, credentials,
record formats, migrations or Lexicons are involved.

Provider detection uses the resolved PDS endpoint, not the handle. Bluesky's
`bsky.social` and `*.host.bsky.network` hosts link to
`https://bsky.social/account`; their password action opens
`https://bsky.app/settings/account`, where users select Password. Independent
HTTPS PDS origins link to `/account`, the reference implementation's convention;
the UI explains that available settings vary. Endpoints containing credentials,
paths, queries or fragments do not produce hosting links. Missing hosting
information leaves account details and CraftSky deletion usable, with retry.

The clients open links externally and require returned details to match the
current DID. The mobile provider also checks the account lease before exposing a
completed lookup; the web provider component is recreated when accounts change.
