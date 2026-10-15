# Publishing loader studies

Three editable, timeline-only Rive concepts for CraftSky. These comparison artboards are design experiments. The selected stitch orbit is prepared separately in `../publishing-indicator/` and bundled into the Flutter publishing overlay.

| Artboard | Motion | Intended use |
| --- | --- | --- |
| Stitch orbit | A red leading stitch circles a cobalt running-stitch ring; the centre arrow rises, fades, and reappears below to suggest uploading. | Closest continuation of the existing CraftSky loader. |
| Stitch wave | Five thread strokes rise and stretch in a staggered wave, with a red centre stitch. | A compact, quiet publishing-button companion. |
| Winding yarn | A striped yarn ball gently rocks and lifts while its loose red thread flexes. | A more playful publishing overlay. |

All loops run for 2.4 seconds, with no scripted logic, network access, or external assets. The warm paper background is part of the study artboards. Small-size legibility, reduced-motion behavior, success/error transitions, and Flutter runtime integration remain work for a selected direction. Loop timing is indeterminate and does not represent upload percentage.

## Preview

From the repository root:

```sh
rive design/rive/publishing-loaders
```

The default `Comparison` artboard shows all three. In the preview terminal, enter `a` to switch artboards, or launch a particular one:

```sh
rive design/rive/publishing-loaders --artboard="Winding yarn"
```

The comparison GIF and poster are in `previews/`. They contain frames rendered by the Rive CLI, with presentation labels added around the capture.

## Validate and build

```sh
rive design/rive/publishing-loaders --verify
rive inspect design/rive/publishing-loaders --summary
rive design/rive/publishing-loaders --once
rive design/rive/publishing-loaders --screenshot --advance=36
```

`--once` writes the local unsigned `.riv` to `build/publishing-loaders.riv`. This study has no scripts. No files have been pushed to a Rive account or published to the web. Editor `.rev` export and signed publishing require Rive authentication and are separate steps.

Source: `scene.rml`; project configuration: `rive.yaml`. Each concept has its own `Publishing` state machine with a default looping animation. The comparison artboard duplicates the concepts for simultaneous viewing.
