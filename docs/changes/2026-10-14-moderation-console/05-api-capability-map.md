# Admin API capability map

Status: Complete locally; account preview added on 16 October 2026.

All routes require a current operator cookie. Unsafe requests also require the
exact Origin and CSRF token. Sets derive capabilities centrally; direct HTTP
calls receive the same restrictions as the console screens.

| Endpoint | Required capability | Console destination |
|---|---|---|
| `GET /v1/admin/accounts/{did}/eligibility` | `eligibility.review` | Eligibility review |
| `GET /v1/admin/accounts/{did}/preview` | `moderation.read` | Account and business profile preview |
| `GET /v1/admin/accounts/{did}/moderation` | `moderation.read` | Account history |
| `GET /v1/admin/accounts/{did}/moderation/history` | `moderation.read` | Account history |
| `POST /v1/admin/eligibility/reviews` | `eligibility.review` | Eligibility review |
| `GET /v1/admin/moderation/cases` | `moderation.read` | Cases and appeals |
| `POST /v1/admin/moderation/cases` | `moderation.manage` | Cases and appeals |
| `GET /v1/admin/moderation/cases/{caseReference}` | `moderation.read` | Cases and appeals |
| `POST /v1/admin/moderation/cases/{caseReference}/appeal-confirmations` | `moderation.manage` | Cases and appeals |
| `POST /v1/admin/moderation/cases/{caseReference}/appeal-resolutions` | `moderation.manage` | Cases and appeals |
| `POST /v1/admin/moderation/cases/{caseReference}/decisions` | `moderation.manage` | Cases and appeals |
| `POST /v1/admin/moderation/cases/{caseReference}/effect-changes` | `moderation.manage` | Cases and appeals |
| `GET /v1/admin/moderation/cases/{caseReference}/preview` | `moderation.read` | Cases and appeals |
| `GET /v1/admin/moderation/cases/{caseReference}/provenance` | `moderation.read` | Cases and appeals |
| `POST /v1/admin/moderation/cases/{caseReference}/restorations` | `moderation.manage` | Cases and appeals |
| `GET /v1/admin/operators` | `operator.manage` | Operators |
| `GET /v1/admin/operators/{operatorId}` | `operator.manage` | Operators |
| `GET /v1/admin/operators/{operatorId}/events` | `operator.manage` | Operator audit |
| `POST /v1/admin/operators` | `operator.manage` | Allow DID |
| `POST /v1/admin/operators/{operatorId}/permission-set` | `operator.manage` | Permission set |
| `POST /v1/admin/operators/{operatorId}/disable` | `operator.manage` | Disable access |
| `POST /v1/admin/operators/{operatorId}/session-revocations` | `operator.manage` | Revoke sessions |
| `GET /v1/admin/operations/{operationId}` | `operator.manage` | Operation detail |
| `GET /v1/admin/safety/incidents/{incidentReference}/assignments` | `incident.confirm` | Incident assignments |
| `POST /v1/admin/safety/incidents/{incidentReference}/assignments` | `operator.manage` | Assign owner/cover/helper |
| `POST /v1/admin/safety/incidents/{incidentReference}/assignments/{assignmentId}/release` | `operator.manage` | Release assignment |
| `GET /v1/admin/operations` | `operator.manage` | Operations |
| `GET /v1/admin/safety/evidence/capabilities` | `evidence.access` | Storage capabilities and limits |
| `GET /v1/admin/safety/evidence/{evidenceId}/accesses` | `evidence.access` | Access audit |
| `GET /v1/admin/safety/evidence/{evidenceId}/holds` | `evidence.access` | Evidence hold coverage |
| `GET /v1/admin/safety/evidence/{evidenceId}/retention-events` | `evidence.access` | Retention audit |
| `GET /v1/admin/safety/evidence/{evidenceId}/retention-jobs` | `evidence.access` | Retention processing |
| `GET /v1/admin/safety/holds/{holdId}/evidence` | `hold.manage` | Held evidence metadata |
| `GET /v1/admin/safety/holds/{holdId}/releases` | `hold.manage` | Hold release audit |
| `GET /v1/admin/safety/evidence` | `evidence.access` | Restricted evidence |
| `GET /v1/admin/safety/evidence/{evidenceId}` | `evidence.access` | Restricted evidence |
| `POST /v1/admin/safety/evidence/{evidenceId}/access` | `evidence.access` | Restricted evidence |
| `GET /v1/admin/safety/external-intakes` | `intake.manage` | External intake |
| `POST /v1/admin/safety/external-intakes` | `intake.manage` | External intake |
| `GET /v1/admin/safety/external-intakes/{intakeId}` | `intake.manage` | External intake |
| `POST /v1/admin/safety/external-intakes/{intakeId}/appeal-link` | `intake.manage` | External intake |
| `GET /v1/admin/safety/external-intakes/{intakeId}/correspondence` | `intake.manage` | External intake |
| `POST /v1/admin/safety/external-intakes/{intakeId}/correspondence` | `intake.manage` | External intake |
| `GET /v1/admin/safety/holds` | `hold.manage` | Restricted evidence |
| `GET /v1/admin/safety/holds/{holdId}` | `hold.manage` | Restricted evidence |
| `POST /v1/admin/safety/holds/{holdId}/releases` | `hold.manage` | Restricted evidence |
| `GET /v1/admin/safety/incidents` | `safety.read` | Safety incidents |
| `GET /v1/admin/safety/incidents/{incidentReference}` | `incident.readSafe` | Safety incidents |
| `POST /v1/admin/safety/incidents/{incidentReference}/authority-reports` | `authority.report` | Safety incidents |
| `POST /v1/admin/safety/incidents/{incidentReference}/confirmations` | `incident.confirm` | Safety incidents |
| `POST /v1/admin/safety/incidents/{incidentReference}/csea-classification` | `authority.report` | Safety incidents |
| `GET /v1/admin/safety/incidents/{incidentReference}/events` | `safety.read` | Safety incidents |
| `POST /v1/admin/safety/incidents/{incidentReference}/evidence` | `evidence.preserve` | Restricted evidence |
| `POST /v1/admin/safety/incidents/{incidentReference}/holds` | `hold.manage` | Restricted evidence |
| `GET /v1/admin/safety/incidents/{incidentReference}/information-requests` | `authority.report` | Safety incidents |
| `POST /v1/admin/safety/incidents/{incidentReference}/information-requests` | `authority.report` | Safety incidents |
| `GET /v1/admin/safety/incidents/{incidentReference}/reports` | `authority.report` | Safety incidents |
| `POST /v1/admin/safety/incidents/{incidentReference}/resolution` | `authority.report` | Safety incidents |
| `GET /v1/admin/safety/incidents/{incidentReference}/subjects` | `incident.confirm` | Safety incidents |
| `POST /v1/admin/safety/information-requests/{requestId}/response` | `authority.report` | Safety incidents |
| `GET /v1/admin/safety/scans` | `safety.read` | Scans |
| `GET /v1/admin/safety/scans/{scanId}` | `safety.read` | Scans |
| `POST /v1/admin/safety/scans/{scanId}/retries` | `scan.retry` | Scans |
| `GET /v1/admin/safety/status` | `safety.read` | Safety incidents |
| `GET /v1/admin/safety/workflows` | `safety.read` | Specialist workflows |
| `POST /v1/admin/safety/workflows/authority-requests` | `workflow.manage` | Specialist workflows |
| `POST /v1/admin/safety/workflows/authority-requests/{workflowId}/closure` | `workflow.manage` | Specialist workflows |
| `POST /v1/admin/safety/workflows/authority-requests/{workflowId}/disclosures` | `disclosure.approve` | Specialist workflows |
| `POST /v1/admin/safety/workflows/authority-requests/{workflowId}/verification` | `workflow.manage` | Specialist workflows |
| `POST /v1/admin/safety/workflows/credible-threats` | `workflow.manage` | Specialist workflows |
| `POST /v1/admin/safety/workflows/credible-threats/{workflowId}/outcome` | `workflow.manage` | Specialist workflows |
| `POST /v1/admin/safety/workflows/intimate-images` | `workflow.manage` | Specialist workflows |
| `POST /v1/admin/safety/workflows/intimate-images/{workflowId}/outcome` | `workflow.manage` | Specialist workflows |
| `GET /v1/admin/safety/workflows/{workflowId}` | `safety.read` | Specialist workflows |
| `GET /v1/admin/safety/workflows/{workflowId}/events` | `safety.read` | Specialist workflows |

Admin receives every listed capability. Mod receives `moderation.read`,
`moderation.manage`, `safety.read`, `intake.manage`, `incident.readSafe`,
`incident.confirm` and `scan.retry`. Helper receives only `incident.readSafe`,
restricted to its current incident assignments. Helpers cannot list global
queues, subjects, evidence, workflows, scans, operations or account history.

CSEA classification/reporting/resolution requires `authority.report`; ordinary
human detection confirmation uses `incident.confirm`. This matches the existing
specialist service authorization.

Restricted byte access remains a separate, reason-audited binary response. It
is never buffered in a command receipt. Evidence preservation returns only an
evidence ID; private object keys are not part of the browser contract.

External authority report/disclosure endpoints record supplied references and
approvals. They do not transmit anything to an authority or a third party.

See [OpenAPI](../../../appview/internal/api/moderation_console.openapi.json) and
[stage report](04-admin-api-implementation.md) for request and verification details.


Stage 5 evidence access requires `{incidentReference, reason}` and returns a
no-store, nosniff binary response through AppView. The incident must match the
stored evidence. Binary responses never enter replay receipts. Preservation
accepts only supported image signatures and at most 512 KiB (or a lower advertised
JSON-body allowance). Existing evidence/hold lists accept the incident filter;
all new audit/retention reads retain bounded opaque-cursor pagination.
