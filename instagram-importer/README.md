# CraftSky Instagram post importer

This package builds the static application at
[`https://import.craftsky.social`](https://import.craftsky.social). It lets an
existing CraftSky member review an Instagram Data Export ZIP locally, then
publish selected historical posts directly to their PDS.

The importer is deliberately independent from the Flutter application and
AppView. It has no importer backend, archive upload endpoint, server-side OAuth
session, database, analytics, advertising, or session replay.

## Privacy and authority

- The selected ZIP remains a browser-owned `File`. Raw archive data, captions,
  media, filenames, and unrelated export data are not uploaded or persisted.
- Archive parsing and media processing run in a dedicated worker. IndexedDB
  contains only bounded, content-free resume and rollback metadata.
- OAuth is requested only after local parsing and review.
- The OAuth grant is exactly:

  ```text
  atproto repo:social.craftsky.feed.post?action=create&action=delete blob?accept=image/jpeg&accept=image/png&accept=image/webp
  ```

- The importer never requests update, wildcard repository, account-management,
  identity-management, or unrelated RPC permission.
- Handle entry uses the public `https://bsky.social` resolver. The UI must
  disclose this before sending a handle. A DID or direct PDS entry can avoid
  handle disclosure where the OAuth library supports it.
- Uploaded images and post records go directly to the authenticated PDS. A
  `social.craftsky.actor.profile/self` record is required before publication.

## Requirements

- Node.js 22.12 or newer.
- npm 11 (the exact package manager version is recorded in `package.json`).
- A current desktop browser with module workers, IndexedDB, `OffscreenCanvas`,
  and the File APIs used by the importer.

Install the checked dependency graph:

```bash
just importer-install
```

## Local development

Start the Vite server:

```bash
just importer-dev
```

Vite prints the loopback URL. Open that exact URL; the development OAuth client
identity includes its callback URL and the same narrow production scope.
Localhost and loopback IP addresses are the only non-HTTPS origins allowed to
perform real OAuth and PDS writes.

Run the package checks:

```bash
just importer-test
just importer-check
just importer-test-e2e
```

`importer-check` runs generated-contract drift checking, type checking, lint,
unit/integration tests, and a production build. Playwright is separate because
its browsers require an explicit one-time installation:

```bash
cd instagram-importer
npx playwright install
```

No real Instagram export belongs in this repository or its test fixtures. Tests
must use synthetic archives with invented content.

## Runtime modes

Runtime authority is derived from the exact browser origin, not merely from a
build-time flag:

| Mode | Origin and configuration | Real OAuth/PDS writes |
|---|---|---|
| Local | `localhost`, `127.0.0.1`, or `[::1]`; `localhost` is immediately canonicalized to `127.0.0.1` before app state is created | Enabled with loopback client metadata |
| Production | Exactly `https://import.craftsky.social` | Enabled with the committed canonical metadata document |
| Stable staging | Exact `VITE_STAGING_ORIGIN`, its own HTTPS client metadata, and `VITE_ENABLE_PDS_WRITES=true` | Enabled only when all three match |
| Ephemeral preview | Any other deployed origin | Hard-disabled; synthetic/mocked flows only |

A stable staging deployment must configure:

```dotenv
VITE_STAGING_ORIGIN=https://import-staging.example.org
VITE_OAUTH_CLIENT_ID=https://import-staging.example.org/oauth-client-metadata.json
VITE_ENABLE_PDS_WRITES=true
```

Start from `.env.staging.example`, which deliberately keeps writes disabled.
The staging client metadata is a separate public document whose `client_id`,
`client_uri`, and `redirect_uris` all describe the stable staging origin. Never
reuse the production client ID for staging. Do not enable the write flag on a
Cloudflare preview hostname.

The committed `public/oauth-client-metadata.json` is the canonical production
public-client document. It contains no secret. `build:staging` replaces the
emitted copy with metadata for the configured stable staging origin.
`build:preview` replaces it with a deliberately invalid, non-client marker, as
well as compiling the runtime with OAuth initialization, session restoration,
preflight, upload, create, and delete disabled. Keeping a marker prevents the
SPA fallback from serving `index.html` at the metadata path.

## Recovery boundary

Resume and bulk rollback depend on content-free local history. Rollback targets
only records durably created by that local import session and sends the stored
record CID as the PDS delete precondition, so edited, replaced, or
CID-different recreated records are preserved.

AT record CIDs address record content rather than a record incarnation. If
someone deletes an imported record and later recreates byte-identical content
at the same rkey, the CID is identical and the importer cannot distinguish the
new record from its original create. The rollback confirmation discloses this
limitation. Clear local history when bulk rollback authority is no longer
wanted.

## Production build

```bash
just importer-build
```

The output is `instagram-importer/dist/`. The build verifies generated Lexicon
contract drift and TypeScript before Vite emits static assets. Inspect the
artifact before deployment:

```bash
find instagram-importer/dist -maxdepth 2 -type f -print
sed -n '1,80p' instagram-importer/dist/_headers
sed -n '1,80p' instagram-importer/dist/oauth-client-metadata.json
```

The deployment contains only static HTML, JavaScript, CSS, fonts, workers,
images, the OAuth client document, and Cloudflare Pages policy files.

## Cloudflare Pages runbook

Create a **separate Pages project** for this package. Do not add it to the
marketing-site or Flutter-web Pages projects.

- Production branch: `main`
- Root directory: `/`
- Build command:

  ```bash
  npm ci --prefix instagram-importer
  if [ "$CF_PAGES_BRANCH" = "main" ]; then
    npm run --prefix instagram-importer build
  else
    npm run --prefix instagram-importer build:preview
  fi
  ```

- Build output directory: `instagram-importer/dist`
- Custom production domain: `import.craftsky.social`
- Analytics, Web Analytics, Zaraz, third-party scripts, and session replay:
  disabled

Use a distinct Pages project or stable branch deployment for staging, configure
the three staging variables above, and run
`npm run --prefix instagram-importer build:staging`. Disable arbitrary preview
branches for that project or route them through `build:preview`.

Before enabling production writes:

1. Confirm the custom domain serves HTTPS and resolves the canonical metadata
   document at `/oauth-client-metadata.json`.
2. Confirm `/oauth/callback` returns the SPA and the metadata callback matches
   exactly.
3. Inspect the deployed `_headers` behavior with browser devtools or `curl -I`.
4. Confirm every script, style, font, image, and worker is served from the
   importer origin.
5. Inspect OAuth consent and the returned grant against the exact scope above.
6. Run create and CID-guarded delete against a dedicated compatible test PDS.
7. Observe browser network traffic through parse, import, resume, and rollback;
   only the chosen OAuth/PDS infrastructure and the disclosed handle resolver
   are expected.
8. Confirm an ephemeral `pages.dev` preview reports
   `previewWritesDisabled` before any OAuth or PDS request.
9. Record a known-good Pages deployment ID so Cloudflare's deployment rollback
   can restore it.

Cloudflare dashboard isolation, live OAuth/PDS interoperability, deployed
headers, browser compatibility, large ZIP64 behavior, and firehose/AppView
convergence remain manual release gates. A successful repository build does
not prove them.

## Safety limits

Production limits are centralized in `src/config/safety.ts`. The initial
envelope allows at most 100,000 ZIP entries, a 64 MiB central directory, 32 MiB
per candidate post JSON file, 128 MiB combined candidate metadata, 25,000
normalized posts, 64 MiB per selected source image, 25 decoded megapixels,
12,000 pixels per source dimension, a 200:1 selected-entry decompression ratio,
one concurrent image decode/re-encode, 4,000 pixels per final image dimension,
and exactly 2,000,000 bytes per final image blob.

There is intentionally no overall ZIP-size or cumulative selected-media limit.
Archive metadata overflows stop locally with Posts-only export guidance; media
overflows omit that media during review.
