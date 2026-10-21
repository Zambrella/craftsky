# Sentry open-issue review — 21 October 2026

Reviewed all 34 unresolved issues in the CraftSky organization through the authenticated `sentry` CLI, with no date restriction on the inventory. Inspected the latest event, exception frames, release, environment, operation context and breadcrumbs for every issue. Expanded APP-VIEW-8 into its available individual events because it groups unrelated HTTP failures.

Raw events remain outside the repository. No credentials, private payloads, recipient routes or export contents are included here.

## Fixes already in HEAD

These nine issues received next-release resolution requests in Sentry. No new app/backend release or deployment was performed.

Sentry accepted `statusDetails.inNextRelease=true`. Readback stores the existing `craftsky_app@1.8.0+13` as the semantic-version anchor; the resolution activity has an empty target version and `current_release_version` set to that anchor. This is Sentry's [semantic-version next-release representation](https://github.com/getsentry/sentry/blob/master/src/sentry/api/helpers/group_index/update.py), which uses a version comparison rather than a pending named release. The CLI shortcut and project-scoped API were both checked; no future release was invented or created.

| Issues | Evidence and existing fix |
| --- | --- |
| FLUTTER-APP-1F, 1G, 20 | Project-feed continuations reached `applyPostRecordOverlay` after provider disposal. `c00c8c12` checks captured account/provider authority after each awaited fetch, before reading overlays. |
| FLUTTER-APP-1K, 1V | The same disposed-provider failure in authored comments. `c00c8c12` guards both initial and paginated continuations. |
| FLUTTER-APP-1N | Comment-section fetch applied overlays after disposal. `c00c8c12` guards that continuation. |
| FLUTTER-APP-25 | Repost results enumerated optimistic overlays after disposal. `c00c8c12` guards initial, paginated and restarted fetch continuations. |
| FLUTTER-APP-1J | The event's source excerpt explicitly rejects `orderedMedia.length > 4`. `67fd8466` replaces this hard-coded cap with the configured image limit. |
| FLUTTER-APP-1H | The event's source excerpt throws at `decoder.isValid(input)`. `00dba434` removes the overly strict whole-container check, retains signature/dimension limits, and re-encodes appended-data JPEG/PNG inputs. Truly corrupt files still fail; the new static format diagnostics distinguish future failures. |

The Flutter events above came from `social.craftsky.app@1.7.5+12`. Their release labels alone do not establish which changes were in the installed binary; the event's source excerpts and current guards were compared directly.

## Fixes made in this worktree

These six issues remain open until the local changes are integrated into a release.

| Issues | Change |
| --- | --- |
| FLUTTER-APP-1P, 1Q, 23 | The Android file-selector implementation allocates `byte[size]`, reads the complete selected file, and serializes bytes through the message codec. The native crashes occur during activity-result delivery, allocation, or codec buffer growth; one reports a negative array size. Instagram imports now use a dedicated Android document picker that streams with a 64 KiB buffer into a private temporary cache file, returns only its path, parses by path, and removes the copy afterward. Missing streams/copy failures return bounded platform errors; activity destruction stops copying and removes incomplete files. iOS/web retain their existing picker. |
| FLUTTER-APP-1M, 21 | Post-card and notification relationship-seeding microtasks read `WidgetRef` after the widget disappears. The follow-up Riverpod audit removes deferred initialization from widget builds entirely and moves the necessary baseline into the user-triggered mutation. It also covers the analogous profile and reposter paths. |
| FLUTTER-APP-1E | An obsolete onboarding lease is account cancellation, rather than an unexpected `StateError`. Existing lease-specific reads and authorization remain intact. |

## Evidence improvements and issues intentionally left open

| Issues | Disposition and next useful evidence |
| --- | --- |
| FLUTTER-APP-1W, 1Y, 1Z, 22 | Hashtag decoding fails at `postCount`, but both current wire types and fixtures use an integer. No payload or decoder reason survived sanitization, so a schema change would be speculative. Add a static schema-shape explanation (missing/null/noninteger count, invalid tag/item/section), preserving the original mapper cause and stack. Never export the search query, field value, response contents, or mapper prose. Reproduce on a new build and use the selected reason to identify the actual contract violation. |
| FLUTTER-APP-24 | The automatic keyed-write retry budget was exhausted. Preserve the original API/network/format cause and stack through both ambiguity and exhaustion wrappers, with static explanations in serialized diagnostics. Retry policy, frozen command and operation key behavior stay unchanged. The next occurrence should identify the server code/status/request reference when one exists. |
| FLUTTER-APP-1X | Flutter framework semantics geometry failure, with no app-owned frame. Retain the exact static Dart null-check explanation when applicable, and a closed framework-stage value in supporting breadcrumbs/local output. No framework workaround or suppression is justified by this single old-build event. Reproduce on the pinned SDK with its retained stack and navigation timeline. |
| FLUTTER-APP-1R, 1S, 1T | Concurrent notification count/list and scheduled-post requests received generic 502 responses. Existing `c00c8c12` diagnostics distinguish AppView/JSON/non-JSON/empty responses, retain status/method and server-generated request IDs when available. No server request ID exists in the old events. Keep open; correlate a new occurrence with AppView by request ID, or investigate the hosting edge if the request never reaches AppView. |
| APP-VIEW-5, J, P | OAuth identity/provider discovery failures. Existing stage diagnostics distinguish identity, discovery, metadata, authorization request and persistence. Add exact Indigo identity sentinel explanations so not-found, mismatch and resolution failures remain distinguishable without handle input, provider prose or URLs. Existing reviewed provider-code/status selection remains intact. |
| APP-VIEW-F | OAuth provider callback returned an error, but the old diagnostic erased its code. Select a closed OAuth error-code set into the diagnostic reason, retaining existing failure/authorization semantics. Unknown codes and all provider descriptions remain private. |
| APP-VIEW-8 | Generic HTTP fallback groups several routes: Instagram suggestion acceptance (502), chat authorization (503), mention resolution (503), link previews (503), and post reads (503). Mention resolution discarded the underlying identity cause using `%v` and omitted capture; retain `%w` and capture once at the handler before returning the same response. Existing Instagram rejection and post-read capture paths already retain causes; disabled link previews have bounded admission metrics. Chat code producing development events is absent from this checkout. Keep the group open and correlate route/request IDs with a new cause-bearing event. |
| APP-VIEW-H | Development push delivery failure classified by the actual Firebase SDK as `firebase.UNKNOWN`, with no retained HTTP response/status. Existing delivery diagnostics include the original typed cause, operation/stage, attempt, initiating account and work reference; SDK selectors retain reviewed Firebase/FCM code and numeric status when available. Keep open. Check the emitting development checkout's Firebase configuration and provider logs by work reference; do not infer an APNs/FCM credential failure or mutate production configuration from this event. |
| APP-VIEW-K, M, N | Development-only chat failures: database deadline, HTTP timeout and upstream PDS 502 respectively. Chat source is absent from this checkout. Existing causes, timeout/network classification, PDS machine code/status and work/run references distinguish the failure boundaries. Keep open for investigation in the emitting chat checkout; do not guess changes to unrelated main-branch code. |
| APP-VIEW-G | Repository reconciliation reached attempt five with `lease_superseded`. The mandatory diagnostics policy deliberately escalates exhausted attempts while durable retries continue. Existing event already retains reason, operation, public actor and attempt. Keep open; inspect the fenced job's current state and recurrence before changing lease/fencing/retry semantics. A single superseded lease is insufficient evidence of a defect. |

## Riverpod follow-up

A deeper ownership review found and fixed mutation/provider misuse rather than relying only on disposal guards. See [riverpod-audit.md](riverpod-audit.md) for the reproduced failures, operation-scoped retention and dependency ownership fixes.

## Validation

Validation results and remaining release/device limitations are recorded in [validation.md](validation.md). Local tests do not establish production delivery, remote symbolication, device behavior, or absence of recurrence.
