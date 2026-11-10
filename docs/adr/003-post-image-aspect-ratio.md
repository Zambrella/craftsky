## Architecture Decision Record
- Status: Accepted
- Aspect: Lexicon (atproto schemas)
- Date: 2026-05-19
- Decision: Add optional `social.craftsky.feed.post#image.aspectRatio` (`width`, `height`) as positive integers

### Why I needed to decide this

Image posting in AppView now needs to preserve and return render-ready metadata that clients already depend on in the upload-then-create flow. The requirements and acceptance tests for `2026-05-19-appview-image-blobs` require optional aspect-ratio metadata to round-trip through create → index → read. Without a lexicon field, aspect ratio supplied by clients would be dropped or require out-of-contract side channels.

Because lexicon edits are load-bearing in atproto, this needs an explicit ADR before changing `lexicon/social/craftsky/feed/post.json`.

### Options I considered

**Option 1: Add optional `#image.aspectRatio` object — CHOSEN**

- Shape: `images[].aspectRatio = { width, height }`
- `width` and `height` are required when `aspectRatio` is present.
- Both must be positive integers.

Pros:
- Additive and backward-compatible; existing records without `aspectRatio` remain valid.
- Keeps metadata colocated with each image in Craftsky’s existing top-level `images` model.
- Supports deterministic read-side rendering without inspecting image bytes.

Cons:
- Adds one more optional nested object to the post image contract.

**Option 2: Do not add lexicon field; infer dimensions server-side**

Not chosen.

Reasons:
- Conflicts with current requirement scope (`validate/pass-through`, no image-byte inspection or transforms).
- Introduces extra processing and potential mismatch between uploaded file metadata and declared intent.

### What I decided

Adopt **Option 1**: add optional `aspectRatio` to `social.craftsky.feed.post#image` with positive integer `width`/`height`.

### Notes

- This is an additive lexicon evolution and preserves compatibility with existing image records.
- After the schema edit, regenerate lexicon-derived Go types via `just lexgen` and commit generated outputs alongside the schema change.
- AppView request validation, record write mapping, index flattening, and read response mapping must all treat `aspectRatio` as optional but strictly positive when provided.
