## Architecture Decision Record
- Status: Accepted
- Aspect: Lexicon (atproto schemas), AppView API, Flutter composer
- Date: 2026-05-22
- Decision: Make post image alt text optional, with a client-side warning before posting undescribed images

### Why I needed to decide this

Image posts currently require `images[].alt` in the `social.craftsky.feed.post` lexicon, AppView create validation, and the Flutter composer. Product direction has changed: alt text should remain encouraged for accessibility, but it must not be required to publish an image post.

Because lexicon edits are load-bearing in atproto, this decision documents the intentional contract change before editing `lexicon/social/craftsky/feed/post.json`.

### Options I considered

**Option 1: Keep alt text required — not chosen**

Pros:
- Maximizes accessibility coverage for image posts.
- Preserves the existing lexicon shape.

Cons:
- Conflicts with the updated product requirement.
- Blocks posting when a user is unwilling or unable to write descriptions.

**Option 2: Make alt text optional and warn before posting undescribed images — CHOSEN**

Pros:
- Matches the updated product requirement.
- Keeps the UI nudge toward accessible posting without hard-blocking users.
- Allows records from other clients or future flows to omit `alt` cleanly.

Cons:
- New records without `images[].alt` will not validate against the previous required-alt lexicon.
- Consumers must tolerate missing or empty alt text.

### What I decided

Adopt **Option 2**: remove `alt` from `social.craftsky.feed.post#image.required`, keep the `alt` property available with the same string constraints, update AppView validation/write mapping to accept omitted or empty alt text, and update the Flutter composer to show a warning confirmation when any attached image lacks alt text.

### Notes

- This is a deliberate schema loosening for the current pre-release image-posting work. If records already exist on externally operated PDSes, clients using the older schema may reject new records that omit `alt`.
- AppView read responses should continue to expose an `alt` string, using `""` when the record omitted the field, so current Flutter rendering remains simple.
- After the schema edit, regenerate lexicon-derived Go types via `just lexgen` and commit generated outputs alongside the schema change.
