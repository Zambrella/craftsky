# Drag-controlled yarn refresh

The approved vertical yarn, packaged as a script-free Rive asset. The host controls `strandLength`, `ballScale`, `rotation`, `yarnOpacity`, `checkOpacity`, and theme-aware `inkColor` through the `YarnRefresh` view model. The `YarnRefresh` artboard is 80 × 240; its neutral state machine applies bindings without an autonomous timeline.

`YarnRefreshScrollView` inserts a real refresh sliver after any fixed header slivers. The loose strand shares the ball’s theme-aware outline colour and follows a soft cubic S-curve; its endpoint follows the exposed sliver height directly. The host also sets `strandFirstY`, `strandSecondY`, and `strandHandle` to preserve the curve as the pull length changes. The 36-pixel ball keeps a constant size, with curved overlapping wraps that suggest wound yarn. The yarn fades in over the first 56 pixels of drag and fades back out as the pull retracts. Reversing a pull rewinds the yarn. Refresh is deferred until the pointer is released beyond the 112-pixel threshold. While the callback is pending, the ball winds in the held 96-pixel space. A successful callback crossfades from yarn to a check over 120 milliseconds before spring retraction; a failed or cancelled callback does not celebrate. Screens whose providers retain failures in their state omit the success check, so a completed attempt cannot look like confirmed success. Reduced motion disables rotation and switches directly to the check. Unavailable assets fall back to a static arrow or the existing loading indicator.

The shared control replaces existing pull-to-refresh surfaces, preserving their callbacks, pagination keys and scroll controllers. The feed retains its pinch-zoom guard. Active account lease changes dispose any pending visual state.

```sh
python3 design/rive/yarn-refresh/generate_scene.py
bash design/rive/yarn-refresh/build-asset.sh
cd app
flutter run -d web-server --web-port 7362 --target test/fixtures/yarn_refresh_preview.dart
```

The preview uses real post cards and the shared refresh control with a local delayed callback; it makes no network writes.
