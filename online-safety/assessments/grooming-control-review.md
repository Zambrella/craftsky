# Grooming Contact and Discovery Control Review

| Field | Value |
|---|---|
| Owner | Douglas Todd |
| Reviewed | 9 November 2026 |
| Status | Source review and focused local backend checks complete; live verification pending |
| Approval | Evidence record only; no new product decision or overall risk approval |

This follows the owner's challenge to High overall CSEA risk. His position that
CraftSky does not target or particularly appeal to children remains relevant;
no measured age distribution is available. Assess that context alongside the
confirmed 16+ policy, launch DMs and video uploads.

## Findings

| Control/pathway | Source or local evidence | Limit |
|---|---|---|
| Messaging connection | `appview/internal/chat/connection_service.go` checks authorization/provider availability before enabling the local connection. Native `messaging_explanation.dart` and web `MessagesWorkspace.svelte` disclose the provider boundary. | Enabling messaging does not set a new restricted incoming policy. No live opt-in journey was performed. |
| Incoming choice | `chat/declaration.go` supports everyone/following/nobody. Missing declaration reads as following; an existing record is respected. Local tests verify conflict-safe updates/readback. | Fallback is not a persisted declaration or provider-enforcement proof. Existing Everyone is not silently narrowed; no universal connected-only default is established. |
| Requests | `chat/details.go` returns history for a request with `CanSend=false`; its test explicitly returns synthetic request text. Native conversation UI displays history; web displays request previews/history and explains reading does not accept. | Acceptance gates replies, not initial text exposure. No live client journey was run. |
| Rich previews | Native `chat_conversation_page.dart` adds `ChatMessagePreview` only for accepted conversations. Web `MessagePreview.svelte` resolves targets only when accepted. | Limits automatic rich previews, not raw text/links or a user's choice to open them. Client behavior source-inspected only. |
| Blocks | Existing local test confirms either-direction blocks prevent chat effects while preserving accessible history; `send.go` checks acceptance/block state before sending. | Does not erase history, prevent alternate accounts or control independent clients/PDS copies. Live cross-client verification pending. |
| Age | `api/onboarding.go` accepts minimum-age confirmation/policy version, not an under-18/adult distinction. The strict test rejects extra DOB/age-band fields. Reviewed `api/profile_store.go` profile/account summaries have no structured age field. | No account-level child classification established in these paths. Unknown age is not evidence of adulthood; this is not an inventory of all possible provider signals. |
| Discovery | Descriptions/photos may voluntarily reveal age. Reviewed graph routes expose own followers/following and another account's mutual followers. | No child label/age-filtered discovery identified. Do not describe these routes as unrestricted full follower lists for every other account. A child in a photo may not be the account holder; voluntary disclosure can still identify an actual child account. |
| Reports | Provider-only report boundary remains as recorded in the children's assessment. | No report sent or provider performance measured; no duplicate local route or DM Brevo alert authorized. |

## Focused Local Verification

Ran against isolated PostgreSQL 16 with migrated schemas and synthetic/mocked
upstream responses:

```text
go test ./internal/chat -run 'Test(IncomingPolicyMergesLatestDeclarationAfterConflict|DetailsLoadsConversationAndHistoryOnce|DetailsFailsClosedBeforePrivateHistory|EitherDirectionBlockPreservesReadAndPreventsEffects|ReadNamesPresentedMessageWithoutAcceptingRequest|ConnectionToggleRequiresCurrentChildAndOptionalAuthority)$' -count=1
```

PASS: `social.craftsky/appview/internal/chat`, 27.624 seconds. Six named top-level
tests plus subtests ran. The first attempt was refused by the test safety guard
because the temporary database lacked a test-name marker; rerun used
`craftsky_contact_review_test`. Temporary container removed. No production
database, real message, provider setting or report changed; no application
changes or new tests were made.

`Child` in the connection/auth test names refers to a child session token, not
age. These tests establish authorization boundaries, not child-age protection.

## Applicability and Risk Consequence

Current ICU F1/F2 have age-determination and relevant functionality/risk
conditions. No reliable under-18 account distinction is established here. Leave
applicability unresolved pending the wider review; do not automatically apply
every measure or assume that no child duties apply.

This evidence narrows the earlier discovery claim and credits actual partial
protections. It does not establish strong evidence to lower the provisional
High grooming rating. The subsequent owner-authorized overall CSEA judgment
is High, recorded in version 0.4 of `illegal-content-risk-assessment.md`. This
control review supplies evidence for that judgment; it does not establish live
effectiveness or whole-assessment approval.

## Deferred Benign Verification After Merge/Deployment

Use controlled accounts, no real child and no harmful material. Confirm
fresh-account and existing-setting provider enforcement for each incoming
choice, following relationships, request text/preview behavior, accepting versus
reading, blocks and native/web presentation. Check shared setting changes and
readback. Provider-only report delivery requires a separately planned benign
exercise. No real outbound exercise was performed here. Record actual results
and limitations before crediting live efficacy.

Source: [Ofcom issued U2U Illegal Content Code, ICU F1/F2](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/upcoming-codes-of-practice/issued-illegal-content-codes-of-practice-for-user-to-user-services.pdf?v=427438),
checked 9 November 2026; see the illegal-content assessment for risk guidance.
