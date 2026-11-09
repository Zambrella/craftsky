# Reporting Mailbox Readiness

| Field | Value |
|---|---|
| Owner | Douglas Todd |
| Recorded | 9 November 2026 |
| Status | Routine external reporting exercise passed, owner-reported; specialized/urgent checks pending |
| Publication | Internal |

## Confirmed Evidence

During launch-readiness review Douglas Todd was asked whether an ordinary
external email reaches `moderation@craftsky.social` and whether he can reply from
that address. He answered “yes, it works”. Record this as owner-reported
verification of external mailbox receipt and replies, including a route that
does not require CraftSky sign-in. The assistant did not independently send an
email or inspect mailbox delivery records. No sender address, message content,
credentials or private correspondence is stored here.

## Separate Operational Checks

Mailbox delivery alone does not verify that a report is acknowledged, assigned,
recorded in the moderation console or handled within its applicable deadline.
The current platform supports manual external intake, safe message references,
urgency and attachment disposition. It does not automatically ingest the inbox.

Douglas Todd approved the following manual launch process on 9 November 2026:

- triaging emails on receipt, with urgent/time-critical concerns handled ahead
  of ordinary acknowledgements;
- recording actionable reports as external intake in the moderation console,
  linking/creating the appropriate case or restricted workflow;
- acknowledging ordinary reports with a reference within the draft policy's
  five-UK-business-day target, without delaying urgent action;
- keeping raw emails in the approved mailbox and restricted content out of
  ordinary notes, email forwarding and logs; and
- recording follow-up and outcomes without exposing reporter identities.

The owner reports that the routine end-to-end exercise below passed. Do not infer completion
of representative/pseudonymous reporting, legal declarations, urgent escalation,
child-appropriate usability or restricted-evidence handling from the mailbox
receipt/reply confirmation alone. REPORT-01 and REPORT-02 remain open for their
remaining workflow evidence.

## Benign Intake Exercise

Run once on an owner-controlled benign post, using ordinary mailbox delivery.
Keep this labelled as a launch-readiness test with no alleged violation.

1. Email `moderation@craftsky.social` from an external address with subject
   `Launch-readiness test — no action required`, identify the owner-controlled
   post, explain this is a test, and include no attachments.
2. In the moderation console's **External intake** workspace, use **Record
   external intake**. Record the email's message reference and original receipt
   time, the post's canonical AT URI, kind `allegation`, urgency `routine`, and
   attachment handling `none`. The corrected intake accepts shared post/profile/event links from
   `app.craftsky.social` and stores their canonical AT URI. Until that fix is
   deployed, use an AT URI because production may reject the app-host HTTPS link.
3. From that intake choose **Create case from intake**. Use the same owned post
   and its author's DID. Record the reason as a launch-readiness exercise with
   no alleged violation. Creating the case must not impose a consequence.
4. Reply by email with the intake reference and confirmation that the test was
   received; retain the sent acknowledgement in the mailbox. Send one harmless
   follow-up from the external address, then record that incoming message using
   **Add correspondence**, its message reference and original receipt time.
5. Resolve the test case with **No action**; do not issue a warning, strike,
   suspension or content restriction. Record the test outcome in the internal
   readiness evidence and verify the post remains visible.

Record whether intake creation, case linkage, acknowledgement/correspondence and
no-action resolution succeeded, with the exercise date and any failure. Keep
raw emails and actual contact details in the mailbox rather than this document.
This routine exercise does not establish urgent legal-deadline handling; that
requires a separate benign tabletop exercise. Douglas Todd subsequently reported
that he went through the full flow and it worked on 9 November 2026.

## Exercise Finding — Shared-Link Rejection

On 9 November 2026 Douglas supplied a screenshot of an external-intake rejection
with an ordinary `https://app.craftsky.social/posts/...` shared link. The current
validator rejected that host. HTTPS references also needed normalization for the
subsequent exact AT URI case-match check.

The failure was reproduced locally before editing. The local fix normalizes
recognized post/profile/event links and focused reply links to canonical AT URIs,
retains exact case-subject matching and rejects untrusted hosts/unsupported
routes. Previously stored recognized HTTPS links can be matched without rewriting
historical intakes. Unsupported input now receives a targeted console error.

Verification: all safetyintake, moderation, routes and safetyalerts Go suites
passed against an isolated PostgreSQL 16 database; Go vet passed. All 53 console
tests and its production build passed. The authenticated route regression covers
shared-link intake, persisted canonical URI, matching case creation and rejection
of an unrelated case target. Local checks do not prove production deployment.
Douglas Todd subsequently confirmed that the full flow worked using the AT URI
workaround on 9 November 2026. This verifies the owner-run routine intake flow,
not deployment of the HTTPS-link fix. No case, sanction, content restriction or
external email was created in production by the assistant.

The form's message reference is the email's unique Message-ID or another stable
mailbox reference. The email subject alone can be reused and should not serve
as the message identity.

## Owner-Reported Exercise Result — 9 November 2026

After receiving the AT URI workaround for the rejected shared link, Douglas Todd
confirmed: “I went through the full flow and it worked.” Record the described
routine exercise as passed: external email intake, linked moderation case,
acknowledgement/reply and no-action resolution with the benign post remaining
visible. This is an operator-reported result; the assistant did not inspect the
live case or mailbox. Separate follow-up correspondence, specialist declarations,
urgent deadlines, child-appropriate usability and restricted-evidence exercises
are not independently established by this confirmation.

The approved five-UK-business-day acknowledgement target is supported by a
successful routine acknowledgement exercise, not by a measured ongoing service
history. Urgent workflows remain the next verification step. The HTTPS-link fix
still requires deployment.


## In-App Routine Estimate — 9 November 2026

Under the owner's narrowed post-submit-copy decision, native/web confirmations
now state that CraftSky aims to decide routine reports within ten UK business
days and prioritizes urgent safety concerns. Treat this as the communicated
routine handling aim, measured from receipt, not permission to defer urgent or
specialist action. The five-UK-business-day manual acknowledgement procedure
above remains unchanged. No automatic reporter correspondence or report-status
UI is introduced; report receipt is not a completed moderator review.
