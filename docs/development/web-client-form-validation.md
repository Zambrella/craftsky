# Web client form validation

Field validation uses `web-client/src/lib/field-validation.ts`, installed once by
root layout. Required, URL, email, date and numeric HTML constraints render inline
instead of browser validation popups. Required custom selects validate their
selected value. Invalid controls receive `aria-invalid`, an associated error
message and an error outline; existing helper descriptions remain associated.
The first invalid control receives focus. Editing clears resolved errors, select
changes clear their errors, and closing a dialog clears its validation state.

Custom validators use `FieldValidationError` or the field helpers. API errors with
mapped `fields` entries use the same treatment. Unmapped errors retain general
feedback so failures cannot disappear. Network, permission and uncertain-write
errors retain their existing messages and retry behavior.

## Audited form families

| Forms/fields | Handling |
| --- | --- |
| Post, comment, reply and project composer | Text and record limits inline; project/media/language constraints attached to their field or section; existing async write recovery retained |
| Profile identity and links | Record text limits and URL/title checks inline; server fields map to controls |
| Business details, products and events | Record text, destinations, country, role and date checks inline; native required/type constraints shared |
| Settings languages, Instagram manual import and billing assignment | Custom field checks inline |
| Scheduling | Native required/date constraints and the schedule-window check inline |
| Sign-in | Required handle and rejected handle input inline; authentication/service failures remain general |
| Reports and messages | Native constraints shared; existing message/report validation text associated with invalid fields |
| Saved folders and save-post dialogs | Native constraints shared; “No folder” remains a valid optional selection |
| Search and project filters | Native constraints shared; optional query/filter fields remain optional |
| Account confirmation and preferences | Native constraints shared; existing eligibility and confirmation gates retained |

No backend routes, record schemas or validation limits change. Tests exercise
invalid submission, correction, retained helper text, server field errors, and
continued general feedback for uncertain writes. Existing form suites cover valid
submissions and account isolation.

## Action feedback

Action feedback uses the root `ToastHost` and shared `Snackbar` reporter. Settings,
composer, drafts, messages, post/profile/event actions, moderation, push actions and
sign-in errors share the same presentation. Use `notify` for an accepted action
that immediately navigates away, so its confirmation survives that navigation.

- Success/information toasts dismiss after four seconds, pausing while hovered or
  keyboard focused. Every toast also offers a labelled Dismiss button.
- Action failures take priority and remain until dismissed or resolved. Preserve
  the original retry callback and idempotency key for uncertain writes; dismissing
  feedback must leave those recovery controls available.
- One toast is visible at a time. Pending feedback is bounded and duplicates are
  suppressed. A form updates its existing feedback when another action completes.
- The host follows open native dialogs to remain visible and interactive. Recovery
  controls must sit outside disabled fieldsets, with `type="button"` to avoid
  accidentally resubmitting a surrounding form.
- Confirmations survive same-account navigation; feedback clears on account
  changes. Never retain a retry control after its originating screen is disposed.
- Validation belongs to the field. Loading, account/session availability,
  permission guidance and ongoing delivery/reconciliation states remain contextual.
  Direct toggles such as likes keep their immediate visual state feedback.
