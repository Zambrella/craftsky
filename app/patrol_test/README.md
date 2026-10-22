# Opt-in live smoke test

This suite uses Patrol 4.10.0 and patrol_cli 4.8.0. It is separate from
`test/`, the loopback `integration_test/` suite, and CI. Use a dedicated Android
emulator or iOS simulator with an English UI. Android requires Chrome as the
default browser with its first-run prompts completed; iOS uses Safari. Web and
physical-device runners are not configured.

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

For a running iOS simulator, use the same account file and local stack:

```sh
# Requires Xcode and its iOS simulator runtime.
just app-test-e2e booted
# If multiple iOS simulators are booted, choose one explicitly:
xcrun simctl list devices booted
just app-test-e2e <simulator-UUID>
```

`booted` requires exactly one running iOS simulator. Patrol reinstalls the app
on the selected simulator. The iOS UI-test target uses the existing Swift
Package Manager setup; no manual Xcode target setup is needed after checkout.

Use a dedicated disposable Bluesky-hosted account with a full account password
(OAuth browser login does not use app passwords). The profile bio is replaced.
The test creates public PDS records, even though AppView runs locally. Do not
use a personal account. Credentials belong only in the ignored `.patrol.env`.
Patrol compiles these defines into local test builds, so do not distribute test
APKs, iOS test bundles, or build artifacts. The wrapper redacts account values
from runner output; avoid verbose build output or inspecting process arguments.
The live wrapper sets Android's `log.tag.PatrolServer` to `INFO` because Patrol
4.10's native DEBUG text-entry logs include entered values. Run the live journey
through the wrapper to keep that source filter in place.
On iOS, a UI-test-only Patrol extension enters the password directly into
Safari's secure field on `bsky.social`, avoiding Patrol's text-value logging.
This extension is compiled into `RunnerUITests`, not the application target.
The wrapper encodes the password before passing it to Dart, preserving characters
such as `#` that Patrol's own dotenv reader treats as comments.

The wrapper discovers this checkout's Docker AppView port, verifies health,
resolves the test account DID, and removes only that DID's local onboarding
completion. It does not reset the entire database or change production state.
On Android it configures ADB reverse and copies the two synthetic PNGs into
`Pictures/CraftSkyE2E`. On iOS the simulator reaches the Mac's loopback directly,
and `simctl addmedia` imports fresh-dated copies into Photos. The iOS picker
selects the newest two photo tiles. Imports accumulate in the test simulator;
the runner does not delete library photos. Never run it on a personal device.

The UI journey performs real OAuth in the browser, changes the onboarding bio,
completes onboarding, selects two photos through the native picker, publishes a
regular post, likes it, and deletes it. Fresh AppView reads verify the bio,
indexed post/image metadata, persisted like, and eventual 404 after deletion.
Both published image URLs must serve nonempty bytes. Indexing is polled with a
60-second bound. Authentication, repositories and upload plugins are not mocked.
Push delivery, billing and remote crash reporting are outside this smoke test.
Before publishing, the test checks the selected image dimensions and fixture
colors and requires one coral and one teal image.
The Android photo-picker helper requires exactly two fixtures in `CraftSkyE2E` and
checks the native selection count after each photo tap before confirming Done.
A stalled selection reports the expected and observed photo counts.

On iOS, the session registry uses the real Keychain plugin in the
`craftsky.patrol.e2e` service namespace. Only that test registry is cleared at
startup so login runs again even though Keychain survives app uninstall.

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
just app-test-e2e-harness booted
```

The harness wrapper overrides account defines with empty values and does not
require `.patrol.env` or Docker. Run only named targets. The live journey fails if
started without the wrapper's `E2E_LIVE` opt-in.

Tests call feature modules; `System` owns OS interactions and `ApiClients` owns
fresh backend verification. Shared keys are aggregated by `lib/keys.dart`.
Use `patrol test`, not `flutter test`, for files in this directory.

Native OAuth and photo-picker selectors depend on the provider and OS version.
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

The iOS native harness and the complete live journey passed on the iPhone 18 Pro
simulator running iOS 27.0 with Safari and the same local AppView/account. Two
consecutive full journeys took 50 and 47 seconds (72.8 and 68.3 seconds of
native test execution), including fresh login, fixture verification, persisted
bio/post/like checks, and confirmed post deletion. Native runner logs retained
extension startup diagnostics without the raw password or its pre-`#` prefix.
The Android regression run with the shared fixture check also passed: 62 seconds
for the journey and 91.0 seconds for native test execution.

Flutter analysis, Python compilation, Xcode project plist validation, and
whitespace checks passed. The Android harness and 124 focused widget tests
passed during initial setup. The Android native log filter was checked with a
protected debug canary and a retained INFO marker. Web, older OS/photo-picker
versions, and physical devices have not been validated.
