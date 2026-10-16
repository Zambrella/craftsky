# Craft-themed Follow-action studies

Three standalone Rive options for a compact Follow-button interaction. These are design previews, not changes to the app's follow behavior.

| Artboard | Motion |
| --- | --- |
| Tie the knot | Cobalt and red thread loops draw on from opposite sides, meet, and pull into a small bow-shaped knot. |
| Sewn together | Two fabric patches slide together; three red cross stitches close the seam. |
| Button buddies | Two buttons roll towards each other, then a red thread draws through their holes to connect them. |

Individual artboards are transparent, 160×120, with a default `Follow` state machine. Each one-shot runs for 1.2 seconds and holds its completed connected pose. There are no scripts, shaders, fonts, or external assets in the Rive graphic. Thread reveals use trims on the actual paths, without independent moving needles.

`Comparison` places the three studies in white button-shaped surfaces on CraftSky paper, replaying every two seconds. The GIF adds presentation titles and a Follow/Following label to illustrate the state transition; these labels are not part of the Rive source. The interaction shown is a confirmation concept. The app's actual follow completion/error timing, reduced-motion behavior, final button layout, and small-size rendering remain integration work after selection.

## Preview

From the repository root:

```sh
rive design/rive/follow-reactions
rive design/rive/follow-reactions --artboard="Tie the knot"
```

The native comparison loops automatically. Individual artboards play once and hold. Use `a` in the preview terminal to select an artboard.

## Edit and validate

```sh
python3 design/rive/follow-reactions/generate_scene.py
rive design/rive/follow-reactions --verify
rive inspect design/rive/follow-reactions --summary
rive design/rive/follow-reactions --once
```

Edit `generate_scene.py` to keep the comparison and individual artboards consistent. Regeneration replaces `scene.rml`. To render and label the comparison, run `python3 design/rive/follow-reactions/render_previews.py` (requires Pillow and graphics access for the Rive CLI).

`previews/comparison.gif` uses 50 ms frame delays, matching the Rive timeline's actual two-second replay cadence. `previews/comparison.png` is the settled pose.

No project has been pushed to a Rive account or published to the web.
