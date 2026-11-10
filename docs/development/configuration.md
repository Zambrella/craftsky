# Configuration and secrets

Configuration has one owner per setting:

| Setting | Owner |
|---|---|
| AppView defaults, types, requirements and ceilings | Go configuration loaders; [generated reference](configuration-reference.md) |
| Local services, container addresses and published ports | Compose and `scripts/compose-dev` |
| Production resources, service wiring and intentional overrides | `render.yaml` |
| Development integration credentials | Ignored `appview/config/.env.secrets.local`; `appview/config/.env.local` can override it |
| Production credentials | Render service secret variables and secret files |
| Flutter public build settings | `app/config/.env.*.local`; only local profiles are inherited |
| Svelte public build settings | `web-client/config/.env.<mode>.local`; Worker runtime settings in `web-client/wrangler.jsonc` |
| Deployment credentials | Operator credential store / environment, never client build inputs |

## Local setup and worktrees

Each component owns its environment files in `config/`: `appview/config/`, `app/config/`, and `web-client/config/`. Root `config/.env.local` holds optional shared Compose project, publish-host, port and memory overrides; copy `config/.env.local.example` when needed. Production backend settings remain in Render.

For an existing checkout, run `just config-init` to move legacy root backend files and web environment files into this layout. The migration preserves values, rejects conflicting destinations, keeps private files at mode 0600, and removes the old inputs after successful writes. Web files move byte-for-byte.

For a fresh primary checkout, copy `appview/config/.env.local.example` to `appview/config/.env.local` and
`appview/config/.env.secrets.local.example` to `appview/config/.env.secrets.local`, then run
`chmod 600 appview/config/.env.local appview/config/.env.secrets.local` and `just config-init`. Keep development
API keys and signing keys in `appview/config/.env.secrets.local`; keep flags, provider selection,
public identifiers in `appview/config/.env.local`. The examples leave generated-key
entries commented so initialization can supply independent keys. Do not copy an
example over an existing configured file. Run `just config-check` before startup.
The examples select credential-free Bluesky video; switch to Bunny only after
providing its complete development library configuration.

Run `just config-init` in the primary checkout after configuring its development integrations. Run it in an existing worktree to inherit missing development values from the primary checkout. `just worktree <branch>` runs this automatically for new worktrees. Worktrees created by other tools need `just config-init` once.

The primary checkout is located through Git's common directory; it does not depend on which branch is currently checked out there. Setup reads its development `appview/config/.env.local` and `appview/config/.env.secrets.local` (with legacy root inputs supported during migration), never `prod.env`, production mobile profiles, E2E passwords, Firebase service-account files, or deployment credentials from other files. Keep the source development-only. A credential accidentally placed in a development input cannot be distinguished from a development credential.

Missing integration values are copied into the destination `appview/config/.env.secrets.local`, with mode 0600. Existing destination keys (including intentionally empty ones) are preserved. Setup refuses to copy into paths that Git does not ignore. Worktree ports, project names, database URLs, remote-access controls, OAuth confidential keys and credential file paths are excluded. Missing mobile `.env.development.local` and `.env.development-android.local` profiles are copied; existing client profiles are left alone. Older base branches retain their legacy root filenames; setup detects the target layout.

If no video provider or Bunny credentials are configured, setup explicitly selects the credential-free Bluesky provider for development.

Setup generates independent random chat, OAuth handoff and Instagram data keys when absent, using each key's required encoding. Existing keys are never rotated automatically. These keys can authenticate outstanding operations: coordinate rotation and preserve backups rather than regenerating them on every startup.

Copies are snapshots. Re-running setup fills missing keys; it does not propagate rotations over existing destination values. Update affected development copies explicitly when rotating. Firebase ADC remains outside the repository and is mounted read-only by the existing push override.

Local AppView precedence, from highest to lowest:

1. Compose `environment:`: container topology and explicit remote-development controls.
2. `appview/config/.env.local`: per-worktree choices and legacy integration values.
3. `appview/config/.env.secrets.local`: inherited development integration values.
4. `appview/config/dev.env`: local profile.
5. Go defaults.

Shared launcher settings live in root `config/.env.local`, overriding any legacy launcher values in the backend files. Only the documented Compose and port controls are read from it; it is never passed to frontend builds as backend configuration.

The wrapper uses AppView's `godotenv` parser for quotes, comments and escapes. It exports only an explicit list of wrapper controls. Shell overrides apply to those controls; arbitrary shell variables are not automatically forwarded into Compose containers. The frontend launcher reads only its port, never sources backend secrets.

Use `just config-status` for variable presence without values and `just config-check` for local configuration validation without contacting services. Invalid integration bundles fail validation. The wrapper also checks remote-development transport/host authorization before starting Compose. Changes to env files require container recreation with `just dev-d`, not merely a restart.

Compose startup commands (`up`), including `just dev-d`, print a credential-free backend integration summary for Firebase push, Instagram, RevenueCat, Sentry, Brevo, video providers, Bunny CDN purge and Bluesky chat, followed by the backend's boolean feature flags. Port lookups used by client launchers do not print backend summaries. Flag names and development defaults are discovered from the Go configuration loaders. The summary describes configuration intent, not provider connectivity or client build settings. Firebase push reflects the wrapper's ADC discovery; the other entries use the development profile and local overrides.

Safety emails are opt-in with `SAFETY_EMAIL_ENABLED=true`, `BREVO_API_KEY` and
`MODERATION_ADMIN_ENABLED=true`. Their moderation-console links use
`SAFETY_EMAIL_CONSOLE_ORIGIN`, independently of OAuth. In development, omitting
it uses the loopback AppView origin from the configured callback (including the
worktree port); an explicit HTTP origin must use `localhost` or a loopback IP.
Production requires an explicit HTTPS console origin, for example
`https://appview.craftsky.social`. Origins cannot contain credentials, query
parameters, fragments or paths; the sender appends `/admin/`. Enabling the local
worker sends real generic alerts to the existing moderation mailbox; it does
not redirect delivery to a test inbox.

`just app-run` also prints mobile service configuration and build flags, including
explicit Flutter define overrides and debug-only gates. `just web-run` prints web
service configuration and the subscription flag using Vite's selected mode,
dotenv precedence and shell overrides. These summaries omit credential values;
configured diagnostics and push still require runtime consent or permission.

The tooling requires Python 3.9+ and the repository's pinned Go toolchain. It uses the existing Go dotenv dependency, without installing a second parser.

## Production

`render.yaml` replaces the old parallel `prod.env.example`. Keep non-secret deployment choices in its configuration group, direct database and service references in service wiring, and credential values in Render. Optional integrations are listed as `sync: false` service variables to document their inputs without enabling or changing them.

Render prompts for `sync: false` values only during initial creation. Adding such declarations does not populate an existing service during Blueprint updates. Before enabling an integration, supply its complete bundle and validate it with `/app/cli --env prod config-check` in the service environment. This validates configuration without database queries, migrations or provider requests; the existing pre-deploy `ping` then checks dependencies before migrations.

Keep Firebase Admin JSON in `/etc/secrets/firebase-admin.json`. Give each provider/bucket the minimum required permissions and use independent development and production credentials. Signing keys require documented, coordinated rotation; do not use a generic generated string when the loader requires a specific key format.

This cleanup changes no live Render settings. Review and validate the Blueprint, then manually sync it under the existing production change controls. Verify removed default-valued entries in the live environment during that sync: dashboard overrides and previously retained variables can still differ from repository intent. Continue exact-tag application deployments through `scripts/appview-deploy`.

## Maintenance

Run `just config-reference` after changing a Go configuration loader. CI verifies that the reference is current, private env files are untracked, Docker excludes secret inputs, and worktree inheritance preserves isolation. The generated table links to the exact default, parsing and validation definitions; it contains no credential values.

Vite and SvelteKit both load web dotenv files from `web-client/config/`. Development uses `.env.development.local`; production builds use `.env.production.local`. Their `.example` counterparts are tracked. Generic `.env` and `.env.local` files in this directory are rejected to prevent settings crossing modes. Shell variables override the selected profile. Mobile uses the same names, plus `.env.development-android.local` for Android development; Flutter receives its selected file explicitly. `just config-init` migrates earlier client filenames without changing values. Web E2E credentials remain local and are never inherited into worktrees.

Client build inputs are public. Firebase web configuration, Sentry DSNs and RevenueCat public SDK keys can be included; provider API secrets, OAuth private keys and service-account JSON cannot. E2E account passwords stay in separate ignored test configuration and are never copied by worktree setup.
