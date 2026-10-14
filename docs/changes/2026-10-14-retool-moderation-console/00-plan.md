# CraftSky Retool moderation console plan

Date: 14 October 2026  
Status: Approach accepted; implementation pending

## Purpose and architecture

Build a private Retool React console that gives authorised operators access to the full suite of CraftSky moderation tools. CraftSky remains responsible for permissions, decisions, audit history, retention and enforcement. Retool provides the operator interface and calls authenticated AppView admin APIs.

The existing backend is a strong foundation, but the full console needs additional read endpoints and several command routes. Extend the current domain services and tables rather than maintaining a separate moderation system in Retool. Retool should not connect directly to CraftSky PostgreSQL or store moderation records in Retool Database.

This plan records the accepted approach, findings from the 14 October exploration, implementation stages and validation criteria. Detailed API contracts and unresolved integration choices are settled during the stages below.

## Console scope

| Area | Operator functionality | Backend work needed |
|---|---|---|
| Cases | Report queue, filters, subject review, decisions and audit timeline | Content previews, subject search and richer queue context |
| Enforcement | Formal warnings, visibility warnings, hiding, takedowns, strikes, severe suspensions and restoration | Reuse existing commands and expose permitted actions |
| Appeals | Correspondence linking, confirmation, reconsideration and reversals | Browse correspondence and show complete appeal details |
| Safety incidents | Prioritisation, assignment, deadlines, confirmation and case creation | Complete incident reads and expose the existing confirmation service |
| External complaints | Intake, correspondence and links to cases or workflows | List/detail APIs and complaint-to-case linkage |
| Specialist workflows | Intimate-image complaints, credible threats, authority requests and child-safety reporting records | Workflow reads, missing transitions and consistent command protection |
| Evidence | Preservation, audited access, holds and retention status | Metadata reads, hold release and a suitable restricted-access flow |
| Accounts and operations | Account moderation history, age eligibility reviews, scanner failures, manual retry and operator administration | Admin account reads, a retry route and operator-management APIs |

Normal posts, structured project posts, accounts and business events must have useful review presentations. Image-safety incidents and restricted evidence retain their separate access boundaries.

Published AT Protocol labels and Ozone integration are separate extensions. The current report-forwarding seam only records `prepared_not_submitted`; it does not submit reports to another service.

## Current backend findings

### Existing moderation commands

[Moderation routes](../../../appview/internal/routes/routes_moderation.go) and [admin handlers](../../../appview/internal/api/moderation_admin.go) already provide case queues and details, decisions, appeal confirmation and resolution, effect changes and severe-suspension restoration.

Core moderation commands use expected revisions, idempotency keys, strict JSON decoding and authenticated actor attribution. Retool must preserve these contracts. A timed-out command is retried with the unchanged request and key; a revision conflict requires refreshed case data and renewed operator review.

### Content review and case context

The [admin detail response](../../../appview/internal/moderation/read_store.go) contains reports, decisions, effect history, strike context, standing and an event timeline. Its [safe subject snapshot](../../../appview/internal/moderation/presentation.go) contains identifiers rather than post text, project fields or event content.

Add dedicated admin preview responses that resolve the current subject, distinguish it from the reported record version and respect image-safety restrictions. Suppressed or unavailable content needs an explicit state. An admin preview must not become a generic route around restricted-media controls.

Case detail should also expose relevant origin, incident linkage and legal-classification context already represented in storage. Operator-created cases and external complaints need attributable provenance and subject linkage.

### Safety queues and workflow reads

The [safety status handler](../../../appview/internal/api/safety_admin.go) supports multiple work readers, but [production wiring](../../../appview/internal/app/routes_adapter.go) supplies only the image-incident reader. That [reader](../../../appview/internal/safetyincident/store.go) includes `detected` and `confirmed` incidents.

Include incidents in later active states, external complaints, workflow deadlines, information requests and relevant evidence/hold work. Provide bounded, paginated list/detail APIs rather than relying on an aggregate dashboard as the only browsing surface. Populate owner, cover, priority and deadline fields from the corresponding records.

### Services without admin routes

[System-detection confirmation](../../../appview/internal/moderation/system_detected.go) can create a moderation case from an incident but has no registered admin command route. [Manual image-scan retry](../../../appview/internal/imagesafety/worker_store.go) likewise exists in Go without an admin route.

Expose both through authenticated, permission-checked commands. Confirm supported subject mappings, including profile-image incidents, before treating incident-to-case conversion as complete.

### Operator authentication and permissions

AppView uses the [database-backed operator authenticator](../../../appview/internal/middleware/moderator_database.go) with the existing [operator, token, permission and assignment tables](../../../appview/migrations/000087_safety_operators.up.sql). Configuration still requires the older shared bearer-token and actor settings when moderation admin routes are enabled.

Resolve this mismatch, reuse individual operator identities and add explicit permissions where handlers currently rely only on successful moderator authentication. Capabilities returned to Retool guide the interface; AppView enforces the same permissions independently.

### Safety command and reporting semantics

[Safety command handlers](../../../appview/internal/api/safety_commands.go) do not consistently provide the revision and replay protection used by core moderation commands. Add equivalent protection where duplicate submissions or stale state could cause incorrect transitions or repeated records. Derive actor and integration provenance from trusted server context.

[Authority reporting](../../../appview/internal/safetyincident/csea_workflow.go) and [disclosure workflows](../../../appview/internal/safetyincident/workflow.go) record external references and audit events. The console must distinguish recording a completed external action from actually transmitting a report or disclosure.

### Evidence dependencies

Evidence preservation and access depend on a configured private object store. [Safety dependency setup](../../../appview/internal/app/deps_safety.go) leaves the evidence service unavailable when those credentials are not configured. Return capability/dependency state so the console can explain unavailable operations accurately.

## Retool findings and build approach

The MCP connection works. The inspected organisation has two existing React apps, `Untitled` and `Untitled-1`, staging and production resource environments, and no CraftSky API resource. The inspected published source of `Untitled` is a starter canvas. Check for unpublished threads before considering either existing app as a build target.

The MCP exposes app creation and continuation, resource configuration, source inspection, thread monitoring, publishing, mutation review and app access management. The writable-folder lookup returned no folders. The general folder lookup encountered a response-schema error; a filtered lookup found a Drafts folder, but that does not establish an eligible creation target. Resolve app placement during the first integration stage.

Use an OpenAPI-backed CraftSky API resource. Plain REST resources without an attached OpenAPI specification are unavailable to the MCP TypeScript resource-query tools. Keep the specification in the repository, aligned with the admin route contracts, and expose it through a suitable URL for Retool resource configuration.

Use the MCP for app creation, resources, inspection and publishing. Use the current `@tryretool/cli` for direct code editing and validation if needed. Prefer a project-scoped installation and retain reviewable app source alongside the backend work. The CLI was not installed during exploration. Retool recommends the CLI for app development and the MCP for resource and organisation operations: [building from anywhere](https://docs.retool.com/build/apps/concepts/build-from-anywhere).

Retool [documents per-user custom authentication](https://docs.retool.com/data-sources/guides/authentication/custom). Verify that flow with React apps and the chosen OpenAPI-backed resource before committing to the final credential design. Credentials belong in resource authentication, not frontend code. AppView must identify the operator from authenticated credentials, not a browser-supplied actor field.

Build with synthetic fixtures. Retool's [app security documentation](https://docs.retool.com/build/apps/concepts/security) states that function outputs can be sent to LLM providers during app development. Restricted evidence and real private case content must not become builder fixtures.

## Implementation stages

### 1. Prove the integration

Create an isolated private draft and an OpenAPI-backed resource against synthetic development data. Resolve the app-folder issue and verify network access to the selected backend; a Retool resource environment name alone does not provide a staging AppView or access to a local Docker service.

Verify an authenticated read, one reversible command, request/error handling, per-user authentication and source inspection. Prove the restricted-evidence transport before selecting that part of the app design.

Completion evidence: a working draft, reviewed source, attributable command result and a concrete authentication/evidence-access design.

### 2. Establish attributable operator access

Reuse the current operator model. Add an admin identity/capabilities endpoint and secure provisioning, token rotation and revocation tooling. Resolve obsolete shared-token configuration requirements. Define and enforce permissions for ordinary moderation, safety reads and commands, intake, eligibility review and operator administration.

Completion evidence: authorised operators can perform their assigned operations; revoked credentials and insufficient permissions fail at the backend; recorded actions identify the correct actor.

### 3. Complete the admin API

Add paginated list/detail endpoints for workflows, intakes, evidence metadata, scans and account moderation history. Add safe subject previews, incident confirmation, manual scan retry and explicit external/operator case provenance. Complete missing workflow and hold transitions.

Use new migrations for permission changes, provenance/linkage, workflow revision/replay protection and attributable hold-release history where required. Reuse existing domain records and preserve append-only audit history. Follow camelCase JSON, opaque cursors and the standard error envelope.

Completion evidence: an endpoint-to-console capability map, aligned OpenAPI contracts and integration tests covering the new reads and commands.

### 4. Build the console in functional slices

Build cases, decisions, appeals and restoration first. Then add safety incidents, external intake, specialist workflows, evidence controls and operations. All areas in the scope table are part of the intended final delivery.

Use CraftSky styling and a queue/detail layout. Keep internal evidence notes distinct from user-visible explanations. Confirm consequential actions with their actual effects. Pending commands show a spinner on the actioned control and disable conflicting controls. Refresh after successful commands and require renewed review after revision conflicts.

Completion evidence: every planned read and action is reachable through the interface, with useful loading, empty, unavailable and error states.

### 5. Complete restricted evidence handling

Retool displays safe metadata and initiates authorised evidence operations. Evidence bytes use a dedicated, audited AppView access flow with appropriate permission checks, an access reason and no-store behaviour. Establish the precise transport during stage 1 rather than assuming Retool's JSON-oriented functions support the required binary handling.

Expose preservation, access, hold creation/release and retention status. Keep private storage keys, credentials and restricted bytes out of ordinary console responses, builder output and diagnostic sinks.

Completion evidence: positive authorised access tests, denied-access tests, attributable audit records and protected-value canary checks.

### 6. Validate and roll out

Exercise complete workflows with synthetic fixtures:

- Report → decision → enforcement → appeal → reversal.
- Incident → confirmation → case → reporting record → resolution.
- External complaint → correspondence → case/workflow linkage.
- Evidence preservation → access → hold → release → retention handling.
- Age eligibility restriction → review → restoration.
- Dead-lettered scan → manual retry → resulting state.
- Operator provisioning → permitted action → credential revocation → denied action.

Check actual standing and content visibility alongside HTTP responses, including restoration convergence. Test duplicate submissions, stale revisions, role/assignment boundaries, safe previews and protected-data handling. Run appropriate Go integration/race tests, `./scripts/appview-check`, Docker smoke checks, OpenAPI validation, Retool code checks, browser verification and `git diff --check`.

The checked-in [Render configuration](../../../render.yaml) disables moderation admin routes. Live Render configuration was not inspected during exploration. Enabling the production console requires a controlled rollout after validation, using the repository's exact-tag release/deployment process. Treat resource secrets, operator provisioning, environment changes and deployment as explicit production operations.

Completion evidence: workflow test results, browser captures, reviewed app access, verified deployed capabilities and an operational runbook covering credential rotation, disabled access and recovery.

## First milestone and remaining decisions

The first implementation milestone is the authenticated Retool integration proof. Resolve these choices before expanding the build:

- New draft placement and handling of the existing starter apps.
- Reachable development backend and resource-environment configuration.
- Individual operator credential flow in Retool React.
- Restricted evidence transport and server-side preservation input.
- Repository location and validation workflow for Retool source.

The console gives operators access to existing and completed CraftSky capabilities. External reporting remains accurately represented, and public launch readiness continues to follow its existing evidence requirements.

## Governing references

- [Architecture reference](../../../atproto-craft-social-app-reference.md).
- [Product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit).
- [API architecture](../../superpowers/specs/2026-04-21-appview-api-architecture-design.md).
- [Logging and error reporting](../../development/logging-error-reporting.md).
- [Moderation strikes requirements](../2026-09-09-moderation-strikes/01-requirements.md).
- [Online safety readiness requirements](../2026-09-16-online-safety-launch-readiness/01-requirements.md).
- [Production Render runbook](../../operations/production-render.md).
- [Release operations](../../operations/releases.md).
