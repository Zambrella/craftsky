# Profile background choices and pinned preview

Removed Dither, Grid, Cross stitch, Scallops, Diagonal weave and Crosshatch
from Flutter and web profile customisation and AppView's accepted catalogue.
All 13 animated backgrounds, five craft backgrounds, None and supported
profile colours remain available.

Previously stored texture keys resolve to None while retaining a supported
colour. The native header and web texture helper also safely render these keys
without requesting removed assets. Saving a retired key is rejected. Existing
private database values are not rewritten; no migration or Lexicon change is
required. Removed the unused PNG assets, native rendering branch and labels.

## Pinned customisation preview

The mobile preview now sits above the scrolling options, below the app bar.
The web preview sticks below the shared app bar while the form scrolls. Both
retain the live colour, background and avatar preview and existing Save flow.
The mobile widget test checks unchanged preview geometry after scrolling to
background choices and Save. Web browser tests exercise desktop (1280×480) and
phone (375×640) layouts, checking placement below the title and live selection.

## Accent-derived pigment palettes

Aurora, Colour pools, Sunset, Pigment, Marble and Woven now derive their pigment
colours from the selected profile accent instead of fixed RGB constants. Each
uses nearby hue offsets and retains the accent's saturation, keeping muted
colours muted. Near-black Ink produces neutral pigments. Per-style brightness
limits preserve the dark overlay treatment and existing light navigation.
Shapes, timing, alpha, grain and reduced-motion behaviour remain unchanged;
the other seven shader palettes retain their existing behaviour. Flutter and
web consume the same shader source, so this applies to both platforms.

Compiled Flutter pixel tests isolate the pigment layer from its base colour and
verify all six styles respond to Cobalt, Seafoam, Terracotta and Ink. The test
failed against the old fixed palettes before the implementation. The existing
animation, reduced-motion and navigation tests also pass (five tests total).
An authenticated browser test compiles the shared shader with software WebGL
and reads pigment pixels for the same six styles and four colours. It passes.
Flutter analysis and web check/build pass; hot restart reports no runtime errors.

## Validation

- Flutter: 52 focused model, header, provider, API, access and settings tests
  passed; focused analysis found no issues. Hot restart succeeded with no
  runtime errors after refreshing the running asset manifest.
- Web: 26 focused settings/social tests passed; Svelte check found zero errors
  and warnings; production build passed with the existing bundle-size notice.
- AppView: focused ProfileCustomisation API and route tests passed with the
  local integration database, covering catalogue rejection and read fallbacks.
- Web authenticated customisation save/reload browser test passed using the
  local mock AppView; checks confirm all six retired options are absent.

After pinning the preview, all nine customisation-page widget tests passed and
focused analysis remained clean. Web check/build passed; all three authenticated
customisation tests passed. Mobile hot reload succeeded without runtime errors.

No production configuration, deployment or stored account data was changed.
