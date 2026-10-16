# Stitched-heart like reaction

The selected, corrected running-stitch heart from `../like-reactions/`. The app bundles only this transparent artboard; the comparison backgrounds and other concepts are excluded.

- Artboard: `Like`, 160×160
- Default state machine: `Like`
- One-shot timeline: 84 frames at 60 fps (1.4 seconds)
- The red seam draws between frames 8 and 36; there is no separate leading needle or solid overlay.
- App asset: `app/assets/animations/like.riv`

`PostCard` wraps both its like button and existing double-tap callback with local feedback. Only the initiating card animates, immediately with the optimistic action. An unlike, failed mutation, post identity change, active-account/session change, or disposal clears it. The overlay is centred, decorative, and ignores pointer events. Already-liked double taps keep their existing no-op behavior.

The decoded file is shared and preloaded across mounted cards. Each playback owns its controller. A like never waits for asset loading: if the file is unavailable, a static red heart appears instead. Reduced motion holds a completed stitched-heart pose for 400 ms. No like action gets a loading spinner.

## Edit and rebuild

Edit `scene.rml`, then run from the repository root:

```sh
./design/rive/like-indicator/build-asset.sh
```

Commit the RML and bundled `.riv` together. This asset uses the existing Rive Flutter runtime and Flutter canvas renderer. It contains no scripts, shaders, fonts, or external assets.

## Preview the real card

This fixture stores its like state in memory and uses an empty session registry. It sends no likes to AppView and does not read saved credentials.

```sh
cd app
flutter run -d web-server --web-port 7359 -t test/fixtures/like_animation_preview.dart
```

Tap Like or double-tap the post body. Use the switches for dark mode and reduced motion. Replace `web-server` with a supported device ID for a native preview.

`previews/in-app-like.mp4` is a cropped recording of the real card running in the collaborative browser.

## Focused checks

```sh
cd app
flutter test test/feed/widgets/post_like_reaction_test.dart test/feed/widgets/post_card_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart
flutter analyze
```

The real-asset tests require Rive's native test library; on a fresh macOS development machine run `dart run rive_native:setup --platform macos` first. The existing Flutter CI job sets up the Linux library before testing.
