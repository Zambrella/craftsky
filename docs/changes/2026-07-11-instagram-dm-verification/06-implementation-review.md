# Implementation Review: Instagram DM Verification And Follow Discovery

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-07-23
Risk level: High

## Summary

The previously reviewed Instagram implementation and its IR-001–IR-011
remediation remain intact. This review covers the new mobile Instagram ZIP
extension against `FR-031`, `NFR-010`, `RULE-011`, `UT-017`, `UT-018`,
`IT-023`, and `REG-013`.

The extension has the right overall boundary: the native path enters an
isolate, the archive stays file-backed, only normalized usernames return, and
the AppView request/storage contract remains `instagramJson`. The current
implementation is not ready to hand off, however. It invokes
`ZipDecoder.decodeStream` before independently validating the actual central
directory entries. Archive 4.0.9 builds every header object and eagerly
decompresses Unix symlink entries during that call. A dishonest declared entry
count or an unrelated symlink can therefore bypass the intended pre-decode
memory boundary. Two Must-level test groups also record broader evidence than
they currently exercise.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-012 | Critical | Risk / Behavior | The bounded preflight trusts the EOCD/ZIP64 declared entry count and then calls `ZipDecoder.decodeStream` before counting the actual central headers. The dependency iterates every actual central header, and for entries marked as Unix symlinks it calls `readBytes()` to determine the link target. A crafted ZIP can declare at most 100,000 entries while supplying more actual headers within the 64 MiB directory, causing excessive allocation before the later `headers.length` comparison. A target or unrelated symlink can also be decompressed before CraftSky checks target size, type, or canonicality. This contradicts the actual-entry cap and “only the target is decompressed” boundary. | `FR-031`, `NFR-010`, `AC-052`, `EC-017`–`EC-019`, `RISK-011`; `app/lib/instagram_migration/services/instagram_export_file_parser_native.dart:53-103`; Archive 4.0.9 `zip_decoder.dart:18-68` | Before invoking the package decoder, scan the bounded central directory from the file to validate every header structure, count actual headers independently, locate exactly one target, and reject unsafe Unix-symlink metadata without reading entry content. Keep the decoder call only after this preflight. Add regressions for a low declared count with too many actual headers and for target/unrelated symlink entries, proving rejection occurs before decompression. |
| IR-013 | Important | Tests / Traceability | `UT-018` requires central-directory entry and byte limits immediately below, exactly at, and immediately above 100,000/64 MiB. The test currently covers only a normal far-below archive and synthetic declared values one above each limit. It does not exercise exact acceptance, below-boundary acceptance, actual-versus-declared header disagreement, or the failure-before-decode property. The execution log nevertheless marks the archive metadata limits covered. | `UT-018`, `TD-012`, `NFR-010`, `AC-052`; `app/test/instagram_migration/services/instagram_export_file_parser_test.dart:269-302`; `05-implementation-plan.md` Z2 | Add focused tests for below/at/above behavior through a small-limit test seam or another resource-safe boundary harness, including actual header counts rather than only EOCD claims. Update Z2 evidence to state exactly what ran. |
| IR-014 | Important | Tests / Traceability | `IT-023` specifies native JSON/ZIP selection, silent picker cancellation, page disposal, every safe parser error, localized UI output, and account-switch fencing. The page tests cover normalized success and one A-to-B late success; the privacy test calls the parser directly. There is no cancellation, disposal, late-error, or localized `invalidArchive`/`archiveTooLarge` coverage, and no test that composes selected native path parsing with the picker boundary. Z3 evidence says the cancellation cases were added even though they are absent. | `IT-023`, `AC-050`–`AC-054`, `EC-022`; `app/test/instagram_migration/instagram_migration_page_test.dart:92-358`; `app/test/instagram_migration/instagram_import_privacy_test.dart`; `05-implementation-plan.md` Z3 | Add widget/provider regressions for null cancellation, disposal, switch-away/back late success and late error, each safe archive error mapping, and a synthetic native path flowing through the selected-file parser to the normalized repository request. Correct Z3 evidence afterward. |
| IR-015 | Suggestion | Code Quality / Scope | The approved coding plan calls for stored and deflated ZIP targets, but the implementation additionally accepts BZip2 without a requirement or test. This expands the decompression surface without traceability. | `04-coding-plan.md` step 20; `app/lib/instagram_migration/services/instagram_export_file_parser_native.dart:91-96`; `UT-018` | Prefer restricting accepted methods to stored/deflate. If BZip2 support is intentional, add it to the approved contract and cover its bounded success/failure behavior. |

## Requirement And Test Traceability

- Requirements implemented: strict current URL/title records; standalone JSON
  compatibility; native path-to-isolate handoff; file-backed ZIP input; exact
  canonical target; declared/actual target byte bounds; normalized-only result;
  unchanged `instagramJson` request; account-switch success fencing; updated
  export copy and safe error categories.
- Tests implemented: `UT-017` exact/fuzzy record grammar; stored and deflated
  ZIP success; missing/duplicate target; encrypted/unsupported compression;
  local/central method disagreement; truncation/CRC; ZIP64 EOCD metadata;
  target byte bounds; normalized-entry cap; repository privacy canaries;
  normalized page success and A-to-B late-result discard.
- Unplanned behavior: BZip2 target acceptance.
- Remaining gaps: safe actual-directory preflight, symlink behavior,
  exact archive metadata boundaries, native picker composition,
  cancellation/disposal/late-error behavior, localized archive error widgets,
  and physical-device `MAN-005`.

## Test Evidence

- Commands reviewed:
  - focused parser and Instagram Flutter suites
  - full `flutter test`
  - full AppView `go test ./...`
  - changed-file `dart format --output=none --set-exit-if-changed`
  - `flutter analyze`
  - `git diff --check`
- Passing evidence:
  - Full Flutter and AppView test suites pass.
  - Changed Instagram Dart files are formatted.
  - `git diff --check` passes.
  - Analyzer output contains only 13 pre-existing info-level findings outside
    this slice.
  - The approved external ZIP parsed to 88 normalized following entries on the
    host and was not copied into the repository.
- Failing or skipped tests:
  - No currently implemented automated test fails.
  - The Must cases described in IR-013 and IR-014 were not implemented.
  - Full repo-wide Dart formatting still reports four unrelated pre-existing
    drifts; changed-file formatting is clean.
  - Physical iOS/Android picker, responsiveness, lifecycle, and peak-memory
    checks remain pending under `MAN-005`.

## Risk Review

- Risk level: High.
- Risk notes: The input is a user-selected private archive that may be very
  large. File-backed processing and bounded target output are good foundations,
  but the pre-decode central-directory/symlink gap leaves an avoidable
  allocation/decompression path outside those bounds.
- Approval notes: Keep the feature mobile-only and do not enable it for release
  until IR-012–IR-014 and the physical-device portion of `MAN-005` are complete.

## UI Polish Recommendation

- Recommendation: Optional.
- Reason: The new selector labels and locality explanation are coherent.
  Behavior/test corrections take priority.
- Suggested polish notes: Consider making the archive-too-large copy generic
  enough to cover excessive directory metadata as well as excessive file count,
  and verify busy/error states on a physical phone after the required fixes.

## Handoff Back To TDD Builder

- Required fixes: IR-012, IR-013, and IR-014.
- Suggested next failing test: Add a small-limit central-directory preflight
  test whose declared count is within the limit but whose actual header count
  exceeds it; then add a Unix-symlink entry regression that proves the package
  decoder is not reached before safe rejection.
- Verification to rerun: focused ZIP/parser/page/privacy tests, all Instagram
  Flutter tests, full Flutter tests, full AppView tests, changed-file formatting,
  analyzer, `git diff --check`, the approved host ZIP, and finally `MAN-005` on
  iOS and Android.
