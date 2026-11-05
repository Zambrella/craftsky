# Carousel image preloading

Inline and fullscreen carousels share `web-client/src/lib/adjacent-image-preloader.ts`.
The action observes the actual displayed image, waits for its successful decode,
then warms only indices `current - 1` and `current + 1`. It uses the same validated
URL as the displayed image, so browser caching/decoding can be reused on selection.
Inline feed views use thumbnails; detail/fullscreen views use their existing
full-size URL selection and fallbacks.

Inline pages attach image sources for the selected image and images already warmed
by a selected image's preload window. Other pages retain their existing layout,
links and controls without allowing browser lazy-loading heuristics to fetch the
whole album ahead of the current decode. Warmed pages remain available during
carousel transitions; selecting another page updates the adjacent window.

Speculative images use low fetch priority and asynchronous decoding. They do not
observe their own neighbours, so warming never recursively downloads the album.
Speculative failures remain silent, while existing visible-image failure handling
is retained. A cached visible image triggers the same decode path. Index, URL-list
and active-image changes replace the watcher; disposal removes it. Revision guards
prevent an obsolete or disposed image's delayed decode from launching new work.

Browser coverage verifies delayed current decoding, thumbnail/full-size URL choice,
first/last boundaries, middle-of-album initial selection, moving windows without
cascading, stale completions after selection/navigation, and speculative versus
visible errors. Existing keyboard, swipe, zoom, captions and accessibility checks
remain in the carousel/social suites.
