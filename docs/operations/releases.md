# Releases

Craftsky releases are created, built, and deployed from the maintainer's local
machine. GitHub Actions only tests pull requests. AppView, the Flutter app and the Svelte web client have independent versions,
tags and changelogs. Web releases also record the reviewed Cloudflare Worker
version ID and use manual dashboard promotion.

## Private beta and full public launch

Private beta releases use the normal build, test, signing, tag, and deployment
workflow. Open online-safety gaps do not block beta builds, website publication,
backend startup, or operational health. Invitation and distribution restrictions
are managed by the owner; there is no new DID allowlist in this release.

- Production images default to `IMAGE_SAFETY_SCANNER_MODE=manual`. The worker
  validates image format and permits serving; moderators review content through
  the existing reporting and moderation tools. This is not automated content
  detection. `/healthz` reports automated scanner readiness separately, without
  degrading operational health solely because the scanner is unavailable.
- AppView video endpoints use `VIDEO_ENABLED=true`. Video selection in both
  clients is always available without a client build or environment flag.
- Restricted evidence storage is optional until its S3 credentials are supplied.
  Without it, evidence preservation/access and its retention worker are
  unavailable. Reports, moderation, workflow metadata, holds, and account deletion
  remain available. Existing evidence cannot be deleted without its object store.
- `just public-release-check` runs quality checks without the gap-register gate.
  `just online-safety-readiness` remains an explicit owner-run full-launch check.
  The gap register and policy manifest remain accurate; beta publication does not
  make draft policies approved or close any gaps.

Before full public launch, complete that readiness check, configure the production
scanner and evidence store, and record the required approvals and deployment
verification. Switching to a content scanner requires scanning existing media
under its own scanner/policy/corpus key; manual-mode results are not scanner
approval evidence.

## Public website deployments

The static website has its own local deployment workflow, independent of AppView
and Flutter versions/tags. Use `just web-preview` to review a checked artifact,
then `just web-deploy` from clean, pushed `main` to deploy to the existing
Cloudflare Worker with static assets `craftsky-landing`. Merging changes alone does not
publish them. Both commands run the website checks and verify an uploaded version;
production then assigns that exact version 100% of traffic and verifies
`https://craftsky.social`. Existing custom domains/routes are preserved. See
[`web/README.md`](../../web/README.md#build-and-deploy) for account setup,
deployment receipts, policy-artifact handling and full-site rollback.

## Web client: preview, then release

The Svelte application in `web-client/` has its own Worker,
`craftsky-public-app`, in the existing CraftSky Cloudflare account. It is separate
from the marketing Worker `craftsky-landing`, service status, and AppView on
Render. See [ADR 026](../adr/026-web-client-preview-promotion.md).

The workflow is **create/push a web release tag → upload that exact tag →
review/sign in on the preview → promote that exact Worker version in Cloudflare**. Merging a PR does not publish the client. Do not connect
Workers Builds with its default automatic deploy command, or use `wrangler
deploy` for routine releases: both immediately assign live traffic. The checked-in
preview command only runs `wrangler versions upload`.

### Addresses and preview sign-in

- Stable review/sign-in URL:
  <https://preview-craftsky-public-app.hugetoothstudios.workers.dev>
- Each upload also prints an immutable version URL and full Worker version ID.
  Use the ID to identify the reviewed candidate when promoting it.
- Public release URL: <https://app.craftsky.social>.

The stable alias follows the most recent preview upload. Record the version ID
and avoid uploading another candidate during review or an OAuth flow. Unique
version URLs remain useful for checking the exact rendering, but **sign in on
the stable alias**: AppView permits only that exact preview origin and the public
app origin, not arbitrary version URLs or workers.dev hosts. Preview/public
session cookies are separate; signing in on preview does not sign in on public.

The preview is unlisted and rendered responses have `X-Robots-Tag: noindex`.
Cloudflare can replace the application's `noindex, nofollow` header with its own
`noindex` header on version URLs. Anyone with the link can visit; no Cloudflare
Access policy is required for this workflow. Do not share it as a public launch.

**This preview uses production AppView and real accounts/data. Posts, profile
changes and deletion actions have real effects.** It is not an isolated staging
environment. PDS OAuth credentials stay in AppView.

### One-time setup and backend prerequisite

Initial setup on 2026-11-04 created `craftsky-public-app`, enabled Version URLs,
disabled its ordinary workers.dev route, and uploaded the initial review alias.
The public app domain remains on the existing `craftsky-verified-links` Pages
project until the separately controlled first public release below. No domain
cutover or AppView release was performed by that setup.

Before testing hosted preview sign-in, release AppView with ADR 026's exact preview
callback allowlist using the [normal AppView release](#appview-release). Keep
production `WEB_APP_ORIGIN=https://app.craftsky.social`. The same backend release
must contain the browser-session and public-read support described in
[`web-client/README.md`](../../web-client/README.md). No new migration is needed
for the preview allowlist itself. Local provider fixtures do not prove live PDS
OAuth; verify sign-in after the backend release before promoting the web client.

On a new development machine, install the pinned tools and log in:

```sh
just web-client-setup
cd web-client
npx wrangler login
npx wrangler whoami
cd ..
```

Confirm the account ID matches `web-client/wrangler.jsonc`. Authentication uses
Wrangler's login or an operator-supplied token; never commit credentials.

The first Worker creation required `wrangler deploy`, because Cloudflare rejects
`versions upload` for a Worker that does not exist. That bootstrap is complete;
do not repeat it after attaching the public domain. If recreating a deleted
Worker, separately review the domain/routing state first. The checked-in config
declares no custom domains/routes, and routine uploads preserve dashboard routes.

### Upload a release candidate

The web version source is `web-client/VERSION`; release notes are in
`web-client/CHANGELOG.md` and annotated tags use `web-vX.Y.Z`. The displayed
version and Sentry release both use the web version, independently of AppView.
The initial version baseline is `1.11.1`, matching the mobile version when this
stream was introduced; it is not an assertion of a published web release.

From the repository root, start from clean, reviewed, pushed `main`. For example,
for the first tagged release, prepare a nonempty notes file outside the checkout:

```sh
git switch main
git pull --ff-only origin main
just release-create-web 1.11.2 /absolute/path/to/web-release-notes.md
just web-client-check
just release-push web web-v1.11.2
just web-client-preview web-v1.11.2
```

The first release requires explicit notes; later releases can omit the notes path
to generate notes from relevant commits since the previous web tag. Creation and
atomic push use the same clean-main, increasing-version, changelog and annotated-tag
checks as the other release streams. Push the release commit/tag before adding
further commits. Upload requires the exact tagged commit at HEAD on clean `main`,
a matching version file, an unchanged pushed tag, and ancestry on remote `main`.
It repeats these checks after building, before uploading.

The command checks the target Worker exists, runs `web-client-check` (type checks,
unit tests, build and public/authenticated browser suites on both runtimes), then
uploads a new version with alias `preview`. It checks the hosted welcome/sign-in
pages, native callback content and association files, plus cache/privacy and
preview indexing headers. It **does not** promote traffic, change DNS/domains,
release AppView, or run real account mutations.

The output includes the Git release tag, Worker version ID, immutable review URL and an ignored local
receipt at `web-client/.deployments/<version-id>.json`. Retain the receipt, commit,
Sentry release and review evidence in the operational release record. A failed
hosted check leaves the upload available but does not promote it. If the upload
succeeded before an interruption, use Cloudflare Deployments to identify it
before retrying.

Untagged and dirty uploads are rejected; `--allow-dirty` is no longer supported.
Use local development for uncommitted experiments. Existing experimental Worker
versions remain in Cloudflare history and must not be promoted. Dashboard
promotion is an operator action: Cloudflare does not enforce the Git release
checks, so match the receipt's release tag, commit and Worker version ID.

Keep web and mobile major/minor versions aligned for coordinated feature releases
where practical, with independent patch versions for platform-specific fixes.
Do not hold a web fix for an app-store release or bump AppView solely to match a
client version. This is a release convention, not a cross-stream equality gate.

Set optional `PUBLIC_SENTRY_*` and public Firebase web configuration in ignored
`web-client/config/.env.production.local` (copy its matching example), or in the
build shell environment. Set `PUBLIC_SENTRY_ENVIRONMENT=production`. Development
settings in `.env.development.local` are excluded from production builds. Runtime AppView/subscription settings come
from the Worker config. The default subscription flag is `false`. If reporting
is enabled, upload the exact candidate's private source maps before promotion:

```sh
cd web-client
npm run sentry:upload
cd ..
```

Use the authorized Sentry CLI login. Do not rebuild between preview review,
source-map upload and promotion; retain `.sentry-artifacts` with the matching
build. Source maps are excluded from the public assets and Worker upload.

### Promote the reviewed version in Cloudflare

1. Open **Cloudflare → Workers & Pages → craftsky-public-app → Deployments**.
2. Match the full version ID from the successful preview receipt and review.
   Check its `web-vX.Y.Z` tag and commit message against the receipt and pushed Git
   tag, not just its position in the list. Reject old `-dirty` or untagged uploads.
3. Select **Promote deployment**, choose that version, and assign it **100%**
   of traffic. Confirm the deployment. This uses the uploaded code, assets and
   bindings without rebuilding. Do not use Edit code or retry a build.
4. Once the public domain has been attached (first release below), check it:

   ```sh
   just web-client-verify https://app.craftsky.social
   ```

5. Verify actual sign-in, public post/profile sharing, authenticated navigation,
   image/video uploads as applicable, and sign-out. Recheck app links on signed
   iOS/Android builds. Record the Worker version/deployment IDs and results.
   The smoke command does not authenticate, write posts or prove device linking.

Cloudflare documents [manual promotion](https://developers.cloudflare.com/workers/versions-and-deployments/deployment-management/)
and [version URLs](https://developers.cloudflare.com/workers/versions-and-deployments/version-urls/).
Only the latest 100 uploaded versions are eligible for deployment. Avoid leaving
an approved release behind a large number of subsequent experimental uploads.

### First public release: move the existing domain once

This is a separate production routing change performed when ready to launch.
Routine preview upload does not perform it.

1. Record the current `craftsky-verified-links` Pages deployment, custom-domain
   settings and DNS record for rollback. Keep the Pages project and deployment.
   Confirm `craftsky-public-app` has the reviewed version promoted to 100%.
2. Verify the Worker preview serves both `/.well-known/assetlinks.json` and
   `/.well-known/apple-app-site-association`, and both native callbacks, with
   direct HTTP 200 responses. Release the narrowed Android intent filters from
   ADR 025 before checking public browser login with the native app installed.
3. Remove **only** `app.craftsky.social` from the Pages project's custom domains,
   then add it under **craftsky-public-app → Settings → Domains & Routes → Add →
   Custom Domain**. Cloudflare manages the Worker's DNS/certificate. Review any
   existing-record conflict before replacing it; preserve the recorded Pages DNS
   target for recovery. Do not alter `craftsky.social`, AppView or service-status
   domains. Allow for certificate/DNS activation; this first cutover may have a
   brief interruption.
4. Run `just web-client-verify https://app.craftsky.social`, then the live checks
   above. Do not add cache-everything rules: HTML, API and browser callbacks
   deliberately remain `no-store`.

After this one-time move, future **Promote deployment** actions update
`app.craftsky.social` without DNS changes. Keep the domain managed in the
dashboard; `versions upload` does not synchronize Worker triggers.

### Rollback

In **craftsky-public-app → Deployments**, select the previously recorded good
version and promote it to 100%. Verify the public host again. This restores the
frontend version/configuration, not database content, PDS writes or backend
migrations. A failed post-promotion smoke check means live state has changed;
inspect and explicitly roll back rather than assuming an automatic recovery.

For the first cutover only, if the Worker cannot serve the domain reliably,
remove its custom domain, reattach `app.craftsky.social` to the retained
`craftsky-verified-links` Pages project, and restore/verify the recorded Pages DNS
configuration and prior production deployment. Check both association documents
and native callbacks again. Pages preview deployments cannot be used as
[production rollback targets](https://developers.cloudflare.com/pages/configuration/rollbacks/).

## Shared rules for AppView and Flutter releases

- Start on a clean local `main` synchronized with `origin/main`.
- Supply the next version explicitly. The release script does not infer SemVer
  from pull requests or labels.
- By default, release notes are the non-merge commit subjects since the previous
  stream tag, limited to that stream's paths. Write clear commit subjects.
- Pass a nonempty Markdown notes file to replace generated notes when a release
  needs curated wording. The file may live outside the worktree.
- Release tags are annotated and immutable. Never move, recreate, or force-push
  a published tag.
- `scripts/release create` creates a local release commit and tag only.
- Build and verify the local tagged commit before `scripts/release push` sends
  the commit and tag to GitHub atomically.
- GitHub Releases are not used. The tracked changelogs are the release history.

## AppView release

`appview/VERSION` is the semantic version source, `appview/CHANGELOG.md` is the
history, and release tags use `prod-vX.Y.Z`. Production binaries embed this
version for `appview version`, startup diagnostics, and the default Sentry
release `craftsky-appview@X.Y.Z`.

Create the local release transaction:

```sh
git switch main
git pull --ff-only origin main
just release-create-appview 1.0.4
```

The command validates that the explicit version is higher than the latest
strict production tag, generates a dated changelog section from relevant commit
subjects, commits the version and changelog together, and creates the matching
annotated tag locally. To replace generated notes, pass a Markdown file as the
second argument:

```sh
just release-create-appview 1.0.4 /tmp/appview-release-notes.md
```

Test the exact release commit, then push it:

```sh
just appview-check
just release-push appview prod-v1.0.4
```

The AppView Docker build also runs the moderation console's pinned frontend
tests, TypeScript check and Vite production build in its Node build stage. Only
the resulting `admin/dist` assets enter the runtime image. `appview-check`
verifies `/admin/`, its referenced JavaScript/CSS and the unauthenticated identity
boundary from that packaged image, using an isolated local instance with admin
routes enabled. It does not provision operators or change production flags.
Console changes therefore use the same exact-tag release as the backend; no
separate frontend deployment or Vite server is needed.

Before enabling the console in production, follow the provisioning, OAuth,
permission, evidence-store and recovery checks in
[Moderation operator access](moderation-operators.md#production-rollout).
Release health checks alone do not verify a permitted operator workflow.

`release-push` requires the release commit to remain exactly one commit ahead of
freshly fetched `origin/main`. It pushes `main` and the tag atomically without an
interactive prompt, so run it only after verifying the local tagged build. If
`origin/main` advanced, do not force the release through; recreate it on the new
tip.

Export local Render credentials and deploy the already-pushed tag:

```sh
export RENDER_API_KEY=...
export RENDER_SERVICE_ID=...
just appview-deploy prod-v1.0.4
```

The deploy command resolves the remote tag to an exact commit, verifies that the
tag version matches the tagged `appview/VERSION`, triggers Render for that SHA,
rejects a mismatched deployed SHA, and waits for `/health` and `/healthz`. Save
its tag, commit, deploy ID, migration, and health output with the operational
release record.

Blueprint synchronization remains a separate reviewed infrastructure operation.
An AppView application release never synchronizes `render.yaml`.

## App release

`app/pubspec.yaml` is the `X.Y.Z+N` version source,
`app/CHANGELOG.md` is the history, and release tags use
`app-vX.Y.Z+N`. Both the marketing version and store build number must be greater
than the prior app release.

Create the local release transaction:

```sh
git switch main
git pull --ff-only origin main
just release-create-app 1.1.0+2
```

For the first app release, or whenever curated store-facing wording is needed,
pass a Markdown notes file as the second argument. Automatic generation fails
closed when there is no prior stream tag or no relevant commit since that tag.

### Run tests

Run the Flutter test suite on the exact local release commit before building:

```sh
just app-test
```

Continue only after the tests pass. The store build commands do not run tests.

### Build and publish

Build the signed store artifacts before pushing the release:

```sh
export SENTRY_AUTH_TOKEN=...
export SENTRY_ORG=...
export SENTRY_PROJECT=...
just app-build-ipa production
just app-build-appbundle production
```

Production IPA, APK, and app-bundle recipes require clean `main` at the matching
annotated app tag. They run `flutter pub get` and `flutter analyze`,
then build with obfuscation, upload matching Sentry symbols, and
print the artifact SHA-256 plus debug-info locations. `app-build-ios` is an
unsigned smoke build and is not a store artifact.

After both desired artifacts succeed, push the release commit and tag:

```sh
just release-push app app-v1.1.0+2
```

Upload and submit the exact locally built IPA and AAB manually in App Store
Connect and Google Play. Retain their checksums, debug-info directories, and
obfuscation maps with the release record. Verify signing identities, bundle or
application IDs, associated domains, and the production APNs entitlement on the
final signed artifacts.

The recipes print checksums for `app/build/ios/ipa/*.ipa` and
`app/build/app/outputs/bundle/release/app-release.aab`. Retain those files with
`app/build/debug-info/ipa`, `app/build/debug-info/appbundle`, and the matching
`app/build/app/obfuscation-*.map.json` files. Inspect the final artifacts before
upload:

```sh
codesign --display --verbose=4 app/build/ios/archive/Runner.xcarchive/Products/Applications/Runner.app
codesign --display --entitlements :- app/build/ios/archive/Runner.xcarchive/Products/Applications/Runner.app
jarsigner -verify -verbose -certs app/build/app/outputs/bundle/release/app-release.aab
```

## Failed local release

Before `release-push`, a release commit and tag exist only locally. If a build or
check fails, do not push them. Correct the underlying code through the normal PR
flow, return to the updated `origin/main`, and create the release again. Do not
reuse a version or tag that was ever pushed.
