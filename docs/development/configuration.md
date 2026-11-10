# Configuration and secrets

Configuration has one owner per setting:

| Setting | Owner |
|---|---|
| AppView defaults, types, requirements and ceilings | Go configuration loaders; [generated reference](configuration-reference.md) |
| Local services, container addresses and published ports | Compose and `scripts/compose-dev` |
| Production resources, service wiring and intentional overrides | `render.yaml` |
| Development integration credentials | Ignored `.env.secrets.local`; `.env.local` can override it |
| Production credentials | Render service secret variables and secret files |
| Flutter public build settings | `app/config/*.env`; only local profiles are inherited |
| Svelte public build settings | `web-client/.env`; Worker runtime settings in `web-client/wrangler.jsonc` |
| Deployment credentials | Operator credential store / environment, never client build inputs |

## Local setup and worktrees

Run `just config-init` in the primary checkout after configuring its development integrations. Run it in an existing worktree to inherit missing development values from the primary checkout. `just worktree <branch>` runs this automatically for new worktrees. Worktrees created by other tools need `just config-init` once.

The primary checkout is located through Git's common directory; it does not depend on which branch is currently checked out there. Setup reads its development `.env.local` and `.env.secrets.local`, never `prod.env`, production/staging mobile profiles, E2E passwords, Firebase service-account files, or deployment credentials from other files. Keep the source development-only. A credential accidentally placed in a development input cannot be distinguished from a development credential.

Missing integration values are copied into the destination `.env.secrets.local`, with mode 0600. Existing destination keys (including intentionally empty ones) are preserved. Setup refuses to copy into paths that Git does not ignore. Worktree ports, project names, database URLs, remote-access controls, OAuth confidential keys and credential file paths are excluded. Missing mobile `local.env` and `local-android.env` profiles are copied; existing client profiles are left alone. Older base branches that do not ignore `.env.secrets.local` receive missing values in their already-ignored `.env.local` instead.

If no video provider or Bunny credentials are configured, setup explicitly selects the credential-free Bluesky provider for development.

Setup generates independent random chat, OAuth handoff and Instagram data keys when absent, using each key's required encoding. Existing keys are never rotated automatically. These keys can authenticate outstanding operations: coordinate rotation and preserve backups rather than regenerating them on every startup.

Copies are snapshots. Re-running setup fills missing keys; it does not propagate rotations over existing destination values. Update affected development copies explicitly when rotating. Firebase ADC remains outside the repository and is mounted read-only by the existing push override.

Local AppView precedence, from highest to lowest:

1. Compose `environment:`: container topology and explicit remote-development controls.
2. `.env.local`: per-worktree choices and legacy integration values.
3. `.env.secrets.local`: inherited development integration values.
4. `appview/environments/dev.env`: local profile.
5. Go defaults.

The wrapper uses AppView's `godotenv` parser for quotes, comments and escapes. It exports only an explicit list of wrapper controls. Shell overrides apply to those controls; arbitrary shell variables are not automatically forwarded into Compose containers. The frontend launcher reads only its port, never sources backend secrets.

Use `just config-status` for variable presence without values and `just config-check` for local configuration validation without contacting services. Invalid integration bundles fail validation. The wrapper also checks remote-development transport/host authorization before starting Compose. Changes to env files require container recreation with `just dev-d`, not merely a restart.

The tooling requires Python 3.9+ and the repository's pinned Go toolchain. It uses the existing Go dotenv dependency, without installing a second parser.

## Production

`render.yaml` replaces the old parallel `prod.env.example`. Keep non-secret deployment choices in its configuration group, direct database and service references in service wiring, and credential values in Render. Optional integrations are listed as `sync: false` service variables to document their inputs without enabling or changing them.

Render prompts for `sync: false` values only during initial creation. Adding such declarations does not populate an existing service during Blueprint updates. Before enabling an integration, supply its complete bundle and validate it with `/app/cli --env prod config-check` in the service environment. This validates configuration without database queries, migrations or provider requests; the existing pre-deploy `ping` then checks dependencies before migrations.

Keep Firebase Admin JSON in `/etc/secrets/firebase-admin.json`. Give each provider/bucket the minimum required permissions and use independent development and production credentials. Signing keys require documented, coordinated rotation; do not use a generic generated string when the loader requires a specific key format.

This cleanup changes no live Render settings. Review and validate the Blueprint, then manually sync it under the existing production change controls. Verify removed default-valued entries in the live environment during that sync: dashboard overrides and previously retained variables can still differ from repository intent. Continue exact-tag application deployments through `scripts/appview-deploy`.

## Maintenance

Run `just config-reference` after changing a Go configuration loader. CI verifies that the reference is current, private env files are untracked, Docker excludes secret inputs, and worktree inheritance preserves isolation. The generated table links to the exact default, parsing and validation definitions; it contains no credential values.

Client build inputs are public. Firebase web configuration, Sentry DSNs and RevenueCat public SDK keys can be included; provider API secrets, OAuth private keys and service-account JSON cannot. E2E account passwords stay in separate ignored test configuration and are never copied by worktree setup.
