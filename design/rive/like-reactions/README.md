# Craft-themed like reactions

Three standalone Rive studies for a brief animation centred over the liked post. No Flutter behavior has been changed by these studies.

| Artboard | Concept |
| --- | --- |
| Stitched heart | Red running stitches draw the heart seam over a soft pink fabric fill, with matching red stitch sparks. There is no separate leading needle or solid overlay. |
| Patchwork pop | Lilac, red, clay and butter fabric panels assemble into a heart with running-stitch seams. Little fabric scraps scatter outwards. |
| Button love | A red heart-shaped button pops into view; two cream threads pull through its four holes to form a sewn cross. |

Each individual artboard is transparent and 160×160. Its `Like` state machine automatically plays a 1.4-second, one-shot timeline: pop in, craft detail, brief hold, then fade upwards. The reaction is fully invisible by the end. There are no scripts, shaders, external images, fonts, or network dependencies in the Rive file.

The `Comparison` artboard shows the reactions in the centre of three mock fabric swatches. It replays every 2.4 seconds to make review easier. These swatches are presentation backgrounds, not part of the transparent reaction artboards.

## Preview

From the repository root:

```sh
rive design/rive/like-reactions
rive design/rive/like-reactions --artboard="Stitched heart"
```

In the native preview terminal, enter `a` to choose an artboard. Restart an individual artboard to replay its one-shot animation. The comparison repeats automatically.

`previews/comparison.gif` contains frames rendered with the Rive CLI, with presentation titles added underneath. `previews/comparison.png` is a representative frame.

## Edit and validate

The standard-library Python generator keeps the repeated comparison and individual artboards consistent. Edit `generate_scene.py` and regenerate the editable `scene.rml`:

```sh
python3 design/rive/like-reactions/generate_scene.py
rive design/rive/like-reactions --verify
rive inspect design/rive/like-reactions --summary
rive design/rive/like-reactions --once
```

Regenerating replaces manual edits to `scene.rml`. Native Flutter/web runtime checks, final display size, reduced-motion behavior, like-action replay, and integration into the post overlay come after a direction is chosen. Nothing has been uploaded or published to a Rive account.
