# Document Review: Flutter Push Notifications

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-07-15
Risk level: High

## Summary

The requirements and acceptance-test specification preserve the confirmed Flutter-only direction while explicitly containing the two narrow AppView follow-ups for default APNs sound and the non-production sender gate. The documents are internally consistent, retain the approved scope boundaries, and give complete traceability for all 40 Must requirements through 30 acceptance criteria into automated tests, with physical-device checks reserved for provider and native OS behavior.

No blocking contradiction, missing Must coverage, or unresolved product decision prevents coding planning. Two non-blocking details should be made explicit in the coding plan and its test design: the Android FCM default-channel binding, and the normalization policy for a future unknown push `type`.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Tests / Risk | The documents require creating one Android channel but do not explicitly require binding FCM notification messages that omit `channel_id` to that channel. The current AppView Android message supplies TTL only, while FR-025 permits no sender change other than APNs sound. Creating the channel without setting `com.google.firebase.messaging.default_notification_channel_id` could leave background OS notifications on FCM's fallback channel, and REG-006 currently checks channel existence rather than the binding. | `01-requirements.md` FR-023, FR-025, AC-027, RISK-015; `02-acceptance-tests.md` AT-012, REG-006, REG-007, MAN-001; `appview/internal/push/firebase_sender.go`; [Firebase Android FCM setup](https://firebase.google.com/docs/cloud-messaging/android/get-started) | In `04-coding-plan.md`, require the Android manifest default-channel metadata to reference the single created channel ID. Extend the static/native configuration test to assert that linkage; keep the AppView Android payload unchanged. |
| DR-002 | Suggestion | Requirements / Tests | The durable feed defines a safe unknown-category fallback, but provider-open parsing does not say whether a syntactically valid future push `type` is rejected or normalized to generic activity. Since resolution is authorized by `notificationId` and the current payload includes `type`, making this policy explicit would avoid an unnecessary forward-compatibility guess in the parser design. | `01-requirements.md` FR-008, FR-011, AC-007, AC-028, EC-009; `02-acceptance-tests.md` UT-002, TD-002 | In `04-coding-plan.md`, select and document a safe policy. Prefer normalizing an unknown but bounded `type` to a generic domain category while still requiring valid `notificationId` and `accountSubscriptionId`, then add that case to UT-002. |

## Traceability Review

- Planning to requirements: The recommended provider-neutral stream service is carried through FR-002, FR-019, NFR-002, and the test ownership model. Confirmed scope decisions are preserved: one Firebase project, direct permission prompt, always-present foreground banner, no receipt deduplication, in-app badge only, no polling, full-screen per-category settings without a master switch, single active account, safe routing resolution, and best-effort logout cleanup. The separate Flutter slice remains distinct from the completed AppView notification work except for FR-025 and FR-026.
- Requirements to acceptance criteria: All 40 Must requirements (BR-001 through BR-004, FR-001 through FR-026, NFR-001 through NFR-002, and RULE-001 through RULE-008) link to at least one externally verifiable acceptance criterion. Both Should requirements also link to acceptance criteria.
- Acceptance criteria to tests: AC-001 through AC-030 all appear in the coverage matrix and have one or more named verification paths. Every test case references requirement and acceptance-criterion IDs. No unknown or dangling requirement/AC reference was identified.

## Coverage Review

- Must requirements covered: 40 of 40. Each has automated unit, integration, acceptance, or regression coverage; native/provider assertions add bounded manual checks where deterministic automation is impractical.
- Missing or weak coverage: No missing Must path. DR-001 identifies a static assertion that should be strengthened. DR-002 identifies a forward-compatibility case that should be added to the payload-parser test. APNs credential provisioning remains an external prerequisite rather than a source-controlled test.
- Manual-only coverage: No Must requirement is exclusively manual. End-to-end FCM/APNs delivery, native permission UI, Android sound/vibration and channel behavior, iOS default sound, physical token rotation, and native settings recovery still require MAN-001 through MAN-004. MAN-005 is the operational verification for the bounded non-production sender gate.

## Risk And Approval Review

- Risk level: High. The change crosses OS permissions, Firebase/APNs configuration, background execution, auth-gated registration, secure account routing, sign-out/401 cleanup, native identifiers, and a narrow AppView sender change.
- Review requirement: Coding planning may proceed. Implementation still requires explicit user approval after the coding plan because the workflow documents mark this as high risk. Physical-device delivery must not begin until credential-aware startup reports push enabled and the intended local device/account is recorded; iOS delivery also requires the APNs authentication key to be configured outside the repository.
- Approval notes: Keep Firebase types at the adapter boundary, maintain one coordinator owner, preserve AppView resolution as the only navigation authority, keep non-production sending disabled outside a bounded manual check, and do not widen the AppView change beyond the default APNs sound and configuration-gate assertion.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Start with UT-002, the provider-data allowlist parser, then establish DID-keyed routing storage and validation before introducing Firebase initialization or listeners. The coding plan should incorporate DR-001 into native setup and REG-006, and settle DR-002 in the parser contract.
- Blocking issues: None. APNs credential provisioning blocks MAN-002 and launch enablement, not coding planning or implementation of the source-controlled work.

## Notes For Next Stage

- Preserve the test order already proposed in `02-acceptance-tests.md`; it establishes privacy and routing boundaries before plugin/native integration.
- Treat `appview/internal/routes/routes.go` and the implemented handlers as the wire-contract source: `/v1/*` remains camelCase JSON, bodyless seen, owner-scoped resolution, and authenticated device registration.
- Explicitly wire the Android manifest default FCM channel ID to the one “Craftsky notifications” channel because the AppView sender is intentionally not adding an Android `channel_id`.
- Keep provider payload parsing allowlisted and decide how a future unknown `type` becomes generic activity without allowing payload data to define a destination.
- Make the 401 cleanup ordering testable: best-effort FCM token deletion must be initiated before session state is cleared, while failure must never block local sign-out.
- Keep all sensitive sentinels out of logs, Sentry contexts/breadcrumbs, analytics, UI diagnostics, and exception strings, including resolution URLs containing notification IDs.
