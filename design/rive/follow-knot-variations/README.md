# Follow: knot variations

Two additional studies alongside the selected original, using the Rive CLI.

- **Tie the knot**: the unchanged original, for comparison.
- **Gentle pull**: broad loops arrive from opposite sides and softly cinch into a symmetric bow.
- **Loop and tuck**: a blue loop rises first, then a red thread wraps around it and tucks into a more asymmetric bow.

Each transparent 160 × 120 artboard has a `Follow` state machine and a 1.2-second one-shot animation that holds its final pose. The comparison artboard repeats every two seconds. Labels and mock button text are added only to the presentation GIF, not the production artboards.

```sh
python3 design/rive/follow-knot-variations/generate_scene.py
rive design/rive/follow-knot-variations --verify
rive inspect design/rive/follow-knot-variations --summary
python3 design/rive/follow-knot-variations/render_previews.py
rive design/rive/follow-knot-variations
```

These are standalone design options; app integration follows selection.
