# Opt-in live smoke test

This suite uses Patrol 4.10.0 and patrol_cli 4.8.0. It is separate from
`test/`, the loopback `integration_test/` suite, and CI. The initial runner
targets a dedicated Android emulator with an English UI and Chrome selected as
the default browser. Finish Chrome's first-run prompts before testing.
iOS and web runners are not configured yet.

## Run locally

From the repository root:

```sh
dart pub global activate patrol_cli 4.8.0
# Add ~/.pub-cache/bin to PATH if needed.
cp app/.patrol.env.example app/.patrol.env
# Fill in E2E_HANDLE and E2E_PASSWORD; no quotes.
just dev
flutter emulators --launch <emulator-name>
# Wait until Android has finished booting; get the device ID from flutter devices.
just app-test-e2e emulator-5554
```

Use a dedicated disposable Bluesky-hosted account with a full account password
(OAuth browser login does not use app passwords). The profile bio is replaced.
The test creates public PDS records, even though AppView runs locally. Do not
use a personal account. Credentials belong only in the ignored `.patrol.env`.
Patrol compiles these defines into local test builds, so do not distribute test
APKs or publish build artifacts. The wrapper redacts account values from runner
output; avoid verbose Gradle output or inspecting build process arguments.
The live wrapper sets Android's `log.tag.PatrolServer` to `INFO` because Patrol
4.10's native DEBUG text-entry logs include entered values. Run the live journey
through the wrapper to keep that source filter in place.
It also encodes the password before passing it to Dart, preserving characters
such as `#` that Patrol's own dotenv reader treats as comments.

The wrapper discovers this checkout's Docker AppView port, verifies health,
resolves the test account DID, and removes only that DID's local onboarding
completion. It does not reset the entire database or change production state.
It configures ADB reverse and copies the two synthetic PNGs into the emulator's
`Pictures/CraftSkyE2E` album. Never run this command on a personal device.

The UI journey performs real OAuth through Chrome, changes the onboarding bio,
completes onboarding, selects both photos through Android's picker, publishes a
regular post, likes it, and deletes it. Fresh AppView reads verify the bio,
indexed post/image metadata, persisted like, and eventual 404 after deletion.
Both published image URLs must serve nonempty bytes. Indexing is polled with a
60-second bound. Authentication, repositories and upload plugins are not mocked.
Push delivery, billing and remote crash reporting are outside this smoke test.
The photo-picker helper requires exactly two fixtures in `CraftSkyE2E` and
checks the native selection count after each photo tap before confirming Done.
A stalled selection reports the expected and observed photo counts.

Each run uses a unique bio/post marker. After finding the published post, the
test attempts UI deletion in a `finally` block even if the like check fails.
A failure before that point or during cleanup may leave its own public post
behind; inspect the failure and remove that marked post through the app.
The runner never bulk-deletes PDS records. The next run resets onboarding only.
The emulator's browser may retain an OAuth login between runs.

## Harness check

No account or backend is required for the minimal native automation check:

```sh
just app-test-e2e-harness emulator-5554
```

Patrol automatically loads `.patrol.env` whenever it is present, including for
the harness. Run only named targets. The live journey deliberately fails if
started without the wrapper's `E2E_LIVE` opt-in.

Tests call feature modules; `System` owns OS interactions and `ApiClients` owns
fresh backend verification. Shared keys are aggregated by `lib/keys.dart`.
Use `patrol test`, not `flutter test`, for files in this directory.

Native OAuth and photo-picker selectors are provider/Android-version specific.
If those screens change, update their modules from the actual native tree.
Patrol MCP can be installed separately for interactive test sessions, native
tree inspection and screenshots: <https://pub.dev/packages/patrol_mcp>.

## Validation

On 20 October 2026, the full live journey passed twice consecutively after
adding native selection-state checks for an intermittent missing-Done failure.
Both runs used the `pixel-10` Android 17 (API 37) emulator with Chrome, this
checkout's Docker AppView, and the dedicated real Bluesky account. Each UI
journey took 64 seconds; Patrol reported 93.6 and 93.3 seconds for native test
execution, in addition to the APK builds. The original failure did not recur
during debugging, so its precise timing cause remains unconfirmed.

Flutter analysis passed after the picker update. The independent native harness
and 124 focused widget tests passed during initial setup. The native log filter
was checked with a protected debug canary and a retained INFO marker. iOS, web,
older Android photo pickers, and physical devices have not been validated.
