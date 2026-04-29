## Architecture Decision Record
- Status: Accepted
- Aspect: Product surface / AppView / auth
- Date: 2026-04-29
- Decision: v1 ships login-required only. An unauthenticated read surface (public feed, public post permalinks) is a planned future addition — either late in v1 or post-v1 — not a launch requirement.

### Why I needed to decide this

Craftsky is built on the AT Protocol, where data on PDSes is fundamentally public and most ATProto AppViews (Bluesky's official app, deer.social, etc.) expose at least some content to logged-out viewers. The question for Craftsky is whether v1 should follow that norm — letting anyone hit the site/app and browse a craft feed without first completing a Bluesky/PDS OAuth login — or whether v1 should be strictly login-walled.

This needs a deliberate answer because unauthenticated access is not a small feature flag. It pulls in real backend surface area: a separate public read pipeline, a label-aware visibility layer, CDN/edge caching strategy, IP-level rate limiting, and a moderation policy that holds up without a viewer identity. Wiring those in late is doable but not free, and getting them wrong (especially the moderation policy) creates user-trust problems that are expensive to walk back.

The forcing question: does the upside of anonymous access — discovery, link-shareability, lower first-run friction — justify the AppView-style infrastructure investment for v1, *given* that v1 is a Flutter client with no server-side rendering and therefore no SEO benefit available?

### Options I considered

**Option 1: Login-required v1, anonymous access added later — CHOSEN**

v1 requires Bluesky/PDS OAuth before the user sees any content. The full app surface (feed, search, profiles, projects, comments, notifications) is behind login. Public unauthenticated reads are explicitly deferred to a later milestone, with a sketch of the eventual shape captured here so we know what we're deferring.

- Pro: Smallest v1 backend. No separate public API tier, no edge cache strategy for anonymous traffic, no IP rate-limiting layer, no anonymous-viewer moderation policy to defend.
- Pro: One auth state to test and reason about. Every page assumes a viewer DID; visibility decisions are uniform.
- Pro: Matches the current client reality. Flutter web is client-rendered SPA — crawlers see an empty shell, social unfurlers get no useful Open Graph tags, shared links don't preview. The biggest argument for anonymous access (organic discovery + SEO) doesn't apply to v1's rendering model.
- Pro: Defers a moderation-policy decision that is hard to make well in the abstract. Whose labelers apply to anonymous viewers? What's the default sensitivity threshold? Easier to answer once we have real users and real content.
- Pro: Buys time to instrument the OAuth funnel and learn whether first-run friction is actually a problem before paying to fix it.
- Con: Higher first-run friction. Curious users who hear about Craftsky on Bluesky must complete OAuth before seeing anything. Some will bounce.
- Con: No public surface for marketing screenshots, podcast demos, or press dossiers to point at — those have to be staged manually or rely on a separate marketing site.
- Con: Slightly out of step with ATProto ecosystem norms, which generally lean toward public-by-default reads.
- Con: Some deferred work is inevitable later. We pay it then instead of now.

**Option 2: Public read, authed write from day one**

Mirror the Bluesky/most-AppView default: anyone can browse a feed, search, and view individual posts without logging in. Logging in is required only to post, comment, like, follow, or see personalised views.

- Pro: Aligns with ATProto ecosystem norms.
- Pro: Public permalinks become useful immediately for sharing, even without SEO — a Bluesky user can drop a Craftsky link in a thread and other users can click through without an OAuth wall.
- Pro: Lower friction first-run experience. The app is its own demo.
- Pro: Forces us to build the public read tier — and the visibility decision function, the cache strategy, the moderation policy — from day one, rather than retrofitting later.
- Con: Real engineering investment up front: separate public endpoints, a hydration layer that takes a `viewer = anonymous` context, edge caching that doesn't vary on cookie, IP-level rate limiting, scraper/bot management, an admin-side takedown path, and the operational habit of treating public traffic as a distinct class.
- Con: We commit to a logged-out moderation policy now. At minimum: honor `!no-unauthenticated`, honor self-labels, apply Bluesky's official moderation labeler, apply a conservative default sensitivity policy for anonymous viewers, redact embeds/quotes/replies whose author is hidden. All real, all needs to be right on day one.
- Con: With Flutter as the only v1 client, the SEO/unfurl upside is unrealised. We pay the full cost of public reads without the discovery dividend that justifies it for SSR-based AppViews.
- Con: Public surface = scraping surface. Adversarial traffic (competitors enumerating the maker directory, spam reconnaissance) lands on us from day one and we have to be ready.

Not chosen for v1.

**Option 3: Public read of a curated subset only (e.g. an editorially-picked "explore" page)**

Ship a small, curated public surface — maybe a hand-picked "featured projects" page and individual post permalinks for those features — without exposing the full feed or search to anonymous viewers.

- Pro: Gets a marketing-friendly public surface without a full anonymous AppView build.
- Pro: Curation sidesteps most of the moderation-at-scale problem; humans pick what's public.
- Con: Still requires the public read pipeline, the visibility decision function, and the edge-cache strategy. Most of the infrastructure cost lands the same way.
- Con: Curation is editorial work we don't have capacity for at v1.
- Con: It's a half-measure that's harder to defend than either extreme. Either anonymous reads matter enough to do properly (Option 2) or they can wait (Option 1).

Not chosen.

### What I decided

**Option 1.** v1 is login-required. The full app sits behind Bluesky/PDS OAuth. Unauthenticated access is a planned later addition and is not blocked or vetoed — it is deferred.

**Why:**

- The strongest argument for anonymous access in a typical ATProto AppView is organic discovery via SEO and social-link unfurls. Flutter client-rendered web removes that argument from v1's calculus. We would be paying the cost of public reads without the discovery dividend.
- The moderation policy for anonymous viewers (default sensitivity, label set, embed cascading, takedown path) is hard to design well before we have real content and real users. Deferring lets us design it against reality, not abstraction.
- The OAuth-friction concern is empirical, not theoretical. Whether it actually hurts trial-to-activation is something we can only know with launched product and instrumentation. If it does, that becomes the trigger to build the anonymous tier; if it doesn't, we've avoided a real chunk of work.
- v1 has finite scope. The most defensible cuts are the ones whose absence is easy to recover from. Adding anonymous reads later is purely additive — it doesn't require redesigning the authed path. So this is a low-regret deferral.

### Trade-offs

**Good:**
- Smaller v1 backend. No separate `/api/public/*` router, no anonymous hydration path, no second cache tier, no IP rate-limit layer designed specifically for unauthenticated traffic.
- One viewer state to reason about everywhere. Pages assume a DID; the visibility decision function only needs the authed branch on day one.
- Deferring the logged-out moderation policy means we don't have to commit, in advance and in public, to which labelers apply, what the default sensitivity is, and how `!no-unauthenticated` interacts with embeds — all in the abstract.
- We retain optionality. The eventual shape (public feed + craft toggle + post permalinks, behind a separate API namespace and a strict label policy) is sketched below, so the deferral is informed, not blind.
- Engineering effort that would have gone into the public tier goes into core v1 instead: OAuth polish, the authed feed, the project-post flow, moderation tooling for logged-in users.

**Bad:**
- First-run experience is rougher. Anyone who shows up curious must complete OAuth before seeing anything. Some non-zero fraction will bounce at the wall.
- No public surface to point at for press, podcasts, or shared Bluesky links during the v1 window. We rely on screenshots, video, or a separate marketing site.
- We are slightly out of step with ATProto ecosystem norms, where public reads are the default. We should be ready to explain the choice if asked.
- Some moderation infrastructure will be built later, under time pressure, when the anonymous tier ships — rather than designed leisurely up front. We mitigate by building the visibility decision function (see Notes) as if anonymous viewers existed, even though none do yet.
- If the anonymous tier ends up being a late v1 addition rather than a post-v1 milestone, we incur the cost during the launch crunch. The decision to slot it into v1 versus push it past is left open here.

### Notes

#### What "later" should look like, when we get there

Captured now so the deferral is informed and the eventual build doesn't start from scratch:

- **Scope.** A simplified public feed (with a craft toggle and search) plus individual post permalinks. *Not* public profiles, *not* anonymous comment trees, *not* anonymous personalisation, *not* anonymous aggregate stats. The smallest surface that delivers the link-shareability and first-run-preview upside.
- **Separate endpoint family.** Public reads live under their own API namespace (e.g. `/api/public/*`), distinct from the authed API. Different cache policy (CDN-cacheable, no `Vary: Cookie`), different response shape (no viewer-relative fields like `viewer.like` or `viewer.following`), different rate-limit class. The `!no-unauthenticated` filter and the anonymous-default label policy apply as middleware on this router so visibility is a property of the endpoint family, not a per-call-site check.
- **Moderation policy for anonymous viewers (working defaults, to be revisited):**
  - Honor `!no-unauthenticated` self-labels at profile and record level. Cascade: redact quotes/replies/embeds whose author carries the label.
  - Apply Bluesky's official moderation labeler and any Craftsky-run labeler. Do *not* apply arbitrary user-subscribed labelers (anonymous viewers haven't subscribed to anything).
  - Default to hiding adult/sensitive labels for anonymous viewers, with a "sign in to adjust" affordance.
  - Deleted records: 410 Gone + `noindex`. `!no-unauthenticated` records on a permalink: 200 with a "sign in to view" page + `noindex`. Never silently 404 a valid URL.
- **Permalink format.** Decide on a stable URL shape for individual posts now-ish, even though the public renderer ships later, so authed deep links don't need to change when the public tier comes online.
- **Trigger to actually build it.** Either (a) instrumentation of the v1 OAuth funnel shows meaningful drop-off at the login step, (b) we ship a server-rendered web build that can benefit from SEO, or (c) marketing/press needs a public surface and a separate static site isn't sufficient. Absence of any of these = comfortable to keep deferring.

#### Build-now-anyway: the visibility decision function

Even though no anonymous viewer hits v1, build the visibility decision as a single function — roughly `visibilityFor(viewer, subject, labels) -> { action: 'show'|'redact'|'hide', reason }` — and route every render path through it. v1 only needs the authed branch (mute lists, blocks, self-labels, viewer's label preferences). When the anonymous tier lands later, it becomes one new branch (`viewer === anonymous → strict policy`) in one place rather than a retrofit through the codebase. This is the highest-leverage piece of moderation code we'll write and is approximately free to design correctly now.

Honoring `!no-unauthenticated` should be wired into this function from day one as well. Even with no public web surface in v1, internal contexts where it could apply (server-rendered email digests, future embed widgets, partner APIs) may show up before the full anonymous tier does.

#### Out of scope for this ADR

- The exact OAuth flow shape and login UX. Covered separately by the auth implementation work.
- Whether Craftsky runs its own labeler. Likely yes eventually, but not a v1 requirement and not gated on this decision.
- The marketing-site question (separate static site vs. eventual SSR Craftsky web). Affects when the anonymous-tier trigger fires, but not whether v1 ships login-required.

#### Related references

- `atproto-craft-social-app-reference.md` for the broader AppView shape Craftsky is building toward.
- `docs/roadmap.md` for v1 scope and post-v1 milestones — the anonymous tier should be added to the roadmap as a deferred item with the trigger conditions above.
- ADR 001 (`001-post-lexicon-project-extensibility.md`) for the "build the data model so future additions are additive" pattern this ADR mirrors at the product-surface layer.
