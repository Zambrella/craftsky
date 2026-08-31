# Requirements: Onboarding Experience

## 1. Initial Request
Replace the placeholder onboarding screen with a full-screen, skippable, multi-step experience. The steps collect a profile picture, name, bio, craft preferences, and optionally connect Instagram. The Instagram settings content should be reusable by both onboarding and settings. Each step has a bottom primary action whose behavior reflects whether the step has unsaved changes.

The user is open to small additions that make onboarding clearer or more useful.

Follow-up: profile identity fields should be pre-filled from available Bluesky profile data, which is expected to have been indexed by the AppView before onboarding.

Grilling follow-up: onboarding completion must be account-wide AppView state rather than device-local state. The Instagram step is limited to linking, import, and inline suggestions; navigation, retry, concurrency, and responsive behavior are resolved below.

OAuth projection follow-up: when OAuth finds an existing `app.bsky.actor.profile`, the callback should project that fetched record directly into the AppView database before handing the session to Flutter rather than relying solely on Tap indexing. Projection failure is best-effort and falls back to the existing Tap/backfill path without failing sign-in.

## 2. Current Codebase Findings
- Relevant files: `app/lib/onboarding/pages/onboarding_page.dart`, `app/lib/onboarding/providers/onboarding_status_provider.dart`, `app/lib/router/router.dart`, `app/lib/profile/pages/edit_profile_dialog.dart`, `app/lib/profile/widgets/edit_profile_crafts_picker.dart`, `app/lib/profile/providers/save_profile_provider.dart`, and `app/lib/instagram_migration/pages/instagram_migration_page.dart`.
- Existing patterns: Riverpod providers own mutations and account-scoped state; profile edits use a full-screen form with dirty tracking, validation, image upload, and an account-operation guard; craft selection already has a reusable picker; the Instagram page owns account verification, discoverability, imports, and suggestions in one large page implementation.
- Current behavior: signed-in accounts whose local per-DID onboarding flag is false are redirected to `/onboarding`. The current onboarding page only displays a `Finish` button. Finishing stores `onboarded_<did>` in `SharedPreferences` and the router then redirects to home. `GET /v1/profiles/me` reads the active CraftSky profile and joins its indexed `bluesky_profiles` row, exposing available Bluesky display name, description, avatar, and banner data through the existing Flutter `Profile` model.
- Constraints discovered: profile records are atomic atproto records, so every profile save must preserve and send the full desired profile state rather than a field-level patch. Unknown craft identifiers must not be lost. Image upload and profile-save failures must remain recoverable. Profile writes currently use full, last-write-wins snapshots without client preconditions. Onboarding completion is currently device-local and must move to private AppView persistence. The OAuth callback already fetches the optional Bluesky profile record and CID but discards them; the canonical `BlueskyProfile` indexer already provides idempotent parsing/upsert behavior that the direct projection path should reuse. Tap backfill remains necessary as resilience.
- Test/build commands discovered: `just app-analyze`; focused Flutter tests can be run from `app/` with `flutter test <test-path>`.

## 3. Clarifying Questions And Decisions
### Q1: What should the bottom action do when the current step has unsaved changes?
Answer: Save and advance.

Decision / implication: A dirty, valid step shows `Save & next` (or `Save & finish` on the final step when applicable), persists the change, and advances only after the save succeeds. A clean intermediate step shows `Next` and advances without a write.

### Q2: Should Skip confirm before discarding unsaved changes?
Answer: No. Skip should be immediate.

Decision / implication: Activating `Skip` immediately discards any unsaved onboarding draft, preserves changes already saved in prior steps, optimistically marks onboarding complete for the active DID, and exits the flow without a confirmation dialog.

### Q3: Should onboarding restore the last visited step after an app restart?
Answer: No.

Decision / implication: Until onboarding is completed or skipped, reopening the app starts onboarding at step 1. Successfully persisted profile or Instagram data is pre-filled normally.

### Q4: Should onboarding analytics be included or planned now?
Answer: No.

Decision / implication: Analytics events, metrics, and related privacy-policy work remain outside this change with no current follow-up requirement.

### Q5: What belongs on the Instagram step?
Answer: Account linking, both import methods, and inline suggestions.

Decision / implication: Reuse composable Instagram sections for verification/link status, discoverability, reactivation, export/manual import, and suggestions with follow, dismiss, and load-more actions. Do not include import history, revocation, or navigation from a suggestion to its profile.

### Q6: Can Instagram activity block finishing onboarding?
Answer: No. `Finish` remains available while Instagram verification, import, or suggestion actions are active.

Decision / implication: Instagram is optional. Already-submitted server work may continue, but external verification or processing cannot trap the member in onboarding.

### Q7: What happens to unsaved drafts during step navigation?
Answer: Preserve them within the current onboarding session.

Decision / implication: Backward and forward navigation restores unsaved profile and craft drafts in memory. `Skip` and process restart discard them.

### Q8: Can the progress indicator navigate between steps?
Answer: No. Navigation is sequential.

Decision / implication: The indicator is informational. Forward movement uses the bottom action so dirty steps cannot bypass save-and-advance behavior.

### Q9: Where is completion persisted?
Answer: Private, account-wide AppView state.

Decision / implication: Add authenticated AppView read/write API support and private Postgres persistence. The server state is permanent and unversioned, applies across devices and reinstalls, and is included in account-deletion cleanup.

### Q10: What happens if completion persistence fails?
Answer: Exit optimistically and retry only in memory.

Decision / implication: `Skip` and `Finish` immediately release the current router gate, retry the AppView write silently while the process remains alive, and do not maintain a durable local retry marker. If all retries fail before process exit, onboarding can reappear at the next cold start.

### Q11: What happens if completion status cannot be read at cold start?
Answer: Show a retryable initialization gate.

Decision / implication: The app does not guess complete or incomplete state when the AppView status read fails.

### Q12: How should platform Back behave?
Answer: Steps 2 and 3 go to the previous step; step 1 stays in onboarding.

Decision / implication: System and app-bar Back never imply `Skip`. The explicit `Skip` action is the only onboarding exit before completion.

### Q13: How is progress presented?
Answer: A localized `Step X of 3` label plus a non-interactive linear progress bar.

Decision / implication: The presentation is compact, clear, and does not imply tappability.

### Q14: Which existing profile and craft controls are reused?
Answer: The current 22-item localized craft chip grid and gallery-only avatar picker. Step 1 also shows the active `@handle` as read-only context.

Decision / implication: Camera capture, avatar removal, banner editing, craft search/grouping, and account switching/sign-out controls are not added to onboarding.

### Q15: How is delayed Bluesky prefill handled?
Answer: Retry automatically for up to 5 seconds, once per onboarding session.

Decision / implication: If Bluesky-backed fields remain absent after the bounded wait, show editable empty optional fields. A failed AppView profile read remains a retryable error rather than becoming a blank writable snapshot.

### Q16: What controls remain available during async work?
Answer: Disable `Skip`, Back, and duplicate submission only after a profile save has been committed and while it is in flight. `Finish` and Back remain available during Instagram work.

Decision / implication: Submitted profile writes cannot be abandoned ambiguously; optional Instagram activity never traps the member.

### Q17: How are concurrent profile edits handled?
Answer: Keep the existing loaded-snapshot, last-write-wins semantics.

Decision / implication: This change does not add client-supplied record preconditions, conflict UI, or a pre-save refresh/merge. Existing AppView read-before-write and `ExpectedCID` behavior remains intact; the narrow stale-client-snapshot race is documented as residual risk.

### Q18: What happens if direct OAuth profile projection fails?
Answer: Continue sign-in and fall back to Tap/backfill.

Decision / implication: A successful OAuth profile fetch is projected before handoff creation when possible, but database projection failure is logged without profile content and does not fail OAuth. The fetched CID and canonical indexer behavior make a later Tap event idempotent.

## 4. Candidate Approaches
### Option A: One onboarding form with a final bulk save
Summary: Hold profile and craft changes locally across all steps, then save everything at completion.

Pros: Fewer profile writes; users can revise all choices before committing.

Cons: Conflicts with the requested per-step save behavior; increases loss if the app exits; complicates the independently stateful Instagram flow.

Risks: Draft state and uploaded images can become stale or orphaned, and a final failure blocks completion of otherwise valid earlier steps.

### Option B: Step-scoped saves with shared feature widgets and account-wide completion
Summary: Each editable step starts from the current account state, retains its draft for the in-process flow, saves its complete profile snapshot when dirty, and advances after success. Composable Instagram linking/import/suggestion sections are reused by onboarding and settings. Completion is private AppView state.

Pros: Matches the requested interaction; reuses existing providers and validation; failures remain local to one step; settings and onboarding cannot drift as easily.

Cons: May perform more than one profile write; shared widgets need clear ownership of layout, loading, and account state; server-backed completion requires an API route, migration, startup gate, and retry behavior.

Risks: Incorrectly composing partial profile values could clear unrelated profile fields; provider state could leak across account switches if not lease-scoped.

## 5. Recommended Direction
Recommended approach: Option B, step-scoped saves with shared profile/craft controls, composable Instagram sections, and account-wide private completion state.

Why: It directly implements the confirmed save-and-advance behavior, builds on current mutation and account-safety patterns, minimizes duplicated behavior between onboarding and settings, and prevents onboarding from repeating merely because a member changes devices. Add the resolved progress, back-navigation, prefill, and privacy behavior, but do not expand onboarding into unrelated preference collection.

## 6. Problem / Opportunity
New CraftSky members currently encounter a placeholder gate that provides no help creating an identifiable profile, expressing craft interests, or finding people they already know. A short optional setup flow can improve profile quality and discovery while preserving the member's ability to enter the app immediately.

## 7. Goals
- G-001: Help a newly signed-in member create a recognizable profile.
- G-002: Capture craft preferences that can be displayed and used by existing product surfaces.
- G-003: Offer the existing Instagram connection and migration capability during initial setup without duplicating it.
- G-004: Keep onboarding optional, understandable, recoverable from errors, and safe across account changes.
- G-005: Make available Bluesky identity data visible to the AppView as early as possible during OAuth without making database projection a new sign-in availability dependency.

## 8. Non-Goals
- NG-001: Changing atproto lexicons, public profile record shapes, the craft catalog, or Instagram backend behavior.
- NG-002: Adding feed ranking, recommendation preferences, notification permission prompts, contact import, account creation, or identity/handle changes.
- NG-003: Redesigning all profile settings or Instagram migration functionality.
- NG-004: Persisting the current onboarding step or unsaved drafts across process restarts.
- NG-005: Requiring any profile field, craft selection, or Instagram connection before entering the main app.
- NG-006: Versioning onboarding completion or forcing completed members through future onboarding revisions.
- NG-007: Adding camera capture, avatar removal, banner editing, account switching/sign-out, suggestion-to-profile navigation, import history, or Instagram revocation to onboarding.
- NG-008: Adding client-supplied profile concurrency control, pre-save refresh/merge, or conflict-resolution UI; existing AppView `ExpectedCID` behavior remains intact.
- NG-009: Removing Tap tracking/backfill or making direct OAuth projection the sole source of Bluesky profile indexing.

## 9. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Newly signed-in member | A signed-in account whose private AppView onboarding completion state is incomplete | A short, optional setup path with clear progress and recoverable saves |
| Returning/settings member | A member opening Instagram migration from settings | The same Instagram behavior and state presentation available outside onboarding |
| CraftSky client | The Flutter app coordinating account state, profile writes, and onboarding completion | Account-safe operations that preserve existing profile data |

## 10. Current Behavior
The router forces a newly signed-in account to the full-screen onboarding route until its local completion flag is set. The page contains no setup controls and only marks onboarding complete. Profile identity and crafts are edited together in a separate full-screen dialog. Instagram migration is a separate settings page whose content is private to that page implementation.

## 11. Desired Behavior
A newly signed-in member sees a three-step, full-screen onboarding flow:

1. Review or update profile picture, display name, and bio, pre-filled from the active account's Bluesky profile data indexed by the AppView when available.
2. Select crafts from the existing craft catalog.
3. Connect or reactivate an Instagram account, choose discoverability, import followed accounts by export or manual entry, and act on inline follow suggestions. Existing linked state is shown when already connected; import history, revocation, and profile navigation remain in settings/the main app.

The app bar offers a text `Skip` action for the whole flow and communicates progress with a label and linear bar. Sequential Back/forward navigation preserves in-session drafts. A persistent bottom action advances immediately when the step is clean, or saves and advances when dirty. The final action optimistically completes onboarding and enters the main app while private account-wide completion syncs to the AppView. Composable Instagram sections are shared with settings rather than duplicated.

## 12. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Onboarding shall help a new member establish profile identity, craft interests, and an optional Instagram connection before entering the main app. | Replaces the non-functional placeholder with useful setup. | Prompt | AC-001, AC-002 |
| BR-002 | Business | Must | A member shall be able to skip the entire onboarding process without supplying or changing any optional information. | Onboarding must not prevent product access. | Prompt | AC-003 |
| FR-001 | Functional | Must | The system shall present onboarding as a full-screen, ordered three-step flow: profile identity, crafts, then Instagram. | Establishes the requested information architecture. | Prompt | AC-001 |
| FR-002 | Functional | Must | The profile identity step shall display the current avatar, display name, and bio and allow the member to choose a profile picture and edit the name and bio. | Lets members create a recognizable profile while supporting pre-existing values. | Prompt, Codebase | AC-004, AC-005 |
| FR-003 | Functional | Must | The profile identity step shall apply the existing display-name and bio constraints and prevent saving while input is invalid, an image upload is running, or an image upload has failed. | Prevents rejected or incomplete profile writes. | Codebase | AC-006 |
| FR-004 | Functional | Must | The crafts step shall display the existing craft catalog, preselect crafts already on the profile, and allow zero or more crafts to be toggled. | Captures preferences without making them mandatory. | Prompt, Codebase | AC-007 |
| FR-005 | Functional | Must | When profile identity or crafts are dirty and valid, the bottom action shall be labeled `Save & next`, save the complete desired profile state, and advance only after a successful save. | Implements the confirmed interaction and protects atomic profile records. | User answer, Codebase | AC-008, AC-009 |
| FR-006 | Functional | Must | When an intermediate step has no unsaved changes, the bottom action shall be labeled `Next` and advance without issuing a profile write. | Avoids unnecessary writes and explains the action. | Prompt, User answer | AC-010 |
| FR-007 | Functional | Must | If a step save fails, the member shall remain on that step with entered selections intact, receive an understandable error, and be able to retry. | Network or server errors must not discard setup work. | Codebase | AC-011 |
| FR-008 | Functional | Must | The Instagram step shall use shared, account-scoped sections for link/verification status, discoverability, reactivation, export/manual import, and inline suggestions; it shall exclude import history and revocation. | Reuses relevant behavior without embedding the entire settings-management page. | Prompt, User answer, Codebase | AC-012, AC-013, AC-024, AC-025 |
| FR-009 | Functional | Must | The final bottom action shall be labeled `Finish`, optimistically mark the active DID complete in the current app process, and allow router navigation to the main app without requiring or waiting for Instagram activity. | Provides a clear optional-flow exit. | Prompt, User answer, Codebase | AC-014, AC-026 |
| FR-010 | Functional | Must | A text `Skip` action shall be available in the app bar on every step and shall immediately discard current unsaved drafts, preserve previously saved changes, optimistically mark the active DID complete, and exit without confirmation, except while a submitted profile save is in flight. | Makes the whole process consistently and immediately skippable without abandoning a committed profile write. | Prompt, User answer | AC-003, AC-015, AC-027 |
| FR-011 | Functional | Should | Sequential navigation shall preserve saved values and unsaved profile/craft drafts in memory for the current onboarding session. | Supports correction without persisting incomplete drafts. | User answer | AC-016, AC-028 |
| FR-012 | Functional | Should | Each step shall show a localized `Step X of 3` label and non-interactive linear progress bar. | Communicates progress without creating a second navigation path. | User answer | AC-017 |
| FR-013 | Functional | Should | The Instagram step shall briefly explain that connection is optional and summarize the privacy purpose of discoverability/import behavior using the existing product disclosures. | Supports informed consent for a sensitive integration. | Recommended direction, Codebase | AC-018 |
| FR-014 | Functional | Must | Before presenting the profile identity step, the system shall load the active member's profile through the existing AppView profile read path and pre-fill every available Bluesky-backed avatar, display name, and bio value; the Flutter app shall not read the PDS directly for this purpose. | Avoids asking members to re-enter profile information they already maintain on Bluesky and preserves the AppView read boundary. | User answer, Codebase | AC-004, AC-023 |
| FR-015 | Functional | Must | The Instagram import section shall support both the existing on-device export picker/parser and manual handle entry. | Preserves both established import paths. | User answer, Codebase | AC-024 |
| FR-016 | Functional | Must | Instagram suggestions in onboarding shall support inline follow, dismiss, and load-more actions but shall not open profile routes. | Provides useful discovery without escaping the onboarding router gate. | User answer, Codebase | AC-025 |
| FR-017 | Functional | Must | App-bar and system Back shall move from steps 2 or 3 to the previous step and shall be intercepted on step 1 without completing or skipping onboarding. | Makes Back deterministic and reserves exit for explicit Skip. | User answer | AC-029 |
| FR-018 | Functional | Must | The AppView shall expose authenticated read and idempotent completion-write operations backed by private, per-DID Postgres state, and Flutter shall use that state as the cold-start onboarding authority. | Makes completion account-wide across devices and reinstalls. | User answer, API architecture | AC-030, AC-031 |
| FR-019 | Functional | Must | After optimistic `Skip` or `Finish`, Flutter shall retry a failed completion write silently while the process and owning account session remain active, without creating a durable local retry marker. | Implements the chosen availability/retry tradeoff without two durable authorities. | User answer | AC-032, AC-033 |
| FR-020 | Functional | Must | If cold-start completion status cannot be read, the app shall show a retryable initialization gate and shall not guess complete or incomplete status. | Avoids incorrectly gating new members or re-onboarding completed members. | User answer | AC-034 |
| FR-021 | Functional | Must | Once per onboarding session, when the profile exists but all expected Bluesky identity fields are absent, Flutter shall retry the AppView profile read for up to 5 seconds before presenting optional empty fields. | Gives eager profile indexing a bounded convergence window without trapping accounts that have no Bluesky profile. | User answer, Codebase | AC-035, AC-036 |
| FR-022 | Functional | Must | While a submitted profile save is in flight, the system shall disable Skip, Back, and duplicate submission; it shall restore applicable controls if the save fails. | Prevents navigation from ambiguously abandoning an uncancellable committed write. | User answer | AC-027 |
| FR-023 | Functional | Must | Step 1 shall show the active `@handle` as read-only context and shall reuse the existing gallery-only avatar replacement flow. | Confirms account identity and avoids new camera permission behavior. | User answer, Codebase | AC-037 |
| FR-024 | Functional | Must | When the OAuth login/registration callback fetches an existing `app.bsky.actor.profile`, it shall attempt to project the fetched record and its CID directly into `bluesky_profiles` before creating the Flutter handoff. | Makes existing identity data available without waiting for Tap indexing. | User answer, Codebase | AC-041, AC-042 |
| NFR-001 | Non-functional | Must | Onboarding shall remain usable without overflow or inaccessible primary actions on supported compact mobile layouts and larger layouts. | The flow is full-screen and must work across Flutter form factors. | Project frontend constraints | AC-019 |
| NFR-002 | Non-functional | Must | All new user-visible onboarding text and semantics shall use the app localization system and expose meaningful accessibility labels, selected states, and disabled states. | Maintains localization and assistive-technology support. | Codebase conventions | AC-020 |
| NFR-003 | Non-functional | Must | Async profile and Instagram operations shall remain bound to the active account so results from a previous account cannot update or complete onboarding for the newly active account. | Multi-account switching is an established safety boundary. | Codebase | AC-021 |
| NFR-004 | Non-functional | Must | Direct OAuth projection and Tap projection shall share canonical Bluesky profile parsing/upsert semantics and remain idempotent for the same `(DID, CID)`. | Prevents duplicate or divergent projection behavior when Tap later delivers the fetched record. | User answer, Codebase | AC-041, AC-043 |
| RULE-001 | Business rule | Must | Profile saves from either onboarding step shall preserve profile fields not edited on that step, including banner data and unrecognized craft identifiers. | atproto profile writes replace the atomic record. | Codebase | AC-009 |
| RULE-002 | Business rule | Must | Avatar, name, bio, crafts, and Instagram connection are optional; empty values or no selection shall not block progression when otherwise valid. | The whole process and all supplied information are optional. | Prompt | AC-002, AC-022 |
| RULE-003 | Business rule | Must | Onboarding status and completion operations shall apply only to the authenticated active DID and owning account session. | Prevents one account's action from onboarding another account. | User answer, Codebase | AC-015, AC-021, AC-031 |
| RULE-004 | Business rule | Must | Server-side completion is permanent and unversioned; completed members shall not be router-gated by future onboarding revisions. | Future education should not force members through onboarding again. | User answer | AC-038 |
| RULE-005 | Business rule | Must | Unsaved drafts and current-step position shall not persist across process restart; an incomplete account shall restart at step 1 with successfully persisted data pre-filled. | Keeps draft persistence intentionally session-scoped. | User answer | AC-039 |
| RULE-006 | Business rule | Must | Profile saves retain the existing loaded-snapshot, last-write-wins concurrency behavior; this change shall not add profile conflict detection or merge behavior. | Keeps concurrency control outside onboarding scope. | User answer, Codebase | AC-040 |
| RULE-007 | Business rule | Must | A missing Bluesky profile remains a successful OAuth condition, and a direct-projection failure shall be logged and shall not fail sign-in or suppress existing Tap tracking/backfill. | Direct projection is an eager optimization, not a new authentication dependency. | User answer, Codebase | AC-042, AC-043 |

## 13. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given a signed-in account has not completed onboarding, when routing settles, then a full-screen flow opens on step 1 of 3 and exposes the profile, crafts, and Instagram steps in that order. |
| AC-002 | BR-001, RULE-002 | Given a member leaves all optional setup values unchanged or empty, when they proceed through the flow, then they can complete onboarding and enter the main app. |
| AC-003 | BR-002, FR-010 | Given the member is on any onboarding step, when they activate `Skip`, then no confirmation is shown, the current unsaved draft is discarded, previously saved changes remain, the whole flow ends, and the main app becomes reachable without requiring remaining steps. |
| AC-004 | FR-002, FR-014 | Given the active profile already has an avatar, display name, or bio, when step 1 loads, then those current values are shown. |
| AC-005 | FR-002 | Given step 1 is ready, when the member chooses an image or edits display name or bio, then the draft state and dirty action update to reflect those changes. |
| AC-006 | FR-003 | Given step 1 has invalid text, an image upload in progress, or a failed image upload, when the page renders, then `Save & next` cannot submit until the issue is resolved and validation or upload feedback is visible. |
| AC-007 | FR-004 | Given step 2 loads, when the member toggles craft chips, then current known crafts begin selected and each chip accurately exposes and updates its selected state, including an empty selection. |
| AC-008 | FR-005 | Given an editable step is dirty and valid, when the member activates `Save & next`, then the button enters a busy/disabled state, exactly one save is attempted, and the next step opens only after success. |
| AC-009 | FR-005, RULE-001 | Given the member changes only identity fields or only crafts, when that step saves, then the resulting profile contains the changed values plus all untouched profile fields and unrecognized craft identifiers from the current profile snapshot. |
| AC-010 | FR-006 | Given an intermediate step is clean, when the member activates `Next`, then the next step opens and no profile mutation is attempted. |
| AC-011 | FR-007 | Given a profile save fails, when the failure is returned, then onboarding stays on the current step, retains the draft, shows an error, re-enables retry when safe, and does not mark onboarding complete. |
| AC-012 | FR-008 | Given the active account has no linked Instagram account, when step 3 loads, then shared content exposes verification, its required discoverability choice, and available import/suggestion sections without import history or revocation. |
| AC-013 | FR-008 | Given the active account is linked or needs reactivation, when step 3 or settings loads, then both surfaces use shared sections to show the applicable linked status, discoverability, and reactivation controls. |
| AC-014 | FR-009 | Given the member is on step 3, when they activate `Finish`, then Instagram linking is not required, current-process status becomes complete immediately, and routing leaves onboarding for the main app. |
| AC-015 | FR-010, RULE-003 | Given two DIDs have separate onboarding states, when one DID skips onboarding, then only the active authenticated DID is optimistically completed and no unsaved onboarding fields are written. |
| AC-016 | FR-011 | Given a step has saved values or an unsaved draft, when the member navigates backward and later returns within the same onboarding session, then both the saved values and that unsaved draft are restored. |
| AC-017 | FR-012 | Given any onboarding step is visible, then a localized label identifies `Step X of 3`, a linear bar represents the same progress, and neither control is tappable. |
| AC-018 | FR-013 | Given the Instagram step is visible, then it states that connection is optional and presents the relevant existing privacy/discoverability disclosures before the member confirms linking or import choices. |
| AC-019 | NFR-001 | Given a supported compact mobile viewport or larger viewport, when each step is rendered with keyboard, loading, validation, and long-content states, then content can be reached by scrolling and the app-bar and bottom actions remain operable without layout overflow. |
| AC-020 | NFR-002 | Given a supported locale and assistive technology, when onboarding is rendered and operated, then new text comes from localization resources and controls announce their purpose, progress, selection, busy, and disabled states as applicable. |
| AC-021 | NFR-003, RULE-003 | Given an async operation belongs to account A, when the active account changes to B before it completes, then A's result cannot alter B's profile, Instagram UI, or onboarding completion state. |
| AC-022 | RULE-002 | Given the member has no avatar, display name, bio, crafts, or Instagram link, when each step is clean and valid, then progression and completion controls remain available. |
| AC-023 | FR-014 | Given the AppView has indexed Bluesky avatar, display name, or bio values for the active DID, when onboarding step 1 loads, then each available value is pre-filled from `GET /v1/profiles/me`; no client request is made directly to the PDS. |
| AC-024 | FR-008, FR-015 | Given Instagram import is available on step 3, then the member can either select a supported Instagram export for on-device parsing or enter handles manually, with the existing validation, disclosure, success, and error behavior. |
| AC-025 | FR-008, FR-016 | Given Instagram suggestions are available, when the member interacts with them, then follow, dismiss, and load-more work inline and tapping a suggestion does not open a profile route. |
| AC-026 | FR-009 | Given verification, import, or suggestion activity is pending or busy, when step 3 renders, then `Finish` remains operable and leaving onboarding does not cancel already accepted server work. |
| AC-027 | FR-010, FR-022 | Given a profile save has been submitted, while it remains in flight, then Skip, Back, and duplicate submission are disabled; after failure they become applicable again, and after success the flow advances once. |
| AC-028 | FR-011 | Given the member restarts the app or activates Skip, then unsaved in-session drafts are absent when onboarding is next opened. |
| AC-029 | FR-017 | Given step 2 or 3 is visible, when app-bar or system Back is invoked, then the previous step opens; given step 1 is visible, Back is intercepted and onboarding remains open without being completed. |
| AC-030 | FR-018 | Given completion has been persisted for a DID, when that DID signs in on another device or after reinstall, then the authenticated AppView status read reports complete and the onboarding route is not shown. |
| AC-031 | FR-018, RULE-003 | Given an authenticated account reads or writes onboarding status, then the AppView stores and returns only that DID's private completion state and no client-supplied DID can target another account. |
| AC-032 | FR-019 | Given Skip or Finish is activated, when the first AppView completion write fails, then the member remains in the main app, no failure message is shown, and the client retries while the owning process/session remains active. |
| AC-033 | FR-019 | Given all in-memory retries fail and the process exits, then no durable local pending-completion marker remains and a later cold start follows the server's still-incomplete state. |
| AC-034 | FR-020 | Given cold-start onboarding status loading fails, then a retryable initialization gate is shown and neither onboarding nor the main app is selected until status is known. |
| AC-035 | FR-021 | Given the profile exists but all Bluesky identity fields are initially absent, when step 1 initializes, then AppView profile reads are retried with bounded backoff for no more than 5 seconds and only once in that onboarding session. |
| AC-036 | FR-021 | Given the bounded prefill wait expires without Bluesky identity fields, then optional empty fields become editable and progression remains available; a true profile-read error instead shows retry UI. |
| AC-037 | FR-023 | Given step 1 is ready, then it shows the active `@handle`, offers avatar replacement from the gallery, and does not offer handle editing, camera capture, avatar removal, banner editing, account switching, or sign-out. |
| AC-038 | RULE-004 | Given an account has completed any onboarding revision, when a later client revision is installed, then permanent completion remains true and the router does not force onboarding again. |
| AC-039 | RULE-005 | Given an incomplete member closes the app on step 2 or 3, when they return, then onboarding opens at step 1 with persisted profile/Instagram data pre-filled and no prior unsaved draft. |
| AC-040 | RULE-006 | Given profile data changes in another client after onboarding loads its snapshot, when onboarding later saves, then the existing last-write-wins endpoint behavior applies without a new conflict prompt, client-supplied precondition, or pre-save merge request. |
| AC-041 | FR-024, NFR-004 | Given OAuth fetches a valid Bluesky profile record with CID, when callback initialization succeeds, then the canonical projection logic has upserted its display name, description, avatar/banner metadata, and record CID into `bluesky_profiles` before handoff creation. |
| AC-042 | FR-024, RULE-007 | Given the Bluesky profile is absent or direct database projection fails, when the remaining OAuth initialization succeeds, then sign-in and handoff creation continue; projection failure emits a non-sensitive warning and missing profile emits no projection row. |
| AC-043 | NFR-004, RULE-007 | Given OAuth directly projected `(DID, CID)` and Tap later delivers the same record, when the Tap indexer handles it, then the stored projection remains correct, no duplicate row is created, and existing repository tracking/backfill behavior remains enabled. |

## 14. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Existing partially completed profile | Current values prepopulate; clean steps advance without rewriting them. | FR-002, FR-004, FR-006 |
| EC-002 | Unknown craft identifier exists on profile | It is not shown as a selectable known chip but remains in the saved profile after either step saves. | RULE-001 |
| EC-003 | Image picker is cancelled | The avatar and dirty state remain unchanged. | FR-002, FR-005 |
| EC-004 | Image upload fails or remains in flight | Save-and-advance is blocked; the member can retry or skip the whole flow. | FR-003, FR-010 |
| EC-005 | Profile save fails or connectivity is lost | Draft values remain, completion is unchanged, and retry is available. | FR-007 |
| EC-006 | Instagram integration is unavailable or its state fails to load | The step displays the existing unavailable/error/retry state and still permits `Finish` or `Skip`. | FR-008, FR-009 |
| EC-007 | Instagram was linked before onboarding | The linked account state is shown rather than restarting verification. | FR-008 |
| EC-008 | Active account switches during onboarding or an async operation | State and results from the prior lease do not affect the new account; routing resolves using the new DID's onboarding status. | NFR-003, RULE-003 |
| EC-009 | App closes midway through onboarding | Successfully saved profile/Instagram changes remain; server completion remains false, so onboarding starts at step 1 without unsaved drafts. | FR-005, RULE-005 |
| EC-010 | Text scale, keyboard, or Instagram content exceeds viewport | Content remains scrollable and the primary action remains reachable. | NFR-001 |
| EC-011 | The active DID has no Bluesky profile record or one or more optional Bluesky fields are absent | Available values are pre-filled, absent values remain empty, and progression is not blocked. | FR-014, RULE-002 |
| EC-012 | The AppView profile read fails | Step 1 shows a recoverable loading error and retry path rather than presenting an unverified empty snapshot that could overwrite existing profile data. | FR-007, FR-014, RULE-001 |
| EC-013 | Completion write fails after optimistic exit | The member remains in the main app, the failure is silent, and retries continue only while the owning process/session remains active. | FR-019 |
| EC-014 | Completion status read fails at cold start | A retryable initialization gate is shown; the app does not guess which route to open. | FR-020 |
| EC-015 | Bluesky identity indexing lags | Step 1 waits and retries for at most 5 seconds once per session, then allows empty optional values. | FR-021 |
| EC-016 | Profile save is in flight | Skip, Back, and duplicate submission are disabled until the request settles. | FR-022 |
| EC-017 | Instagram action is in flight | Finish and Back remain available; accepted server work is not treated as an onboarding draft. | FR-009 |
| EC-018 | Suggested account row is tapped | No profile route opens; inline follow/dismiss controls remain available. | FR-016 |
| EC-019 | Another client edits the Bluesky profile during onboarding | The next onboarding save uses existing last-write-wins behavior; no conflict UI is introduced. | RULE-006 |
| EC-020 | OAuth fetches an existing Bluesky profile | The fetched record and CID are projected directly before Flutter handoff creation. | FR-024 |
| EC-021 | OAuth finds no Bluesky profile | OAuth continues without a `bluesky_profiles` row and Tap/backfill remains available. | RULE-007 |
| EC-022 | Direct OAuth projection fails | A non-sensitive warning is logged, OAuth continues, and Tap/backfill may populate the row later. | RULE-007 |
| EC-023 | Tap replays the directly projected CID | Canonical idempotency leaves one correct row without divergent field parsing. | NFR-004 |

## 15. Data / Persistence Impact
- New fields: Private per-DID AppView onboarding completion state, including at least a durable completion timestamp or equivalent permanent flag. No step, draft, completion version, or completion-source analytics field is required.
- Changed fields: Existing profile `avatar`, `displayName`, `description`, and `crafts` values may be updated through existing profile persistence. Existing Instagram state may be updated through existing migration providers.
- Migration required: Yes. Add private onboarding-completion persistence keyed by account DID and register it with existing owner/account-deletion cleanup.
- Backwards compatibility: The app has no production users. Server state becomes the cold-start authority; the existing `onboarded_<did>` `SharedPreferences` key is not retained as a second durable authority. Completion is permanent and account-wide once the AppView write succeeds.
- OAuth projection: No additional profile schema migration is required; direct projection writes the existing `bluesky_profiles` columns using the PDS record CID.

## 16. UI / API / CLI Impact
- UI: Replace the onboarding placeholder with three full-screen sequential steps, a localized label plus linear progress bar, in-session draft preservation, resolved Back/Skip behavior, bottom primary actions, bounded profile-prefill loading, and the selected shared Instagram sections.
- API: Add authenticated `/v1/` operations to read the active DID's onboarding completion status and idempotently mark it complete, following the existing bearer/device headers, camelCase JSON, error envelope, and policy registration conventions. Existing profile and Instagram routes remain unchanged.
- OAuth: Extend login/registration callback initialization to inject a Bluesky profile projector and attempt projection after the existing PDS fetch and before handoff creation. OAuth endpoint shapes do not change.
- CLI: None identified.
- Background jobs: None identified.

## 17. Security / Privacy / Permissions
- Authentication: Onboarding remains available only to signed-in accounts. Cold-start routing waits for authenticated account-wide completion status.
- Authorization: Completion, profile, and Instagram operations must derive the DID from the authenticated session, use active account lease/operation guards, and may affect only that account.
- Sensitive data: Preserve current Instagram disclosures and on-device export parsing behavior. Do not introduce storage of Instagram credentials or PDS tokens on device.
- Abuse cases: Rapid taps must not submit duplicate writes; stale async results after account switching must be ignored.

## 18. Observability
- Events: None. Onboarding analytics are outside the scope of this change.
- Logs: Log non-sensitive completion read/write failures, exhausted in-memory retries, and direct OAuth projection failures for diagnostics. Projection logs must identify operation/result without logging fetched profile content, tokens, or raw records; existing restrictions on profile drafts, Instagram handles, and export contents remain.
- Metrics: None identified.
- Alerts: None identified.

## 19. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | A split-step profile save composes an incomplete client-editable snapshot for an atomic profile update. | Display name, bio, crafts, or other intended values may be cleared; image fields depend on the existing AppView merge. | Require complete editable snapshots plus regression tests for identity-only, crafts-only, AppView avatar/banner preservation, and unknown-craft preservation. |
| RISK-002 | Refactoring the large Instagram page into shared content changes settings behavior. | Existing verification, imports, suggestions, or account controls may regress. | Keep providers and behavior unchanged; add shared-widget parity/regression coverage for unlinked, linked, loading, and error states. |
| RISK-003 | Async completion crosses an account switch. | The wrong account could be updated or marked onboarded. | Retain active lease and operation guards; test stale-result rejection. |
| RISK-004 | Optimistic completion fails and all in-memory retries end before the AppView persists it. | Onboarding reappears at the next cold start despite the member having exited it. | Accept this explicit tradeoff; keep retries account-scoped and silent, and test the cold-start fallback to server state. |
| RISK-005 | Long Instagram content competes with a fixed onboarding bottom action. | Content or controls may be obscured, especially with accessibility text scaling. | Use scroll-safe layout and test compact, large-text, keyboard, and long-content states. |
| RISK-006 | Bluesky profile indexing has not converged before onboarding opens. | Existing profile information may briefly be absent or stale. | Use the authoritative existing AppView profile response, never replace a failed read with a blank writable snapshot, and keep all fields editable and optional. |
| RISK-007 | Server-backed status cannot be loaded during startup. | The app cannot safely choose onboarding or the main shell. | Integrate status with the active-account initialization gate and provide explicit retry UI. |
| RISK-008 | A migration, route, or cleanup omission leaves completion unavailable or retained after account deletion. | Members may be repeatedly gated or private state may outlive the account. | Test migration up/down, authenticated policy, per-DID isolation, idempotent completion, and terminal private cleanup. |
| RISK-009 | A profile is edited by another client while onboarding holds an older snapshot. | A later onboarding save can overwrite the external edit. | Retain and document existing last-write-wins behavior; concurrency control is explicitly outside this change. |
| RISK-010 | OAuth direct projection duplicates or drifts from Tap parsing/upsert behavior. | Fields or CIDs can diverge depending on ingestion path. | Reuse one canonical projection boundary and test direct-then-Tap idempotency on `(DID, CID)`. |
| RISK-011 | Best-effort OAuth projection fails. | Onboarding may still briefly wait for Tap despite the eager path. | Log safely, continue repository tracking/backfill, and retain the bounded Flutter prefill retry. |

## 20. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `Skip` means skip the entire onboarding flow, not only the current step. | Navigation labels and completion behavior would need revision. |
| ASM-002 | Only permanent completion is persisted for onboarding; the current step and drafts are intentionally not restored after process death. | If changed later, a resumable step-state model and persistence tests would be required. |
| ASM-003 | Activating `Skip` with a dirty draft immediately discards unsaved onboarding edits and optimistically exits without confirmation; already successful saves remain. | If changed later, confirmation or save-before-skip behavior would need to be specified. |
| ASM-004 | Shared Instagram UI can be decomposed into linking, import composer, and suggestion sections without changing their provider/service behavior. | A broader Instagram refactor or duplicated onboarding behavior would be required. |
| ASM-005 | Instagram actions persist independently and therefore do not create onboarding dirty state or block `Finish`. | A separate Instagram save/finish contract would be needed. |
| ASM-006 | A 5-second bounded profile-prefill retry is sufficient for ordinary eager-index convergence. | Members may more often see empty optional fields despite having Bluesky data. |
| ASM-007 | The canonical Bluesky index projection can be exposed through an injected interface without coupling the auth package to index implementation details. | A small application-layer adapter would be needed to preserve package boundaries. |

## 21. Open Questions
None identified.

## 22. Review Status
Status: Reviewed
Risk level: Medium
Review recommended: Yes
Reviewer: User
Date: 2026-08-31
Notes: Annotation feedback, grilling decisions, and eager OAuth Bluesky projection have been incorporated. Further technical review is recommended because the change adds private AppView persistence/API behavior, changes OAuth callback initialization, writes atomic public profile records, refactors an established Instagram settings surface, and changes multi-account startup routing. No open product questions remain.

## 23. Handoff To Test Design
- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001, BR-002, FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, FR-010, FR-014, FR-015, FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, FR-022, FR-023, FR-024, NFR-001, NFR-002, NFR-003, NFR-004, RULE-001, RULE-002, RULE-003, RULE-004, RULE-005, RULE-006, RULE-007
- Suggested test levels: widget tests for step rendering, in-session drafts, validation, sequential navigation, Back/Skip locking, progress semantics, bounded prefill, responsive layout, and selected shared Instagram sections; provider/unit tests for optimistic completion, in-memory retry, account ownership, and restart behavior; AppView route/store/migration/cleanup tests for status reads and idempotent completion; OAuth initialization tests for direct projection ordering, missing records, best-effort failure, and direct-then-Tap idempotency; integration tests for save-and-advance, atomic profile preservation, cross-device completion, linked/unlinked Instagram states, and account switching; regression tests for existing OAuth, Instagram settings, Tap backfill, and edit-profile behavior.
- Blocking open questions: None.
