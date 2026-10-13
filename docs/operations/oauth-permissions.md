# OAuth permission publication and production rollout

CraftSky requests `include:social.craftsky.authFull` plus explicit Bluesky,
media and video-service scopes. The canonical client ID is
`https://appview.craftsky.social/oauth-client-metadata.json`.
See [ADR 017](../../adr/017-granular-oauth-consent.md).

## Publish before activating

From the reviewed checkout, use the official publishing account and verify:

```sh
scripts/lexicon-publication manifest
scripts/lexicon-publication publish --bootstrap
scripts/lexicon-publication configure-dns --dry-run
scripts/lexicon-publication configure-dns
scripts/lexicon-publication verify
scripts/oauth-permissions-check --publication-only
```

`--bootstrap` permits creating the new permission set before its new root DNS
record exists. The script never updates existing schemas. Review the dry-run;
the new authority is `_lexicon.craftsky.social` with the existing official DID.
Retain publication and DNS verification evidence. If public resolution is not
working, do not activate the include scope.

## Migrate the old OAuth client

The metadata URL change creates a different OAuth client identity. Before the
application switch, use a brief maintenance window to prevent new logins and
wait for in-flight OAuth callbacks to complete or expire. While the old deployment
is still running, revoke each remaining account's sessions with the supported
production command:

```sh
/app/cli --env prod sessions revoke '<DID>'
```

Use the production runbook's Render SSH procedure. Wait for the old deployment's
revocation worker to remove both access and refresh credentials. Verify there are
no remaining ordinary OAuth sessions, pending/unverified callback credentials or
accepted deletion credentials before switching client ID. A pending deletion must
reach its existing reauthentication state safely; do not delete rows to force the
migration. Store only counts and approved operational references in the receipt.

The application also prevents old broad grants from performing further ordinary
PDS operations and uses `401 pds_permissions_required` to request fresh consent.
This guard does not replace draining old-client credentials during this rollout.
Future permission-set additions preserve the client ID and require no such drain.

## Configure and deploy

Validate `render.yaml` with the Render CLI and confirm `valid: true`. Review and
manually synchronize the CraftSky production Blueprint/environment group's exact
`OAUTH_SCOPES` value. Application releases do not synchronize the Blueprint.
Publish the permission set first; stale `transition:generic` configuration will
cause the new application to reject startup rather than request broad access.

Create and verify an explicit AppView release version from clean, synchronized
`main` using [the release workflow](releases.md). Deploy its pushed immutable tag
using `scripts/appview-deploy`. The script verifies the permission set and DNS
against the tagged source before deployment, then verifies the live canonical
client ID and exact scopes after health succeeds.

Reopen login and exercise new consent, profile updates, post/media creation,
likes, follows, blocks and video authorization when video is enabled. Confirm the
consent screen shows the readable CraftSky title and domain. Exercise a missing
permission response and confirm the account-safe sign-in recovery works. Record
these provider/device checks separately from local automated test results.

## Evolve and recover

Provider permission-set caches may remain stale for up to 24 hours. Updates apply
when access tokens refresh; do not assume that waiting 30 minutes guarantees
availability. Verify missing-permission recovery before releasing a new collection.

Keep the permission set and root DNS authority published during rollback. Do not
restore transitional permissions. An older release may require the direct granular
scope list instead of `include:`; review its requirements and drain credentials
before changing client IDs again. Never move a production tag or delete PDS records
as part of rollback.
