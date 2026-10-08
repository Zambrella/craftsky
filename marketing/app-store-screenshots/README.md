# CraftSky App Store screenshots

Five English marketing slides for iPhone, iPad and Android phone, built with the app-store-screenshots skill template. Brand palette: cream paper, cobalt, warm ink, lilac and butter, with stitched accents. Copy follows https://craftsky.social/.

## Run

From this directory:

```sh
bun install
bun dev
```

Open http://localhost:3000 (or the port printed by Next). The dev script uses webpack with polling to avoid local file-watcher limits.

## Edit and export

The canonical decks are in `app-store-screenshots.json`. The editor saves changes there automatically. Select a platform and device to switch decks, then select a screen to edit its caption, emphasis text, device position or colour. Every deck uses the same five-slide sequence and copy, with native captures and layouts adapted to that device.

| Device | PNG dimensions | PNG count | Bundle |
| --- | --- | --- | --- |
| iPhone | 1320 × 2868, 1284 × 2778, 1206 × 2622, 1125 × 2436 | 20 | `exports/craftsky-iphone-en.zip` |
| iPad portrait | 2064 × 2752, 2048 × 2732 | 10 | `exports/craftsky-ipad-en.zip` |
| Android phone portrait | 1080 × 1920 | 5 | `exports/craftsky-android-en.zip` |
| Google Play feature graphic | 1024 × 500 | 1 | `exports/craftsky-feature-graphic-en.zip` |

Export bundle generates every configured size for the selected device and saves its ZIP into `exports/`. Re-extract the bundle after editing to refresh the individual PNG folders. ZIP bundles and review previews are generated locally and ignored by Git. Only final PNGs in the device/resolution/locale folders are tracked. The editor code, deck JSON, native captures and brand assets are retained so the graphics can be updated and reproduced.

Source captures are copied intact into `public/screenshots/apple/iphone/en/`, `public/screenshots/apple/ipad/en/` and `public/screenshots/android/phone/en/`. The five selected features are projects, feed, details, search and conversation. The iPad conversation capture shows a different project, with the same community headline. The Android frame matches the supplied Pixel 10 captures with a 1080 × 2424 screen, 150 px screen corner radius and a uniform bezel. It retains the camera hole already present in the native capture. All Android phones share a 620 px frame width, with extra clearance below the headline and stitch. Mac and Android tablet decks have no native captures or completed layouts.

Headlines share a consistent size and left alignment within each deck. All text sits above the devices, and every device is fully visible, including slide 3. The app icon comes from `app/assets/app-icon/app-icon-light.png` in the parent CraftSky repository. No API, app code or lexicons changed.

The Google Play feature graphic is editable under Android → Feature Graphic. Its upload file is `exports/android/feature-graphic/1024x500/en/01-feature-graphic.png`. It uses the app icon, cobalt background, cream lettering and stitch motif.
