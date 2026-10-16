# Follow reaction: original Tie the knot

The original approved blue-and-red bow, exported as a script-free 2.4 KB Rive asset. The transparent `Follow` artboard is 160 × 120 and its `Follow` state machine plays a 72-frame, 60 fps one-shot timeline, holding the finished knot.

Regenerate and bundle:

```sh
python3 design/rive/follow-indicator/generate_scene.py
bash design/rive/follow-indicator/build-asset.sh
```

`FollowReactionScope` owns feedback for one initiating profile or notification and active account lease. `FollowReactionAvatar` paints over its avatar without changing layout, semantics or hit testing. Successful new follows play the original motion, hold, then fade out over the final 200 ms of a 1.4-second feedback window. Reduced motion shows the completed knot for 400 ms. Unfollow, failed requests, unavailable assets, disposed surfaces and stale account completions do not celebrate. Asset loading never delays a mutation.

Integrated into full profiles, compact profile cards, and notification Follow buttons. Existing server-pending button states and mutation handling are preserved. The separate Instagram migration acceptance workflow is unchanged.

Local in-memory preview (no network writes):

```sh
cd app
flutter run -d web-server --web-port 7360 --target test/fixtures/follow_animation_preview.dart
```
