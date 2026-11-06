# CraftSky in numbers

A standalone, buildless HTML/CSS/JavaScript site for
`https://stats.craftsky.social`. The public assets live in `public/`; the
Cloudflare Worker serves only those files. No application server, database,
frontend framework or chart library is needed by the website.

Craft icons in `public/craft_icons/` are unmodified copies of the Flutter app's
`app/assets/design/icons/*.svg`, selected by the same five craft names as
`CraftIcon.assetPathFor`. They use decorative CSS masks, matching the web
client's icon approach; keep these copies in sync with the app assets.

## Develop and check

From the repository root:

```sh
just stats-setup
just dev-d
just stats-dev
```

Open `http://127.0.0.1:4186`. The dev command discovers this worktree's running
AppView through `scripts/compose-dev` and serves the page with its local stats
API URL. The public source and production API URL stay unchanged. Counts come
from the local database; a fresh local stack may return zero counts.

Opening the public HTML through an ordinary static server instead requests
production. Until the backend release is deployed, that endpoint returns 404;
production's CORS allowlist also intentionally excludes local development
origins. Use `just stats-dev` for real local data. The `dev:static` command on
port 4187 is reserved for browser tests, which intercept the production URL.

```sh
just stats-check
just stats-dry-run
```

Browser tests intercept the stats request with explicit test fixtures; they do
not use production data. They cover loading, populated and zero snapshots,
malformed data, safe open craft labels, failure/retry and recovery focus,
reduced motion, credential omission, the join link and layouts from 320px to 1440px. The dry run validates/packages the Worker
locally without publishing. These checks do not establish hosted CORS, DNS,
certificates or production database behavior.

## One public API

`GET https://appview.craftsky.social/v1/public/stats` needs no authentication
and returns a fresh database snapshot. Requests omit browser credentials and
responses use `Cache-Control: no-store`; no stats cache is implemented.

```json
{
  "generatedAt": "2026-11-06T12:00:00Z",
  "accounts": 0,
  "posts": 0,
  "projectPosts": 0,
  "finishedProjectPosts": 0,
  "projectsByCraft": []
}
```

Each chart row has `craft` (a full craft identifier such as
`social.craftsky.feed.defs#knitting`) and integer `count`. The timestamp records
when the snapshot was generated. The headline fields count:

- **Accounts:** CraftSky profiles eligible for anonymous viewing.
- **Posts:** visible top-level CraftSky posts, including project posts; replies
  are excluded.
- **Project posts:** visible top-level project posts, excluding quotes.
- **Finished project posts:** those project posts marked finished.

The chart groups project posts by their primary craft, so one post contributes
to one bar. Additional crafts do not multiply counts. A project post is a
snapshot of a make, rather than a unique physical project: two updates about
the same sweater count twice. The API applies existing anonymous-view owner,
moderation, lifecycle and media eligibility rules. These are AppView-indexed
public counts, not a census of the entire AT Protocol network.

## Hosting and publication

`wrangler.json` declares a separate static-assets Worker, `craftsky-stats`, in
the existing CraftSky Cloudflare account. Its custom domain is
`stats.craftsky.social`; `workers.dev` is disabled. The source does not create
remote resources or alter DNS. There is no automatic production deployment.

Before first publication:

1. Review and merge the site and AppView changes. Release/deploy AppView using
   the repository's exact-tag procedure in
   [`docs/operations/releases.md`](../docs/operations/releases.md).
2. Review the `render.yaml` change and manually sync the Blueprint so production
   `ALLOWED_ORIGINS` includes `https://stats.craftsky.social`. Infrastructure
   auto-sync remains disabled. Follow the
   [Render runbook](../docs/operations/production-render.md).
3. From clean, pushed `main`, run `just stats-check` and `just stats-dry-run`.
   Authenticate using Wrangler's normal local login and verify the configured
   Cloudflare account and intended hostname before publication.
4. From `stats/`, the operator can perform the reviewed first publication and
   attach the custom domain with:

   ```sh
   npm exec -- wrangler whoami
   npm exec -- wrangler deploy --config wrangler.json
   ```

   This command creates/updates live infrastructure and serves the page. Check
   the Cloudflare custom-domain certificate/DNS state before calling it live.

For later visual review, upload a version without promoting it, from `stats/`:

```sh
npm exec -- wrangler versions upload --config wrangler.json
```

Review the returned version URL. The production API's exact-origin CORS
allowlist does not include arbitrary version URLs, so preview data should use a
local API or explicit test fixture; a version URL alone does not prove hosted
production data access. Promote the exact reviewed version explicitly:

```sh
npm exec -- wrangler versions deploy <reviewed-version-id>@100 --config wrangler.json
```

`versions deploy` does not apply route/custom-domain changes; review those
separately if the configuration changes. Keep the previous production version
ID for rollback via the same explicit version-deploy command.

After publication, verify the real page on `https://stats.craftsky.social`, its
single stats request with that Origin, genuine API values, loading/failure
behavior and the `https://app.craftsky.social` join link. API release, CORS
configuration and site publication are separate operations; local tests are
not proof of those live steps.
