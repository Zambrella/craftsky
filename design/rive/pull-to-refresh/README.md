# Pull-to-refresh studies

Three standalone Rive concepts, presented above a simplified moving feed:

1. **Unspooling yarn**: dragging reveals a loose red strand and a blue yarn ball. The ball winds during refresh.
2. **Stitch circle**: pulling draws a red stitched ring around a downward arrow. The arrow disappears and the stitched ring turns during refresh.
3. **Little loom**: pulling reveals the warp and courses of a tiny weave. A shuttle moves across while the courses pulse during refresh.

Each finishes with a blue check, then the feed returns to rest. The comparison repeats a 3.6-second demonstration: pull, release, refresh, success, settle. These are timed motion studies, not yet bound to an actual gesture or request; production integration would bind pull distance to progress and let the refreshing phase loop until completion. No app code is changed.

The project has a comparison artboard and three individual transparent indicator artboards. All vectors and timelines are script-free. No remote publishing is required.

```sh
python3 design/rive/pull-to-refresh/generate_scene.py
rive design/rive/pull-to-refresh --verify
rive inspect design/rive/pull-to-refresh --summary
python3 design/rive/pull-to-refresh/render_previews.py
rive design/rive/pull-to-refresh
```

The GIF is rendered from Rive at 20 fps with uniform 50 ms delays. Labels are presentation-only overlays.
