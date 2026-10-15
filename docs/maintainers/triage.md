# Issue and pull request triage

This is the maintainer workflow for the [CraftSky issue tracker](https://github.com/Zambrella/craftsky/issues). Contributors can use [CONTRIBUTING.md](../../CONTRIBUTING.md) to prepare a report or PR. Doug Todd currently makes scope, merge, and release decisions; contributors do not need repository administration access.

## First review

Review new reports as capacity allows. Prioritize possible security, privacy, data-loss, and service-blocking issues over ordinary feature requests. There is no guaranteed response or resolution time.

1. Check for exposed credentials, private information, or vulnerability details. Route security reports to [the private channel](../../SECURITY.md). Restrict or remove sensitive content using GitHub's available controls; arrange credential revocation where needed rather than treating comment deletion as sufficient.
2. Check whether the report belongs here. Service content reports and moderation appeals go through [the service reporting process](https://craftsky.social/reporting); contributor conduct concerns follow the [Code of Conduct](../../CODE_OF_CONDUCT.md).
3. Search for duplicates. Link the canonical issue and preserve useful reproduction details before closing a duplicate with an explanation.
4. Identify the affected component, impact, and reproduction. Ask for the smallest missing detail. Do not ask for secrets, real exports, or private payloads.
5. Decide the next step: investigate a bug, discuss a proposal, accept a bounded task, or close it with a reason. An issue being open does not mean its implementation is approved.

## Labels and status

Keep labels small and useful. Use existing equivalents where possible; create the following labels in repository settings if they are missing. The issue forms deliberately do not auto-apply labels, so they work before this setup is done.

| Label | Meaning |
| --- | --- |
| `bug` | A reported or confirmed defect; the issue text records reproduction status. |
| `enhancement` | A proposed change to product behavior. |
| `documentation` | Documentation or contributor experience work. |
| `question` | A development or usage question. |
| `needs information` | The next step requires specific details from the reporter. |
| `help wanted` | A maintainer has agreed the scope and welcomes a contribution. |
| `good first issue` | An accepted, bounded task with clear pointers and verification steps, suitable for someone new. |

Record component, urgency, acceptance criteria, and blockers in the issue body or a maintainer comment; additional label taxonomies are optional. Use a linked draft PR to show active implementation. Remove `needs information` when the requested details arrive and remove contributor labels if the scope is no longer ready.

Do not promise an assignment just because someone comments. Confirm coordination in the thread, link any implementation, and check in before taking over apparently abandoned work. No automatic stale closure is configured by this guide. If an issue cannot progress, explain what is missing and how it can be reopened.

## Prepare a contributor-ready task

Before applying `help wanted`, record:

- The problem, expected result, and boundaries of the change.
- Relevant files or guides and any existing reproduction.
- Acceptance criteria and commands or manual steps that verify the result.
- Known dependencies, privacy constraints, and required design decisions.

Use `good first issue` only when the task can be completed without production credentials or unresolved architectural decisions. Starter tasks should not require an unfamiliar contributor to discover the scope themselves.

## Review a pull request

Check that the PR explains the problem and resulting behavior, links agreed scope where needed, and uses the [template](../../.github/pull_request_template.md). Review correctness, privacy and authority boundaries, error handling, maintainability, and relevant coverage. Distinguish required fixes from suggestions.

Require the relevant local and PR checks described in [CONTRIBUTING.md](../../CONTRIBUTING.md#check-your-work). Verify skipped checks and manual gaps explicitly; path-aware CI does not cover every component. UI changes need screenshots or recordings. API and schema changes need explicit impact notes; Lexicon changes also need an ADR and generated output. Do not require a CLA signature while [the CLA remains inactive](../../CLA.md).

A maintainer merges once review concerns and required checks are resolved. Close related issues only when the stated acceptance criteria are met. Update the roadmap when delivery changes its status. Merging is separate from release: use the [release runbook](../operations/releases.md) for tags, deployment, and verification, and describe a change as deployed only after observing that deployment.

## Keep the backlog useful

Periodically review accepted tasks, unresolved reports, and the [roadmap](../roadmap.md). Link superseding work, remove obsolete instructions, and close declined or completed work with a short explanation. Revisit starter tasks when their dependencies change. Do not translate roadmap ideas into release commitments without an explicit decision.
