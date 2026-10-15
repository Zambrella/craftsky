# CraftSky publishing indicator

The selected stitch-orbit study, prepared for Flutter: transparent 72×72 artboard, a rotating running-stitch ring and red leading stitch, and an arrow that rises and fades twice per 2.4-second loop. Only this artboard is bundled in the app.

- Artboard: `Publishing`
- Default state machine: `Publishing`
- View model: `PublishingTheme`; color property: `threadColor`
- Runtime: `rive` 0.14.11, using `Factory.flutter` for this vector-only graphic
- App asset: `app/assets/animations/publishing.riv`

Flutter binds `threadColor` to the theme primary color, including the lighter cobalt used in dark mode. The red accent remains fixed. Reduced motion holds a readable pose with the arrow visible and disables the animation ticker. The overlay provides localized status semantics. Loading/decoding failures keep the existing stitched indicator visible.

Both post and project composers watch the auto-disposed animation-file provider before submission to preload the asset. Each visible indicator owns its controller and view model; the provider owns the shared decoded file and releases it after the composer closes, including when decoding finishes late.

## Editing and rebuilding

From the repository root:

```sh
rive design/rive/publishing-indicator
./design/rive/publishing-indicator/build-asset.sh
```

Edit `scene.rml`, then run the script to verify, inspect, compile, and replace the bundled asset. Commit source and asset together. This file uses only paths, timelines, and color data binding. It has no scripts/shaders that require signed publishing; if scripts are introduced, use the CLI's authenticated signed publishing flow and recheck runtime support.

The original three studies remain in `../publishing-loaders/` for design reference. They are not bundled.

## Flutter preview

This runs the actual submission overlay and current app themes without signing in, connecting to AppView, or publishing a post:

```sh
cd app
flutter run -d web-server --web-port 7357 -t test/fixtures/publishing_animation_preview.dart
```

Use the switches to inspect light/dark themes and reduced motion. For a native preview, replace `web-server` with a supported device ID.

## Verification

```sh
cd app
flutter test test/theme/publishing_progress_indicator_test.dart test/feed/widgets/submission_overlay_test.dart test/feed/widgets/video_submission_overlay_test.dart
flutter analyze
```

The real-asset widget test needs the Rive native library. On a fresh macOS development machine, if Rive reports that the library is missing, run `dart run rive_native:setup --platform macos` before testing. The native application build performs its own platform setup. The Flutter CI job downloads the corresponding Linux test library before running the suite.
