# Instagram export picker migration

Replaces the custom Android MethodChannel/Kotlin document picker with
`file_picker` 13.1.0. The resolved Android implementation is
`android_file_picker` 2.1.1, whose document-copy path uses an 8 KiB buffer on
`Dispatchers.IO` and returns a local cache path rather than the file bytes.
The package owns platform integration for Android, iOS, and web.

Native imports continue using the existing Dart path-based ZIP/JSON parser in
an isolate. Archive metadata, decompressed JSON, and normalized-entry limits
are unchanged. Web retains its size-limited JSON import and rejects unknown
length before reading bytes; this change does not add web ZIP support.

Android and iOS document picks are temporary app-owned copies. The selected
copy is deleted after parsing succeeds or fails. Android's package cache is
also cleared after cancellation, picker failure, or parse completion, removing
partial copies and leftovers from interrupted imports. Calls are serialized
across provider lifetimes so this cleanup cannot race another import. This
flow is currently the app's only `file_picker` consumer; future consumers must
coordinate package-wide Android cache cleanup.

Do not use `FilePicker.clearTemporaryFiles()` on iOS: the inspected
`file_picker_darwin` 2.3.2 implementation clears the entire app temporary
directory. iOS cleanup deletes only the selected copy. Unreturned partial
copies after failed picks or process death are left for system temporary-file
reclamation on iOS.

The custom `InstagramExportPicker.kt`, its MainActivity registration and
lifecycle hooks, and its channel-specific tests are removed. Tests now exercise
the package platform boundary, path-based reads, cancellation, Android/iOS
copy cleanup, failure/retry, concurrent-call rejection, and web size checks.
The existing privacy test continues verifying that only normalized following
usernames cross the repository boundary.

No API or lexicon changes. No deployment or release performed.

## Validation

- `fvm flutter test test/instagram_migration`: 92 passed (rerun after the iOS cleanup restriction).
- `fvm flutter analyze`: no issues after the final cleanup adjustment.
- Android `:app:compileDebugKotlin`: build successful, including `android_file_picker`.
- `fvm flutter build web --debug --no-pub`: built successfully; Wasm dry run succeeded.
- `fvm flutter build ios --simulator --debug --no-codesign --no-pub`: built successfully. Existing media-kit Swift Package Manager adoption warnings remain.
- Physical-device large ZIP, cloud-provider, low-storage and process-death journeys have not been exercised.

## Inspected package sources

- https://pub.dev/packages/file_picker/versions/13.1.0
- https://pub.dev/packages/android_file_picker/versions/2.1.1
- https://pub.dev/packages/file_picker_darwin/versions/2.3.2
- Published source archives were inspected before integration; versions are recorded in `app/pubspec.lock`.
