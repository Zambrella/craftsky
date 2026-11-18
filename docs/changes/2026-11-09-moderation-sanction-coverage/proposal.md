# Moderation sanction coverage proposal

Status: owner approved the proposal and implementation on 9 November 2026.
Deployment, a real sanction and policy publication remain outside this approval.

## Verified Problem

`appview/internal/moderation/policy.go` permits `other` to warn/hide/take down,
but rejects strikes and severe suspension for that reason. This leaves no direct
account-sanction route for some supported illegal-content findings, including
unlawful supply or trafficking where another named reason does not fit.

`fraud_or_scam` and `self_harm` already support strikes and therefore the existing
strike-threshold suspension path. They are outside the immediate severe-
suspension allowlist. Relabelling those findings as impersonation/threats solely
to obtain severe suspension would make the decision record inaccurate.

Named serious reasons such as child safety, sexual violation, threats,
stalking/privacy, harassment and hate already support serious sanctions. This
is a coverage gap, not an absence of moderation or inability to remove content.

Verified with existing pure policy tests (PASS, 0.317 seconds):

```text
go test ./internal/moderation -run 'Test(ValidateDecision|EveryNamedReasonExceptOtherMayExplicitlyIssueStrike|OnlyApprovedSevereReasonsMaySuspend|SignalsNeverExposeEffectsWithoutSupportedHumanDecision)$' -count=1
```

No source changed and no production action occurred during this review.

## Recommended Policy Decision

1. Add an internal decision reason `illegal_content`, displayed as **Other
   illegal content**. Use an existing specific reason when it accurately fits;
   the new reason covers supported illegal-content judgments without a suitable
   specific reason. It is separate from the broad report/decision reason Other.
2. Require a human finding, factual evidence, an explicit assessed legal
   classification (`otherIllegalContent` for this general route), a separate
   non-empty internal illegal-content rationale identifying the offence/basis
   and reasoning, and a safe explanation for the account owner. A reporter's
   selection, number of reports or scanner signal is not the legal finding.
3. Permit proportionate visibility action and strikes for the new reason. Permit
   severe suspension only with a separate recorded severity rationale explaining
   the serious harm, intent or facilitation and why immediate account restriction
   is proportionate. Preserve existing permissions, review, case revisions,
   audit, appeal and reversal behavior.
4. Add `fraud_or_scam` and `self_harm` to the severe-suspension eligibility list,
   subject to the same human evidence/severity review. Examples are a deliberate
   serious fraud operation or deliberate encouragement/assistance of serious
   self-harm. Distress, recovery/help-seeking, an ordinary transaction dispute
   or an unsubstantiated accusation are not grounds by themselves.
5. Keep `other` ineligible for strikes/severe suspension. Do not automatically
   upgrade an Other report to illegal content, impose sanctions from report
   counts, or treat every Guidelines breach as a criminal offence. Assess
   proportionality and context rather than automatically selecting the maximum
   consequence.

This is a proposed CraftSky policy response to identified risks. It is not a
claim that the law mandates these exact reason names, every offence requires
immediate suspension, or this change alone completes compliance.

## Approved Implementation Scope

- Server validation, database reason constraint and private rationale storage;
  reject missing/unassessed or incompatible classification for the new route.
- Existing admin decision API, documented JSON schema and console form/review/
  history; show the legal-rationale requirement only when relevant and explain
  that the moderator must make the supported judgment.
- Owner-facing reason labels and safe explanations in native/web moderation
  history, including existing fallbacks. Keep private evidence, legal rationale,
  reporter identities and legal classifications off owner/public responses.
- Align Guidelines/operator instructions and the gap register with the approved
  capability. Policy wording remains draft until separately approved/published.
- Use a new migration; do not rewrite applied production migrations. No Lexicon
  changes, new public report flow, PDS deletion, DM-report destination changes,
  production configuration or deployment are included.

## Required Verification

- New reason accepts a valid reviewed decision and rejects missing factual
  evidence, legal rationale, safe explanation or assessed classification.
- Severe decisions for the new/fraud/self-harm reasons require a severity
  rationale; benign/no-action/Other cases cannot bypass those guards.
- Database/API/console persist and present the exact chosen reason and both
  internal rationales. Stale revisions/replays retain existing protections.
- Safe owner responses disclose the explanation/reason without internal notes,
  legal assessment, sensitive evidence or reporter-routing data.
- Existing strike threshold, permissions, effects, appeals and reversals remain
  correct. Client labels handle the added reason on native and web separately.

Source context: [Ofcom illegal-content duties](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/illegal-content-duties-under-the-online-safety-act)
requires proportionate protection and response; final measure applicability and
whole-assessment approval remain separate.
