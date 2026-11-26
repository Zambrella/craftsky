# Recent search profile decorations

Profile search history stores a DID plus a small presentation snapshot: handle,
display name, avatar and crafts. It does not store a complete profile. Existing
AppView identity middleware hydrates current customisation and account type from
the DID on every recent-search response, applying current subscription access.

Flutter's recent-search decoder and summary construction discarded those
server-provided decorations. Web's recent-author conversion also discarded
them, and recent/autocomplete rows lacked the profile-scoped avatar border
colour already used by submitted profile results.

Both clients now preserve the read-only customisation and account type for
rendering. Web recent/autocomplete rows apply the shared profile theme. Flutter
history writes still serialize only the original display snapshot, so decorated
fields do not become stored claims or change history deduplication. Existing
history benefits when loaded again. Names and handles remain display snapshots,
and navigation uses DID.

When either client loads history, AppView resolves the current visible avatar
from the indexed profile by DID. Removing an avatar clears the saved URL in the
response. Both clients display the first 10 entries from the newest-first
history list.

## Validation

- Flutter: 52 focused recent-search model, page, API-client and provider tests
  passed. New checks cover decorated rows, business icons, avatar colour and
  exclusion of decorations from save payloads. Focused analysis is clean.
- Web: three focused search-model tests passed. Eight authenticated search
  browser tests passed, including decorated recent/autocomplete rows and DID
  navigation; two existing Worker-only cases were skipped by this configuration.
- Web check/build and lint passed.
- AppView: focused recent-search persistence and decoration tests passed with
  the local test database. A regression test confirms old snapshots receive
  current Business/customisation values and downgrade to regular/defaults when
  current subscription access is removed.
- The running Flutter app was restarted after the model change. This refreshes
  the new fields; no production or provider configuration was changed.
