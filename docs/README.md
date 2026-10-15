# CraftSky documentation

Start with the [repository README](../README.md) for the product and quick start, or [CONTRIBUTING.md](../CONTRIBUTING.md) to make a contribution.

## Product and architecture

- [Product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit?usp=sharing) — community principles and project-post intent.
- [Roadmap](roadmap.md) — implementation history, open work, and future ideas.
- [Architecture reference](../atproto-craft-social-app-reference.md) — PDS/AppView split and design rationale; use later ADRs for subsequent decisions.
- [Architecture decisions](../adr/) — recorded choices and their constraints.
- [API architecture](superpowers/specs/2026-04-21-appview-api-architecture-design.md) and [wire conventions](superpowers/specs/2026-04-22-api-wire-alignment-design.md) — required reading for API changes.
- [Lexicons](../lexicon/README.md) — public record schemas and generation rules.

## Development

| Guide | Covers |
| --- | --- |
| [Flutter client](../app/README.md) | SDK, local configuration, devices, and OAuth links |
| [AppView](../appview/README.md) | Compose, backend ownership, tests, and OAuth debugging |
| [Website](../web/README.md) | Local serving, browser checks, and static build |
| [Instagram importer](../instagram-importer/README.md) | Browser-only importer, synthetic fixtures, and package checks |
| [Service status](../service-status/README.md) | Worker checks and local status simulation |
| [Verified links](../verified-links/README.md) | App-link hosting and OAuth fallbacks |
| [Flutter integration tests](../app/integration_test/README.md) | Native journey coverage and its limits |
| [Logging and error reporting](development/logging-error-reporting.md) | Mandatory diagnostic rules and verification |
| [Repository conventions](../AGENTS.md) | Architecture boundaries and contributor/agent guidance |

## Community and maintenance

- [Contribution guide](../CONTRIBUTING.md), [Code of Conduct](../CODE_OF_CONDUCT.md), and [security reporting](../SECURITY.md).
- [Issue and PR triage](maintainers/triage.md) — scope decisions, starter tasks, reviews, and backlog upkeep.
- [Release runbook](operations/releases.md) — maintainer-operated versions, tags, builds, and deployments.
- [Production Render runbook](operations/production-render.md) — infrastructure inspection, deployment, and recovery.
- [Online safety documentation](../online-safety/README.md) — policy sources, assessments, and readiness gaps. Drafts and implementation checkboxes are not evidence of approval or full public-launch readiness.

## Design history

[Specs](superpowers/specs/), [plans](superpowers/plans/), and [change packages](changes/) preserve the reasoning and verification for individual changes. They are historical context, not a sequential setup guide. Start with the component README and current code, then consult the relevant package. If an older design conflicts with a later ADR or implementation, flag the mismatch and update the contributor-facing guidance.
