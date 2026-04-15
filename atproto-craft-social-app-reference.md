# AT Protocol – Crafting Social App Reference

A reference guide for building a crafting-focused social platform on the AT Protocol, compiled from research and architectural discussions.

---

## AT Protocol Overview

### Core Architecture

The AT Protocol (atproto) is a federated protocol for decentralised social applications. It uses a modular microservice architecture with four key components:

- **PDS (Personal Data Server):** Each user's "digital home." Stores their data repository (posts, follows, likes) and blobs (images, media). Handles auth, identity, and signing.
- **Relay:** Connects to all PDS instances across the network and aggregates their events into a single real-time stream (the "firehose"). The main public Relay is at `bsky.network`.
- **Firehose:** The real-time WebSocket stream of events output by the Relay. Downstream consumers subscribe to this to receive all network activity.
- **App View:** A backend service that subscribes to the firehose, filters for relevant records (based on a specific lexicon), indexes them, and serves an API to client applications. This is what you build.

### Data Flow

```
User creates post → Written to their PDS → PDS emits event to Relay →
Relay broadcasts via firehose → Your App View catches & indexes it →
Your frontend queries the App View to display it
```

- **Writes** go through the user's PDS.
- **Reads** come from your App View.
- The Relay is the aggregation layer that means your App View only needs one WebSocket connection, not connections to every PDS.

### Identity

- Users are identified by a **DID** (Decentralised Identifier), which is independent of any server.
- A DID points to exactly **one PDS at a time** — no splitting data across multiple PDS instances simultaneously.
- Users can **migrate** their DID (and all their data) between PDS hosts. Identity, followers, and social graph carry over.
- **Handles** (e.g. `alice.bsky.social` or `alice.customdomain.com`) resolve to DIDs, which resolve to PDS endpoints.

### Lexicons

- Lexicons are schemas that define record types (like `app.bsky.feed.post` for Bluesky posts).
- For a custom app, you define your own lexicon namespace (e.g. `com.craftapp.craft.post`) with fields specific to your domain.
- The PDS stores records against any lexicon — it doesn't need to understand your custom schema.
- Once users create records with your lexicon, changing it is painful — existing data on PDS instances can't be easily migrated. **Design carefully upfront.**

---

## Media Storage

- **Images and media** are stored as "blobs" on the user's PDS.
- The PDS has a generic blob upload limit of ~50 MB.
- Bluesky's `app.bsky` lexicon limits images to 1 MB each — but this is a lexicon-level constraint, not a PDS constraint. Your custom lexicon can use the full 50 MB.
- The Relay distributes records but doesn't serve blobs — consumers fetch media directly from the originating PDS.
- **Video on Bluesky** specifically goes through their transcoding service and CDN, so the actual served video lives outside the PDS. This is a Bluesky app-level decision, not a protocol constraint.

---

## Data Visibility & Privacy

### Current State

- **All data on a PDS is currently public.** Everything in the repo — posts, records, blobs — is accessible to anyone.
- Blocks stored on the PDS are also public knowledge (anyone reading the repo can see them).

### Private/Permissioned Data (Coming Soon)

- A Private Data Working Group is actively working on private lexicon records with access control applied by the PDS.
- Private content would not hit the Relay — instead, the PDS would send metadata notifications for interested App Views to fetch directly.
- "Permissioned Data" is a major focus on the Spring 2026 atproto roadmap, expected through summer 2026.

### Practical Split for Now

| Data Type | Where to Store | Why |
|---|---|---|
| Published posts, public profile, follows, blocks | PDS | Portable, user-owned, public is fine |
| Drafts, private collections, wishlists, sensitive data | Your backend | Not yet supported privately on PDS |
| UI preferences, theme, layout | Client-side (local storage / shared prefs) | App-specific, no server needed |

---

## Authentication

### OAuth Flow (with Token Mediating Backend)

The recommended approach for a Flutter app is to use your backend as a **Token Mediating Backend (TMB)** — a confidential client that gets longer-lived tokens.

1. User enters their handle in your Flutter app
2. App resolves handle → DID → PDS endpoint
3. App fetches PDS OAuth metadata from `/.well-known/oauth-authorization-server`
4. App sends a Pushed Authorization Request (PAR) to the PDS with client_id, PKCE challenge, and scopes
5. PDS returns a `request_uri`
6. App opens browser/webview to PDS authorization endpoint
7. User authenticates on their PDS and approves your app
8. PDS redirects back with an authorization code
9. **Flutter app sends the auth code to your backend**
10. **Your backend exchanges it with the PDS, gets access + refresh tokens**
11. **Backend creates a session and returns a session token to the Flutter app**
12. Flutter app stores session token locally

### Two Separate Auth Layers

- **atproto OAuth layer:** Between your backend and the PDS (access tokens, refresh tokens, DPoP).
- **Your session layer:** Between the Flutter app and your backend (your own session token). These are independent.

### Key Details

- **DPoP (Demonstrating Proof of Possession):** Required by atproto OAuth. Binds tokens to specific client instances. Each request includes a DPoP proof.
- **Access tokens** are often signed JWTs — can be validated locally by checking signature, expiry, and claims.
- **Client metadata JSON** must be hosted as a static file at a public URL (e.g. `https://craftapp.xyz/client-metadata.json`). The PDS fetches this during OAuth.
- **Token refresh** is handled transparently by your backend. The Flutter app never deals with PDS tokens directly.

### Login UX

- For most users: just enter handle + password. Handle resolution automatically finds their PDS.
- For users on custom PDS: handle resolution handles this too — no manual config needed in most cases.
- Power users could have a manual PDS override option, but it's rarely needed.

---

## User Onboarding

### Creating Accounts on Bluesky's PDS

- From the user's perspective: normal sign-up (email, password, handle). They don't need to know what a PDS is.
- From the developer's perspective: call `com.atproto.server.createAccount` against Bluesky's PDS entryway.
- Users get a handle like `username.bsky.social` by default; can bring their own domain later.

### Alternative: Run Your Own PDS

- Create accounts on your own PDS for full control.
- Feasible on a VPS with Caddy.
- Users can always migrate away if they choose.

---

## Moderation, Blocking & Muting

### Blocks

- Store as records on the user's PDS (e.g. `com.craftapp.graph.block`).
- Your App View indexes them from the firehose.
- On feed requests, the App View filters out blocked users server-side — no extra data needed from the client.
- **Blocks are public** (anyone reading the repo can see them). Generally acceptable since the blocked person can tell anyway.
- Slight delay between block creation and indexing (~1-2 seconds). Optimistically filter client-side during the gap.

### Mutes

- Store on your App View backend (Postgres) or client-side — **not on the PDS**, because mutes should be private.
- Keyed by DID in your database.

### Content Moderation

- **Your App View decides what gets served.** You can't delete data from someone's PDS, but you can stop surfacing it.
- **Reporting system:** Users flag content → stored in your DB → reviewed by you or community moderators.
- **Labelling system:** atproto has built-in labelling where labellers tag content (NSFW, spam, etc.) and App Views subscribe to trusted labellers. You could run your own labeller.
- For a crafting community, moderation needs are lighter — mostly spam and off-topic content rather than heavy abuse.

---

## Push Notifications

- Only your App View knows when notification-worthy events happen (likes, comments, follows).
- Your backend detects the event from the firehose → sends push via FCM/APNs.
- Requires storing a DID → device push token mapping on your backend.

---

## Feed Construction

### Basic Approach

```sql
SELECT p.*
FROM posts p
INNER JOIN follows f ON f.subject_did = p.did
WHERE f.follower_did = :current_user_did
  AND p.did NOT IN (SELECT subject_did FROM blocks WHERE did = :current_user_did)
  AND p.did NOT IN (SELECT subject_did FROM mutes WHERE did = :current_user_did)
ORDER BY p.created_at DESC
LIMIT 50
```

### Optimisation Path (When Needed)

- **Materialised feed tables:** Pre-compute feeds at write time (fan-out-on-write) rather than computing joins on every read.
- **Caching:** Redis or in-memory cache for recent feed data.
- **Cursor-based pagination:** Use `WHERE p.created_at < :last_seen_timestamp` instead of OFFSET.
- **Indexing:** Ensure proper indexes on `posts.did`, `posts.created_at`, `follows.follower_did`, `follows.subject_did`.

A crafting community has naturally lower posting volume than a general social network — the basic join approach handles thousands of users easily.

---

## Request Flows

### Reading a Feed

```
Flutter app → session token → Your backend validates session →
Queries Postgres index → Returns results
(No PDS involvement)
```

### Creating a Post

```
Flutter app → session token + post data → Your backend validates session →
Looks up PDS tokens from sessions table → Writes record to user's PDS with DPoP proof →
PDS emits to Relay → Your firehose consumer indexes it into Postgres
```

---

## App View Database Schema

```sql
-- Core index tables (populated from firehose)

posts (
  id              SERIAL PRIMARY KEY,
  did             TEXT NOT NULL,        -- author's DID
  rkey            TEXT NOT NULL,        -- record key from PDS
  uri             TEXT NOT NULL,        -- at:// URI
  cid             TEXT NOT NULL,        -- content hash
  text            TEXT,
  project_type    TEXT,                 -- sewing, knitting, quilting etc
  techniques      JSONB,               -- array of techniques
  materials       JSONB,               -- array of materials
  status          TEXT,                 -- wip, finished
  media_refs      JSONB,               -- blob references
  created_at      TIMESTAMPTZ,
  indexed_at      TIMESTAMPTZ DEFAULT NOW()
);

follows (
  id              SERIAL PRIMARY KEY,
  follower_did    TEXT NOT NULL,
  subject_did     TEXT NOT NULL,
  uri             TEXT NOT NULL,
  created_at      TIMESTAMPTZ,
  indexed_at      TIMESTAMPTZ DEFAULT NOW()
);

likes (
  id              SERIAL PRIMARY KEY,
  did             TEXT NOT NULL,
  subject_uri     TEXT NOT NULL,        -- post being liked
  uri             TEXT NOT NULL,
  created_at      TIMESTAMPTZ,
  indexed_at      TIMESTAMPTZ DEFAULT NOW()
);

blocks (
  id              SERIAL PRIMARY KEY,
  did             TEXT NOT NULL,
  subject_did     TEXT NOT NULL,
  uri             TEXT NOT NULL,
  created_at      TIMESTAMPTZ,
  indexed_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Auth & session tables

sessions (
  id              SERIAL PRIMARY KEY,
  did             TEXT NOT NULL,
  session_token   TEXT NOT NULL,        -- what Flutter app holds
  pds_endpoint    TEXT NOT NULL,        -- which PDS this user is on
  access_token    TEXT NOT NULL,        -- PDS OAuth access token
  refresh_token   TEXT NOT NULL,        -- PDS OAuth refresh token
  dpop_key        TEXT NOT NULL,        -- keypair for DPoP proofs
  scopes          TEXT,
  expires_at      TIMESTAMPTZ,         -- access token expiry
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Server-side user state

push_tokens (
  id              SERIAL PRIMARY KEY,
  did             TEXT NOT NULL,
  device_token    TEXT NOT NULL,        -- FCM/APNs token
  platform        TEXT NOT NULL,        -- ios, android, web
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

mutes (
  id              SERIAL PRIMARY KEY,
  did             TEXT NOT NULL,
  subject_did     TEXT NOT NULL,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Moderation

reports (
  id              SERIAL PRIMARY KEY,
  reporter_did    TEXT NOT NULL,
  subject_uri     TEXT NOT NULL,        -- post or user being reported
  reason          TEXT,
  status          TEXT DEFAULT 'pending', -- pending, reviewed, actioned
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  reviewed_at     TIMESTAMPTZ
);

moderation_actions (
  id              SERIAL PRIMARY KEY,
  subject_did     TEXT NOT NULL,
  action          TEXT NOT NULL,        -- warn, restrict, remove_content
  reason          TEXT,
  created_by      TEXT,                 -- moderator
  created_at      TIMESTAMPTZ DEFAULT NOW(),
  expires_at      TIMESTAMPTZ
);
```

---

## Token Storage Summary

| Token | Where | Who Uses It |
|---|---|---|
| PDS access + refresh tokens | `sessions` table in Postgres | Your backend only |
| DPoP keypair | `sessions` table in Postgres | Your backend only |
| Your app's session token | Flutter device (secure storage) | Flutter app → your backend |
| Client metadata JSON | Static file at public URL | PDS fetches during OAuth |

---

## Tech Stack (Decided)

### Frontend: Flutter + Dart atproto SDK

- **atproto.dart** SDK ([atprotodart.com](https://atprotodart.com)) — production-ready, used by SkyFeed, deck.blue, and others.
  - `atproto` package — core `com.atproto.*` operations (repo management, identity, sync, firehose)
  - `bluesky` package — `app.bsky.*` endpoints (Bluesky-specific, may not be needed for custom lexicon)
  - `bluesky_text` package — text parsing, facets, mentions, links
  - Supports both session-based and OAuth authentication
- For a custom crafting app, the `atproto` package is the main one — use `createRecord` with your own collection namespace.
- GitHub: [github.com/myConsciousness/atproto.dart](https://github.com/myConsciousness/atproto.dart)

### Backend/App View: Go

Go was chosen over Serverpod (Dart) primarily for **open source contributor accessibility**:

- **Contributor pool:** Go has a massive developer community. Serverpod's is tiny. The odds of a contributor knowing Go are dramatically higher.
- **Ecosystem maturity:** Go tooling for this workload is battle-tested — WebSocket consumption, Postgres (pgx), HTTP routing (chi/echo/stdlib), structured logging (slog), migrations (goose/golang-migrate). Years of Stack Overflow answers and blog posts.
- **Simplicity of onboarding:** Go is famously readable. New contributors can clone the repo and understand the architecture quickly. Serverpod has its own conventions and code generation that would need learning.
- **atproto ecosystem alignment:** The Go `indigo` SDK and much community tooling is in Go. Contributors interested in atproto are more likely to be Go developers.
- **Deployment:** Compiles to a single static binary. Docker image ends up ~10-20MB.

**Trade-off:** Two languages (Dart + Go) instead of one. But for open source, frontend and backend contributors are often different people anyway.

#### Go Libraries & Tools

| Purpose | Library | Notes |
|---|---|---|
| **atproto SDK** | `github.com/bluesky-social/indigo` | Official Bluesky Go SDK. Firehose, repo operations, XRPC. [GitHub](https://github.com/bluesky-social/indigo) |
| **OAuth** | `indigo/atproto/auth/oauth` | Built into Indigo. Handles DPoP, token refresh, session management. [Tutorial](https://atproto.com/guides/go-oauth-cli-tutorial) |
| **OAuth (alternative)** | `github.com/haileyok/atproto-oauth-golang` | Community library with more manual control, custom XRPC client. [GitHub](https://github.com/haileyok/atproto-oauth-golang) |
| **Postgres** | `pgx` | High-performance Postgres driver |
| **SQL → Go codegen** | `sqlc` | Write SQL, generates type-safe Go code. Explicit, optimisable queries. |
| **Migrations** | `golang-migrate` or `goose` | File-based, well understood |
| **HTTP routing** | `chi` or stdlib `net/http` (Go 1.22+) | Lightweight, well-known |
| **Logging** | `slog` (standard library) | Built-in, no dependency |
| **Config** | `envconfig` or `viper` | Environment variable based |

**Important:** Generic OAuth libraries won't work with atproto's OAuth profile due to DPoP, dynamic server discovery, and client metadata documents. Use the atproto-specific libraries above.

#### Project Structure

```
craftapp/
  cmd/
    appview/          -- main entrypoint
  internal/
    firehose/         -- relay subscription & filtering
    index/            -- writing to postgres
    api/              -- HTTP handlers
    auth/             -- OAuth / session management
    models/           -- generated from sqlc
  migrations/         -- SQL migration files
  queries/            -- SQL files for sqlc
  lexicon/            -- your lexicon schema definitions
  docker-compose.yml
  Dockerfile
  Makefile
  CONTRIBUTING.md
```

A `Makefile` with targets like `make dev`, `make migrate`, `make generate`, `make test` lowers the barrier for contributors. Someone clones the repo, runs `make dev`, and they're up.

### Database: Postgres

- Used natively with `pgx` + `sqlc`.
- Full-text search for craft discovery without a separate search service.
- `sqlc` means contributors can read the SQL directly — explicit and transparent.

### Infrastructure: Hetzner VPS + Docker Compose

**Hetzner** is recommended over GCP for this use case — dramatically cheaper for equivalent workload (€4-6/month vs 5-10x on GCP). A CX22 (2 vCPU, 4GB RAM) handles the workload comfortably. GCP's Cloud Run has cold start issues with persistent WebSocket connections to the firehose. The crafting community would need tens of thousands of active users before scaling beyond a single box.

#### Docker Compose (Production)

```yaml
services:
  appview:
    build: .
    restart: always
    depends_on:
      - postgres
    environment:
      - DATABASE_URL=postgres://craft:secret@postgres:5432/craftapp
      - FIREHOSE_URL=wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos
    ports:
      - "8080:8080"

  postgres:
    image: postgres:16
    restart: always
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=craft
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=craftapp

volumes:
  pgdata:
```

Caddy sits on the host, proxying `craftapp.yourdomain.com` to `localhost:8080`. Multi-stage Dockerfile: build in Go image, copy binary into scratch/alpine.

#### Dev Environment

- Run Postgres in Docker: `docker compose up postgres`
- Run Go binary natively with hot-reload: `air` or `go run .`
- Flutter app points at `localhost:8080`
- For firehose in dev: connect to real firehose (little activity for your lexicon early on) or write a small mock that generates fake events in your lexicon format.

### Push Notifications: FCM

### Communication: JSON over HTTP (REST)

Protobuf/gRPC was considered but **JSON/REST is the pragmatic choice** for an open source project:

- atproto itself is entirely JSON-based (XRPC uses JSON over HTTP). Protobuf would mean bridging two serialisation worlds.
- The Dart gRPC ecosystem is less mature than REST/JSON tooling in Flutter.
- The API surface is small and payloads aren't complex enough to warrant protobuf overhead.
- JSON is easier for contributors to debug and understand.
- If type safety is needed later, OpenAPI/Swagger can generate Dart client code from the Go API.
- **Connect** (from the Buf team) is a middle ground option — protobuf schema benefits with JSON fallback.

### Considered Alternatives

| Stack | Pros | Cons | Why Not |
|---|---|---|---|
| **Serverpod (Dart)** | One language, built-in ORM/logging | Tiny community, immature | Poor for open source contributor access |
| **TypeScript/Node** | Most mature atproto SDK, biggest community | Three mental contexts (Dart + TS + SQL) | Go better aligned with atproto ecosystem |
| **Rust** | Maximum performance, minimal resources | Steep learning curve | Too slow for solo dev + contributor onboarding |
| **Python (FastAPI)** | Fastest to prototype, Flask OAuth demo | Performance concerns at scale | Go is better long-term |
| **GCP** | Managed services, Firebase integration | 5-10x cost, Cloud Run cold starts | Hetzner better value for this workload |

---

## Key Considerations

### Lexicon Design
The most important early decision. Once people create records, you can't easily migrate the schema. Think carefully about fields: project type, materials, techniques, difficulty, pattern links, WIP vs finished status, etc.

### Community Bootstrapping
The tech is the easier part. Getting crafters to sign up when they could just post on Instagram is the real challenge. Stash Hub's existing user base is the seed community. A "share your project" feature in Stash Hub that posts to the crafting network is a natural growth flywheel.

### Discoverability
Consider a public web frontend so craft projects are indexable by search engines. People searching "how to sew a French seam" finding your platform is a major growth channel.

### Monetisation
You can't gate the data (it's public on the PDS). Your value is the App View — curation, search, feeds, community features. Freemium around premium features (advanced search, analytics, pattern tools) or tie into Stash Hub subscriptions.

### Legal
Hosting an index and doing moderation puts you in content platform territory. Understand obligations under UK law and the Online Safety Act.

### Why atproto Reduces Complexity (for a solo dev)
- Auth and identity handled by the PDS
- User data storage handled by the PDS
- Real-time event streaming via the Relay
- Account portability built in
- No media storage bills, reduced GDPR burden
- If your App View dies, rebuild from the Relay — no user data lost
- Your server is essentially a read-only index + thin auth/session layer

### Start Simple
Posts, a chronological feed, and basic search. Validate whether crafters want this before building algorithmic feeds, notifications, labelling, and rich profiles.

---

## Not P2P

Despite being "decentralised," atproto is federated, not peer-to-peer. Data flows through centralised-ish infrastructure (PDS → Relay → App View → Client). The decentralisation is about **portability and choice** — move your identity between PDS hosts, anyone can run a Relay or App View, no single company controls the identity layer. More like email than BitTorrent.

For actual P2P social protocols, look at Scuttlebutt (SSB) or Nostr — but they lack the rich indexing and feed generation atproto enables.

---

## Useful Links

### AT Protocol

- [AT Protocol Specification](https://atproto.com)
- [AT Protocol Spring 2026 Roadmap](https://atproto.com/blog/2026-spring-roadmap)
- [AT Protocol Community Wiki](https://atproto.wiki)
- [Bluesky Developer Docs](https://docs.bsky.app)
- [OAuth Specification for atproto](https://atproto.com/specs/oauth)
- [OAuth Client Implementation Guide](https://docs.bsky.app/docs/advanced-guides/oauth-client)
- [OAuth for AT Protocol Blog Post](https://docs.bsky.app/blog/oauth-atproto)
- [Private Data Working Group](https://atproto.wiki/en/working-groups/private-data)
- [Bluesky Rate Limits](https://docs.bsky.app/docs/advanced-guides/rate-limits)
- [Posts & Blobs Guide](https://docs.bsky.app/docs/advanced-guides/posts)
- [PDS Self-Hosting Repo](https://github.com/bluesky-social/pds)

### Go / Backend

- [Indigo – Bluesky's Go SDK](https://github.com/bluesky-social/indigo)
- [Go OAuth CLI Tutorial (official)](https://atproto.com/guides/go-oauth-cli-tutorial)
- [atproto-oauth-golang (community library)](https://github.com/haileyok/atproto-oauth-golang)
- [Go OAuth Example (potproject)](https://pkg.go.dev/github.com/potproject/atproto-oauth2-go-example)
- [sqlc – SQL to Go codegen](https://sqlc.dev)
- [pgx – Postgres driver for Go](https://github.com/jackc/pgx)
- [chi – HTTP router](https://github.com/go-chi/chi)
- [air – Go hot-reload for dev](https://github.com/air-verse/air)

### Flutter / Dart

- [atproto.dart SDK](https://atprotodart.com)
- [atproto.dart GitHub](https://github.com/myConsciousness/atproto.dart)
- [atproto Dart package (pub.dev)](https://pub.dev/packages/atproto)
- [bluesky Dart package (pub.dev)](https://pub.dev/packages/bluesky)

### Community

- [AT Protocol PDS Admins Discord](https://discord.gg/atproto-pds) — for PDS hosting discussion
- [ATmosphereConf](https://atproto.science/events/atmosphere2026/) — annual atproto conference
