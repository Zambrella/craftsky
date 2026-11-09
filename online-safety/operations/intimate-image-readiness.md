# Intimate-Image Response Readiness

| Field | Value |
|---|---|
| Owner | Douglas Todd |
| Last updated | 9 November 2026 |
| Status | Internal target approved; production verification pending |
| Publication | Internal |

## Approved Target and Current Evidence

Douglas Todd explicitly approved a 24-hour internal response target from the
original report receipt on 9 November 2026, with action as soon as practicable
and separately applicable statutory requirements. This approval is recorded in
`online-safety/governance/launch-approval.md`; it is not approval of the full
intimate-image operating process or evidence of compliance with every legal duty.

The checked-in Blueprint and production environment example specify
`SAFETY_INTIMATE_IMAGE_RESPONSE_WITHIN=24h`. No live configuration or deployment
has been verified in this review. The service retains an explicit configured
policy rather than an implicit default.

`appview/internal/safetyincident/workflow.go` reads the saved receipt time when
opening a linked intake review. The form displays that linked time as read-only;
an unlinked review requires entry of the original receipt time. The local
regression uses an intake received 26 hours earlier and verifies that a 24-hour
target is already overdue rather than restarting the clock at review creation.
See `appview/internal/safetyincident/intimate_receipt_test.go` and
`docs/operations/moderation-operators.md`.

The workflow records standing, declarations, judgment, same-image search,
substantially-same search, action, any exception and outcome. Recording these
fields does not perform a search or enforce a visibility decision. Real matching
and action evidence must be recorded separately. No universal video scanning or
network-wide removal is established by this workflow.

## Pending Benign Operational Exercise

Perform after the merged release and configuration are deployed. This is a
verification plan, not a record of a passed test or approval of new procedures.
Use a benign test post and explicitly synthetic complaint; never upload intimate
or illegal material for testing.

1. Record an external intake with its actual original receipt time and a clear
   test reference. Review the standing/declaration fields using synthetic facts.
2. Start its linked intimate-image review. Confirm the saved receipt time is
   preserved and the due time is original receipt plus 24 hours, accounting for
   the displayed timezone.
3. Exercise and record the available same/substantially-same search process with
   benign matching variants. Record coverage and limitations, including video.
4. Use the appropriate moderation action on the benign test subject and verify
   the resulting visibility through the relevant client/API surfaces. Recording
   an outcome code alone is insufficient. Restore benign test visibility through
   the supported procedure when appropriate and record that action.
5. Record judgment, searches, actual action, any assessed exception and outcome;
   verify attributed history and resolution. Review how an overdue item is
   recognized without creating misleading historical receipt data in production.
6. If urgent Brevo alerts are activated, separately verify a generic workflow
   alert reaches the mailbox/phone. This is not a deadline reminder service.

Mailbox handling, actual visibility action, matching coverage, deadline response
and approved retention/evidence handling remain outstanding. Evidence storage
connection and hold/deletion tests are separately deferred until after merge in
`docs/operations/restricted-evidence-storage.md`.

## Record Integrity

On 9 November 2026 this file was found to contain duplicated launch-approval
text rather than a dedicated readiness note. It was replaced with this focused
record from the current source, operator runbook and recorded target approval.
The canonical approval history remains in the governance launch-approval record;
no test result or additional approval has been inferred from the correction.
