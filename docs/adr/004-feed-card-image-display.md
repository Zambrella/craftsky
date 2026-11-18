## Architecture Decision Record
- Status: Accepted
- Aspect: Flutter app / feed image display
- Date: 2026-05-20
- Decision: Display post images in feed cards as an inline bounded-aspect carousel with count, dots, and tap-to-gallery behavior

### Why I needed to decide this

Craftsky image posting is adding render-ready image metadata to post responses, including image URLs, alt text, and optional aspect ratio. The Flutter feed needs a deliberate display policy before image rendering is implemented so feed cards do not drift into inconsistent one-off layouts.

The product tension is between image detail and image overview. Carousel-style layouts keep each image large but hide secondary images behind interaction. Grid-style layouts expose all images at once but shrink or crop the craft details users may care about: stitches, seams, yarn texture, fabric drape, in-progress construction, and fit details.

For feed cards, the chosen priority is **image detail**. The feed should let users inspect one image at a useful size without leaving the scroll context, while still making multiple images obvious and accessible.

### Options I considered

**Option 1: Inline bounded-aspect carousel — CHOSEN**

Feed cards render one large image at a time. Multi-image posts can be swiped horizontally inside the image area. A compact count indicator (for example, `1/4`) and page dots show that more images are available. Tapping the image area opens a full-screen gallery; tapping other parts of the card keeps normal post/thread navigation and action behavior.

Images use the stored aspect-ratio metadata when available. The card preserves the image aspect ratio within feed-safe minimum and maximum height bounds so most images are uncropped, layout can be reserved before load, and extreme panoramas or tall images do not dominate the feed.

Pros:
- Preserves useful image detail in the feed, which fits craft posts better than tiny thumbnails.
- Supports up to 4 images without shrinking every image into a dense grid.
- Matches a familiar social interaction model while keeping the current post card structure.
- Uses existing aspect-ratio metadata to reduce layout shift and avoid byte inspection.
- Gives users both lightweight inline browsing and an explicit path to a full-screen gallery.

Cons:
- Secondary images are hidden until the user swipes.
- Horizontal paging inside a vertical feed needs careful gesture handling.
- Feed cards become stateful: current page, indicator state, image loading, and gallery entry all need tests.
- Users may still miss later images if the count/dots treatment is too subtle.

**Option 2: Fixed collage grid**

Render all images in a tiled grid: 1 image full width, 2 split, 3 or 4 in a collage.

Pros:
- Every image is visible at a glance.
- Avoids nested scrolling inside feed cards.
- Simple mental model and common in social apps.

Cons:
- Multi-image posts shrink important craft details.
- Cropping becomes likely for mixed aspect ratios.
- Optimises scanning over image inspection, which conflicts with the chosen feed-card priority.

Not chosen.

**Option 3: Hero image plus thumbnails**

Render the first image large and show the remaining images as small thumbnails below or beside it.

Pros:
- Keeps a strong primary image while still previewing the rest.
- More informative than a pure carousel because secondary images are visible.
- Can fit Craftsky's photo-forward card style.

Cons:
- Secondary images still become small thumbnails.
- Privileges the first image heavily.
- Adds more visual clutter around the post content and actions.

Not chosen.

**Option 4: Stacked-paper cue with gallery-only browsing**

Render only the first image in the feed, with Craftsky-style offset paper/photo edges and a count badge to suggest more images. Users tap to open the gallery for the rest.

Pros:
- Compact and visually distinctive.
- Avoids tiny thumbnails and avoids nested swiping.
- Strong fit with the paper-cutout design language.

Cons:
- Does not allow inline browsing.
- Hides all secondary images until gallery open.
- Optimises brand character over feed utility.

Not chosen.

### What I decided

Adopt **Option 1** for feed cards:

- Image posts render an inline carousel in the feed card.
- The carousel shows one large image at a time.
- Multi-image posts expose horizontal swipe paging.
- Multi-image posts show both a compact image count and page dots.
- The image area tap target opens a full-screen gallery.
- Non-image areas of the card keep the normal post/thread navigation and action behavior.
- Images preserve their declared aspect ratio within feed-safe bounds.
- Aspect-ratio metadata should be used to reserve layout before image load; missing metadata falls back to a safe default frame.

### Trade-offs

**Good:**
- The feed remains photo-forward without making image details too small to inspect.
- A single layout policy covers 1 to 4 images.
- The UI can use existing image metadata instead of inspecting image bytes client-side.
- Count plus dots make multi-image state explicit without adding thumbnail clutter.
- Tap-to-gallery gives users a clear path to a larger view when the feed frame is not enough.

**Bad:**
- The feed no longer shows every image at once.
- Carousel state and gesture behavior increase implementation complexity.
- Image-area tap behavior diverges from rest-of-card tap behavior, so hit targets and tests must be explicit.
- Exact min/max height bounds still need implementation tuning for phone, tablet, and web widths.

### Notes

- This ADR is scoped to **feed cards only**. It decides that image taps open a full-screen gallery, but it does not define the gallery's complete UI, gestures, chrome, or sharing behavior.
- The carousel should use the feed image cache manager established by the Flutter image caching design once feed image call sites are implemented.
- Alt text remains part of the image contract and must be exposed through accessibility semantics for the rendered image and gallery entry point.
- The first rendered image is the first image in the post response order. The app should not reorder images for layout convenience.
- If aspect-ratio metadata is missing or invalid in older/indexed data, the feed should render a stable fallback frame rather than blocking the post.
- Implementation should test at least: one image, two images, four images, missing aspect ratio, very wide image, very tall image, horizontal swipe in a vertical feed, image-area tap opening gallery, and non-image tap opening the post/thread.
