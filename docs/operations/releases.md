# Releases

Craftsky releases are created, built, and deployed from the maintainer's local
machine. GitHub Actions only tests pull requests. AppView and the Flutter app
have independent versions, tags, and changelogs.

## Shared rules

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

`release-push` requires the release commit to remain exactly one commit ahead of
freshly fetched `origin/main`. It pushes `main` and the tag atomically. If
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

Build the signed store artifacts before pushing the release:

```sh
export SENTRY_AUTH_TOKEN=...
export SENTRY_ORG=...
export SENTRY_PROJECT=...
just app-build-ipa production
just app-build-appbundle production
```

Production IPA, APK, and app-bundle recipes require clean `main` at the matching
annotated app tag. They run `flutter pub get`, `flutter analyze`, and
`flutter test`, then build with obfuscation, upload matching Sentry symbols, and
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
