## Architecture Decision Record
- Status: Accepted
- Aspect: Post Lexicon and media policy
- Date: 2026-10-14
- Decision: Allow up to eight photos per CraftSky post

### Why I needed to decide this

CraftSky posts currently allow four photos. Crafters need more room to show their work, and the requested limit is eight. General and project posts share `social.craftsky.feed.post.images`.

### Options I considered

- Raise the existing image-array limit to eight: chosen because the app has no active users and shipped-app compatibility is not required.
- Introduce a new post NSID: unnecessary at this stage and would duplicate the post ingestion and authoring paths.
- Change only the composer: insufficient because AppView and the record schema would still reject larger posts.

### What I decided

Change `social.craftsky.feed.post.images.maxLength` from four to eight in place. Align Flutter's shared composer limit, AppView's default and maximum configurable count, local development configuration, and the Instagram importer's generated contract and selection limit.

Existing records with four or fewer images remain valid. Records with five to eight images fail validation against the old schema, so this is a deliberate compatibility change permitted by the repository's no-active-users policy. Update consumers before enabling eight-photo authoring in a release.

Per-image size, MIME, dimensions, upload authorization, and video rules remain as previously approved. Business forms retain their explicit single-image limits. Production configuration and deployment are separate maintainer operations; any explicit production `MAX_POST_IMAGES=4` override must be updated when releasing this change.
