# Vertical unwinding yarn

A Rive motion demonstration of the proposed drag-controlled refresh. The left panel pulls beyond the threshold and refreshes. The right panel makes a smaller pull, reverses direction, and releases without refreshing. The red outlined dot represents the finger; the numeric caption shows the demonstrated pull distance.

The ball stays beneath the header. A red strand extends vertically to the feed edge, while the ball gets smaller and turns in direct proportion to the drag. Reversing the drag rewinds it. Both the feed offset and the strand endpoint use the same drag-distance samples. On a successful release the ball turns while loading, then fades into a check and the feed settles.

These are timed demonstrations of a gesture, not an interactive drag implementation. The app integration would supply the live scroll overscroll distance to these visual properties and use the actual request lifecycle for refresh completion. No app code is changed here.

```sh
python3 design/rive/vertical-yarn-refresh/generate_scene.py
rive design/rive/vertical-yarn-refresh --verify
rive inspect design/rive/vertical-yarn-refresh --summary
python3 design/rive/vertical-yarn-refresh/render_previews.py
rive design/rive/vertical-yarn-refresh
```

The comparison and separate full/cancel artboards are editable, script-free RML. GIF captions are presentation-only. GIF timing is 20 fps with 50 ms frame delays.
