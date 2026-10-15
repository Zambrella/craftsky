# Contributing to CraftSky

Thanks for helping build a social space for makers. Code, documentation, bug reports, accessibility feedback, translations, and product ideas are welcome. You do not need to know AT Protocol to start with a documentation or UI fix.

## Find a starting point

1. Read the [README](README.md) for the product and local quick start, and the [roadmap](docs/roadmap.md) for context.
2. Search [existing issues and pull requests](https://github.com/Zambrella/craftsky/issues) before opening new work. Issues marked `help wanted` are open for contributions; `good first issue` indicates a bounded task with setup and verification guidance.
3. Comment on an issue you want to tackle so others can coordinate. For a substantial feature, dependency change, API change, or schema change, agree on the approach with a maintainer before implementation. Small, focused fixes can go straight to a PR.

If no starter issues are available, a reproducible bug report or a correction to these setup instructions is useful. The roadmap is a planning document, not a list of approved assignments. Maintainer review capacity varies; there is no guaranteed response time or acceptance of a proposal.

## Report bugs and propose changes

Use the [issue chooser](https://github.com/Zambrella/craftsky/issues/new/choose) for bugs, feature proposals, documentation improvements, or development questions. Include reproduction steps, expected and actual results, platform/version information, and a small synthetic example where possible.

Remove credentials, session tokens, OAuth callback URLs, personal data, and private content from logs and screenshots. Use [private security reporting](SECURITY.md) for vulnerabilities. For reports about content or account moderation, use the [service reporting routes](https://craftsky.social/reporting) rather than public GitHub issues.

## Set up your workspace

Fork the repository on GitHub, clone your fork, and create a branch:

```bash
git clone https://github.com/YOUR_USERNAME/craftsky.git
cd craftsky
git remote add upstream https://github.com/Zambrella/craftsky.git
git switch -c fix/describe-your-change
```

Run commands from the repository root unless a guide says otherwise. Use only development configuration and dedicated test accounts. Do not copy production environment files or credentials.

| Area | Tools and setup |
| --- | --- |
| Backend | Docker + Compose v2, `just`, and Go matching [go.mod](appview/go.mod). Run `just dev-d`. See [AppView setup](appview/README.md). |
| Flutter | FVM with the SDK pinned in [.fvmrc](.fvmrc), plus target platform tooling. Run `fvm install`, then `(cd app && fvm flutter pub get)` and `fvm exec just app-run`. See [client setup](app/README.md). |
| Website | Python for local serving; Node.js 22+ and npm for checks. See [website setup](web/README.md). |
| Instagram importer | Node.js 22.12+ and the npm version in its package manifest. See [importer setup](instagram-importer/README.md). |
| Service status | See [status setup](service-status/README.md) for local checks and preview. |
| Documentation | A text editor; check relative links and any commands you change against their source. |

The local stack supplies PostgreSQL and MinIO, and the app's example configuration can leave billing and Sentry disabled. Production signing, deployment, and provider credentials are not required for ordinary contributions. Some journeys, including real OAuth and external media services, need separate test accounts or provider configuration; describe what you could and could not verify.

Linked Git worktrees receive isolated Compose projects, volumes, and alternate ports automatically. `just worktree feature/my-change` creates a branch and checkout under `.worktrees/`; see `./scripts/worktree-create --help` for helper options. Use `./scripts/compose-dev` instead of raw `docker compose` when operating on your development stack so it selects the correct worktree.

## Make the change

Keep a PR focused on one problem. Add regression coverage for changed behavior and update the relevant documentation. Avoid unrelated formatting, generated output, or dependency upgrades.

Read [AGENTS.md](AGENTS.md) for the full conventions. In particular:

- Flutter reads from AppView; record writes are AppView-mediated and go to the PDS. OAuth access/refresh tokens and DPoP keys stay server-side.
- Public PDS records and private AppView data have different privacy boundaries. Never use real personal exports or private content as fixtures.
- For API changes, follow the [API architecture](docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md) and [camelCase wire conventions](docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md).
- For schema changes, read the [Lexicon guide](lexicon/README.md) and [project Lexicon rules](.agents/skills/atproto-lexicon/SKILL.md), include an ADR, and regenerate with `just lexgen`. Coding agents should invoke the Lexicon skill before edits. Human contributors can follow the same written checklist without installing an agent.
- Write SQL queries in `appview/queries/` and migrations in `appview/migrations/`; preserve transaction and lifecycle invariants. Use typed atproto identifiers and parse at boundaries.
- Follow the mandatory [logging and error-reporting guide](docs/development/logging-error-reporting.md) for diagnostic changes, including protected-value checks.

Existing design specs and change packages explain decisions. A small correction does not need a new planning document; discuss architectural changes before writing a large implementation.

## Check your work

Run the checks for the parts you changed, and report their actual results in the PR. A skipped check is not a passing check.

| Change | Local validation |
| --- | --- |
| Go / AppView | `just fmt`, then `just test` with `just dev-d` running. `just appview-test-unit` is a fast, incomplete iteration path. AppView changes require `just appview-check` before merge; it uses disposable services for the full release gate. |
| Flutter | `fvm exec just app-analyze` and `fvm exec just app-test`. For UI changes, inspect the affected screens and attach screenshots or a recording. See [device integration tests](app/integration_test/README.md) for applicable native journeys. |
| Lexicons / generated types | `just lexgen-check` and the relevant backend/client checks, including `just appview-check`. Commit regenerated output with the schema. |
| Website | `just web-setup` once, then `just web-check`. This checks a local build; no login or deployment is needed. |
| Instagram importer | `just importer-install`, `just importer-check`, and `just importer-test-e2e` for browser changes. Install Playwright browsers as described in its README. |
| Service status | `just service-status-test`. |
| Documentation / issue forms | Check links, command names, formatting, and YAML structure. Run any area-specific checks needed for changed executable examples. |

The full backend gate also needs `curl`, `diff`, Git, Go, Docker Compose v2, `just`, and `rg`; it downloads pinned check tools and images. It is more expensive than the unit path. If your environment cannot run a required check, explain the limitation so a maintainer can arrange verification before merge.

[GitHub Actions](.github/workflows/backend-ci.yml) selects PR checks by changed paths. It does not cover every package or every manual journey, and it does not publish releases. A green PR is not evidence of a deployed or device-verified change.

## Submit a pull request

1. Commit your work on your branch. Focused commit messages are helpful; conventional commit prefixes are welcome but optional.
2. Push the branch to your fork and open a PR against `Zambrella/craftsky:main`. A draft PR is welcome for early feedback.
3. Fill in the [PR template](.github/pull_request_template.md): explain the problem and resulting behavior, link related issues, list checks and limitations, and explicitly state API and Lexicon impact. Attach visual evidence for UI changes.
4. Respond to review feedback and keep the scope focused. A maintainer reviews the change and merges it after the applicable checks and concerns are resolved. Agree on substantial scope changes before adding more work.

You do not need production access, release tags, or deployment credentials to submit a PR. Maintainers handle releases separately using the [release runbook](docs/operations/releases.md). The [triage guide](docs/maintainers/triage.md) explains how issues and PRs are handled.

## License and community

Contributions are submitted under the repository's [AGPL-3.0 license](LICENSE). Submit only work you have the right to contribute and preserve applicable third-party notices. No CLA signature or DCO sign-off is currently required. [CLA.md](CLA.md) remains an inactive proposal; any future change to contribution terms will need to be announced explicitly.

Follow the [Code of Conduct](CODE_OF_CONDUCT.md) in issues, reviews, and other project spaces. Ask development questions through the issue chooser so answers can help the next contributor too.
