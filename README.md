# CraftSky

A crafting-focused social platform built on the [AT Protocol](https://atproto.com), where makers share projects, techniques, and works in progress while keeping ownership of their identity and public records.

CraftSky is being developed for private beta. The Flutter client and Go backend are implemented, and production infrastructure is deployed. Full public launch still has [readiness checks and approvals](docs/operations/releases.md#private-beta-and-full-public-launch) to complete.

[Website](https://craftsky.social) · [Contribute](CONTRIBUTING.md) · [Roadmap](docs/roadmap.md) · [Documentation](docs/README.md) · [Report a bug](https://github.com/Zambrella/craftsky/issues/new/choose)

## What we're building

- A chronological feed for sharing craft projects, progress, and conversation.
- Structured project posts for sewing, knitting, crochet, and quilting.
- Portable public records on each member's Personal Data Server (PDS).
- No ads, with transparent business accounts and optional paid features.

The [product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing) explains the community principles and project-post field set. The [roadmap](docs/roadmap.md) records implemented work and future ideas; it is not a release schedule.

## Contributing

Code, documentation, accessibility feedback, bug reports, and product ideas are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for setup, checks, and the pull request process. For substantial work, open an issue to agree on scope before implementing it. Small fixes can go straight to a pull request.

Contributions use the repository's [AGPL-3.0 license](LICENSE). The [CLA proposal](CLA.md) is an inactive draft; no signature is required. Please follow the [Code of Conduct](CODE_OF_CONDUCT.md). Report vulnerabilities privately using [SECURITY.md](SECURITY.md).

## Run locally

For the backend, install Git, Docker with Compose v2, and [`just`](https://just.systems). Start Docker, then run:

```bash
git clone https://github.com/Zambrella/craftsky.git
cd craftsky
just dev-d
just ping
just tap-status
```

Compose starts PostgreSQL, MinIO, migrations, Tap, and AppView, including their bootstrap services. `just ping` checks database connectivity. `just tap-status` reports the firehose connection and may initially report no events while Tap connects. These checks do not prove every feature is working.

The startup output prints the local AppView and storage addresses. AppView uses `http://localhost:18080` in the primary checkout; linked worktrees get isolated services and alternate ports. Services bind to loopback by default. See the [AppView guide](appview/README.md) for configuration, OAuth, tests, and protected remote-device access.

To run the client, install [FVM](https://fvm.app) and the platform tools for your target. From the repository root:

```bash
fvm install
(cd app && fvm flutter pub get)
fvm exec just app-run
```

The repository pins Flutter in [.fvmrc](.fvmrc). `fvm exec` makes that SDK available to recipes that call `flutter`. `app-run` creates ignored local config files and discovers this checkout's AppView port. Choose a running Android emulator or iOS simulator for native development. See the [Flutter guide](app/README.md) for platform requirements and other targets.

Stop the local stack with `just down`; its data volumes are preserved. Local setup does not require production credentials. OAuth testing uses a real PDS, so use a dedicated test account: records published there are public and persist after resetting the local database.

## How it works

```text
Flutter client ── reads / write requests ──> AppView (Go + PostgreSQL)
                                              │          ▲
                                      record writes      │ indexed records
                                              ▼          │
                                         User's PDS → Relay → Tap
```

AppView mediates record writes and keeps OAuth credentials server-side. Reads come from its PostgreSQL index. Public records live on the PDS; private data such as drafts and mutes stays in AppView. Narrow, temporary credentials permit direct image and video uploads under [ADR 017](adr/017-direct-image-upload-service-token-handoff.md) and [ADR 012](adr/012-ephemeral-video-service-token-handoff.md).

Read the [architecture reference](atproto-craft-social-app-reference.md) and [architecture decisions](adr/) before proposing architectural changes.

## Repository map

| Path | Purpose |
| --- | --- |
| [app/](app/README.md) | Flutter client |
| [appview/](appview/README.md) | Go API, OAuth backend, and firehose indexing |
| [lexicon/](lexicon/README.md) | `social.craftsky.*` public record schemas |
| [web/](web/README.md) | Public website and policy pages |
| [instagram-importer/](instagram-importer/README.md) | Standalone browser importer |
| [service-status/](service-status/README.md) | Public service-status Worker |
| [verified-links/](verified-links/README.md) | Native app-link association files and OAuth fallbacks |
| [docs/](docs/README.md) | Development guides, roadmap, and operations |
| [online-safety/](online-safety/README.md) | Policy sources, assessments, and readiness records |

[AGENTS.md](AGENTS.md) documents repository conventions for contributors and coding agents. Releases and production deployments are maintainer operations, described in the [release runbook](docs/operations/releases.md).

## License

[GNU Affero General Public License v3.0](LICENSE) (AGPL-3.0). Copyright © 2026 Doug Todd and contributors.
