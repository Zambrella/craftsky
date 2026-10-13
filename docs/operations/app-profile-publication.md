# CraftSky app profile publication

CraftSky describes itself to Atmosphere directories using the published
`community.lexicon.app.profile` schema. The single record is owned by the
official `craftsky.social` account:

```text
at://did:plc:lmmx63zcns6gewgxqfdt4kof/community.lexicon.app.profile/self
```

The reviewed source is [`metadata/app-profile/profile.json`](../../metadata/app-profile/profile.json).
It declares the nine collections the client/AppView produce and consume, and
`social.craftsky.actor.profile/self` as the record indicating CraftSky use.
Reusable definitions, embed objects and RPC methods are not record collections
and are not listed as interoperability capabilities. This operator-owned record
is separate from members' profiles; it adds no app OAuth grants, ingestion
filters, API routes or database state.

The app is marked `unreleased`, matching the public waiting-list website.
iOS and Android describe the native app's platforms. The profile also links to
the [Apple App Store destination](https://apps.apple.com/app/id6812428632), using
the maintainer-provided app ID, and the verified
[CraftSky Social Google Play listing](https://play.google.com/store/apps/details?id=social.craftsky.app).
Store links identify the app's destinations; change launch status separately
when the release is confirmed. The current app icon is referenced
as a PDS blob, rather than the older monogram on the marketing website.

## Validate and inspect

Run from the repository root with `goat` installed:

```bash
scripts/app-profile-publication validate
scripts/app-profile-publication show
python3 -m unittest discover -s scripts -p test_app_profile_publication.py
```

Validation is offline against the two unchanged community schemas in
`metadata/app-profile/schemas/`. Their public AT URIs and CIDs are recorded in
`schema-pins.json`. They were retrieved from the community authority
`did:plc:mtr7qrqtcyseedx3jyr5o7db` on 13 October 2026. Updating these dependencies
is a reviewed source change, not a network fetch during a build.

The source is a complete record and includes the initial `createdAt`. Leave
that timestamp unchanged for routine metadata updates; the command also retains
an existing public record's creation time. It manages `updatedAt` only when
metadata changes. After changing the app icon, update its raw SHA-256 CID and
byte size in the source; validation rejects a stale reference. Each image has
exactly one blob or remote URL, as required by the schema's image semantics.

## Publish and verify

Publication is an explicit public-record mutation. Use the maintainer's `goat`
login for the exact official DID; credentials stay in CLI-managed storage,
outside this repository and command output.

```bash
scripts/app-profile-publication publish
scripts/app-profile-publication verify
```

The command resolves the official account's current PDS, reads the existing
record, validates the source, and checks the authenticated DID. When needed, it
uploads only the current public app icon and writes only the app profile at
`self`. Every write is locally validated against the pinned published schemas.
The `putRecord` request leaves `validate` unset, using the protocol default of
server validation for schemas known to the PDS. The official account's PDS
currently rejects `validate: true` with an unknown-lexicon error for this
community schema, even though public lexicon resolution succeeds. The command
never sends `validate: false` or skips local validation.
Creation uses `swapRecord: null`
to require absence; an update uses the observed CID to reject a concurrent
replacement. A matching record is verified without uploads, writes or timestamp
churn. Known metadata omitted from the source is removed on update; future
unknown fields are preserved.

After a write, unauthenticated public readback must match the expected complete
record and the returned CID. If a write times out or fails, run `verify` and
inspect public state before retrying. The command does not automatically replay
an uncertain write or delete any records. Publication does not deploy AppView,
change DNS, update the marketing website or guarantee immediate directory
indexing. To revert metadata, restore the previous reviewed source and publish
again; the same record identity and original creation time remain.

Initial publication on 13 October 2026 was verified through unauthenticated
record readback and a byte-for-byte public icon download. The profile CID was
`bafyreifuniolox5qa2pluvjic642t2mtxiiqryfbzqknumbtjeqvoltvii`.

## References

- [Published app profile schema](https://lexicon.garden/lexicon/did:plc:mtr7qrqtcyseedx3jyr5o7db/community.lexicon.app.profile/docs)
- [Published shared definitions](https://lexicon.garden/lexicon/did:plc:mtr7qrqtcyseedx3jyr5o7db/community.lexicon.app.defs/docs)
- [Community App Lexicon WG](https://discourse.atmosphere.community/t/community-app-lexicon-wg/761)
- [CraftSky lexicon publication](lexicon-publication.md)
