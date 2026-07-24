# Coding Plan: Flutter Saved Posts

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Document-review verdict: Approved with notes
- Risk level: Medium
- Blocking issues: None
- Traceability rule: use the detailed AT/UT/IT/REG rows in `02-acceptance-tests.md` as authoritative ownership links, per DR-001; do not copy broader coverage-matrix-only links into implementation names.

## 2. Implementation Strategy

Build one private `saved_posts` Flutter feature boundary on top of the already implemented AppView save/folder API, plus the approved additive folder-delete mode. Keep the data flow account-keyed from the fixed-account Dio client through repositories, collection providers, mutation state, dialogs, and pages. Full `PostCard` instances get one reusable bookmark control; compact quote, notification, and saved-list presentations share a separate action-free `PostSummary`.

The implementation should proceed in the approved order:

1. Extend the canonical `Post` wire model with saved viewer fields.
2. Add typed saved/folder DTOs and fixed-account API/repository seams.
3. Add the account-scoped URI state map and per-URI selector.
4. Add confirmation-driven save/move, optimistic unsave, and the bookmark/dialog UI.
5. Add independently paginated folder, Unfiled, and per-folder collection state, then Settings routes/pages.
6. Extend AppView folder deletion with the atomic `deleteSaves=true` mode.
7. Extract `PostSummary` and migrate quote, notification, and saved-row consumers.
8. Prove account races, privacy, regressions, accessibility, generated-code, dependency-diff, and full-suite gates.

Key implementation decisions:

- `accountSavedPostStateProvider(AccountKey)` owns a redacted in-memory map keyed by canonical `AtUri`. A derived `savedPostPresentationProvider(SavedPostKey)` selects one URI so unrelated cards do not rebuild. The map is the only save/move/unsave reconciliation seam. (`FR-018`, `FR-019`; `UT-004`, `IT-005`, `IT-006`)
- Canonical `Post` data seeds an absent URI entry. A confirmed local mutation remains authoritative for that provider lifetime, so an older rendered `Post` cannot overwrite it; account-boundary invalidation or a new feature load reseeds from AppView. Mutation responses explicitly reconcile folder assignment and `savedAt`. (`FR-001`, `FR-018`; `AC-024`, `UT-004`)
- Saved/folder repositories are keyed by `AccountKey` and use `accountDioProvider(account)`, never an identity supplied in the request body. Provider state, route arguments, and keys override `toString`/generated methods where needed so diagnostics do not expose account, URI, folder ID, name, or cursor. (`FR-019`, `FR-026`, `NFR-001`; `IT-006`, `IT-012`)
- Folder rows, overview Unfiled posts, and each folder/sort list keep separate cursor/loading/error state. Folder create/rename/delete always invalidates an unsafe alphabetical cursor and restarts or reconciles from page one while deduplicating by opaque ID. A small known-folder-by-ID cache retains a newly selected folder and confirmed open-folder title across that restart. (`FR-010`, `FR-013`; `UT-005`, `IT-003`, `IT-008`, DR-002)
- Save and move send an explicit nullable `folderId` and wait for confirmation. Unsave applies an immediate overlay, stores the exact prior confirmed state, sends one `DELETE`, and either commits or rolls back. (`FR-003`, `FR-006`; `UT-004`, `IT-004`, `IT-005`)
- The Saved overview consumes only the Unfiled list plus the folder list; it does not call the server's flattened `all` scope. Foldered posts appear only on the corresponding folder screen. (`FR-009`, `RULE-010`; `AT-004`, `UT-006`)
- AppView parses `deleteSaves` into an explicit delete mode. Preserve mode deletes the folder and relies on the existing composite foreign key's `ON DELETE SET NULL`; remove mode deletes the owner's rows and folder inside one transaction. Neither path enumerates hydrated Flutter rows or contacts a PDS. (`FR-015`–`FR-017`; `IT-009`–`IT-012`)
- `PostSummary` renders only optional compact data fields supplied by adapters: author, bounded text, timestamp/metadata, first representative image, project title, and availability state. The widget owns no engagement, bookmark, folder, or mutation logic. Parents retain outer chrome, navigation, reveal policy, notification context, saved time, Move, and Unsave. (`FR-020`–`FR-023`; `UT-008`, `AT-010`–`AT-012`)

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Canonical post model | `Post` is `dart_mappable` wire state; protected placeholders are normalized by `PostWireHook`. | Decode/copy `viewerHasSaved` and nullable `viewerSavedFolderId`; default only protected/legitimate omission to false/null. | FR-001 | AT-001, UT-001, REG-005 |
| Saved Flutter data layer | Feature does not exist; API features use Dio + `unwrapApi` + repository interfaces/fakes. | Add typed saved/folder models, fixed-account API client, repository interface/implementation, and fake. | FR-007, FR-019, FR-026 | UT-002, UT-003, IT-001, IT-002, IT-006 |
| URI mutation state | Like/repost mutate copied `Post` values and update several live caches. | Add one account-level `AsyncNotifier` map with exhaustive `AsyncValue` projection, per-URI selectors, duplicate guards, optimistic unsave rollback, and server-response reconciliation. | FR-006, FR-018, FR-019, NFR-003 | AT-003, AT-009, UT-004, UT-014, IT-004–IT-006 |
| Collection state | Existing list providers accumulate one opaque cursor. | Add separate folder, Unfiled/sort, and folder/sort resources with retained content, incremental retry, invalid-cursor restart, ID dedupe, and mutation reconciliation. | FR-009, FR-010, FR-013, NFR-002 | AT-004, AT-005, UT-005, UT-006, IT-003, IT-008 |
| Bookmark/dialog UI | `PostCard` has public engagement actions and overflow but no private bookmark. | Insert `SavedPostBookmarkButton` immediately before overflow; add one save/move chooser with inline folder creation and degraded folder loading. | FR-002–FR-006, FR-024, RULE-001, RULE-002, RULE-004 | AT-001–AT-003, UT-013, IT-004, IT-005, REG-001 |
| Settings/navigation | `/profile/saved` and `ProfileTab.saved` are placeholders; Settings is a typed full-screen route. | Make `/profile/settings/saved` canonical, add a Settings tile, add a static generic folder child route using redacted `$extra`, and remove the profile Saved surface. | FR-008, FR-013, RULE-006 | AT-004, UT-011, IT-007, REG-002 |
| Saved pages/widgets | No collection UI exists. | Add overview, separate folder page, sort/pagination controls, folder management, exact-post navigation, `SavedPostRow`, and safe delete choices. | FR-009–FR-014, FR-024–FR-026 | AT-004–AT-007, UT-006, UT-007, UT-014, IT-008 |
| Compact post UI | Quote preview is private code in `post_card.dart`; notifications render subject text inline. | Add shared `PostSummary`/data adapters and migrate quote, post-bearing notification, and saved-row content without changing parent behavior. | FR-020–FR-023, NFR-004 | AT-010–AT-012, UT-008, REG-003, REG-004 |
| AppView delete mode | `DeleteFolder` performs one owner-scoped folder delete; FK unfiles saves. | Strictly parse optional `deleteSaves`; add transactional remove-saves mode while preserving absent/false behavior and idempotent privacy. | FR-015–FR-017, NFR-001, RULE-003 | AT-008, UT-009, UT-012, IT-009–IT-012, REG-006–REG-008 |
| Account boundary | Account activation invalidates registered feature providers before switching. | Register every saved repository/provider/controller family with `accountStateInvalidatorProvider`; guard dialog/navigation/message completions with the initiating lease. | FR-018, FR-019, RULE-009 | AT-009, IT-006, REG-009 |
| Localization/errors/diagnostics | ARB-generated strings, sealed `ApiException`, `AppErrorMapper`, bounded endpoint categories, and Sentry allowlists. | Add saved strings, safe error projections, saved endpoint categories, and redacted provider/route keys; never render raw server text. | FR-024, FR-025, NFR-001 | AT-013, UT-010, IT-012 |

## 4. Files And Modules

### Flutter production files

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/feed/models/post.dart` | Change | Add saved viewer fields and protected-placeholder defaults. | FR-001 | UT-001, REG-005 |
| `app/lib/feed/models/post.mapper.dart` | Change generated | Decode/encode/copy the saved viewer fields, including explicit nullable clear. | FR-001, NFR-005 | UT-001, AT-013 |
| `app/lib/saved_posts/models/saved_post.dart` | Create | Define `SavedPostState`, `SavedPostItem`, page DTO, assignment, scope, and saved-time sort wire/domain types. | FR-007, FR-011 | UT-002, IT-001 |
| `app/lib/saved_posts/models/saved_post_folder.dart` | Create | Define folder/page DTOs plus normalized folder-name validation matching AppView without uniqueness rules. | FR-004, FR-005, FR-013, FR-026 | UT-002, UT-003 |
| `app/lib/saved_posts/models/saved_post_keys.dart` | Create | Define equality-safe, redacted account/URI/list/dialog/folder-route keys and args. | FR-019, FR-026, NFR-001 | UT-004, UT-011, IT-006, IT-007 |
| `app/lib/saved_posts/models/saved_post_viewer_state.dart` | Create | Model confirmed/pending/optimistic state, exact rollback snapshot, revision, and redacted account map. | FR-006, FR-018, NFR-003 | UT-004, UT-014 |
| `app/lib/saved_posts/models/saved_posts_collection_state.dart` | Create | Model folder and post pages, incremental error/busy state, overview projection, and known-folder retention. | FR-009, FR-010, FR-013 | UT-005, UT-006 |
| `app/lib/saved_posts/models/*.mapper.dart` | Create generated | Checked-in `dart_mappable` outputs where the source models use mappers/copy support. Do not generate diagnostic `toString` for private-value state. | FR-007, NFR-001, NFR-005 | AT-013 |
| `app/lib/saved_posts/data/saved_post_api_client.dart` | Create | Implement all existing save/list/folder calls and `deleteSaves` serialization through fixed-account Dio. | FR-007, FR-015, FR-017 | IT-001 |
| `app/lib/saved_posts/data/saved_post_repository.dart` | Create | Expose typed operations without JSON or Dio at provider/UI call sites. | FR-007 | IT-002 |
| `app/lib/saved_posts/data/api_saved_post_repository.dart` | Create | Delegate the repository contract to `SavedPostApiClient`. | FR-007 | IT-002 |
| `app/lib/saved_posts/providers/saved_post_repository_provider.dart` | Create | Provide account-keyed API/repository instances via `accountDioProvider(account)`. | FR-019, NFR-001 | IT-006 |
| `app/lib/saved_posts/providers/account_saved_post_state_provider.dart` | Create | Own the account URI map; seed canonical posts; perform/save/move/unsave; reconcile list entities and folder deletion. | FR-006, FR-018, FR-019 | AT-003, AT-009, UT-004, UT-014, IT-004–IT-006 |
| `app/lib/saved_posts/providers/saved_post_folders_provider.dart` | Create | Load/page/retry folders and create/rename/delete with page-one reconciliation, cursor invalidation, and ID dedupe. | FR-005, FR-010, FR-013, FR-026 | AT-005, AT-007, UT-005, IT-003, IT-008 |
| `app/lib/saved_posts/providers/saved_posts_provider.dart` | Create | Load/page/refresh/retry one Unfiled or folder scope and sort; retain confirmed rows on incremental failures. | FR-009–FR-011, FR-025 | AT-005, AT-006, IT-003, IT-008 |
| `app/lib/saved_posts/providers/save_post_dialog_controller.dart` | Create | Own selection, independent inline-create state, confirm busy/error state, and initiating account lease. | FR-004–FR-006, FR-019 | AT-002, UT-013, IT-004, IT-006 |
| `app/lib/saved_posts/providers/*.g.dart` | Create generated | Riverpod generated outputs for the new providers. | FR-007, NFR-005 | AT-013 |
| `app/lib/saved_posts/widgets/saved_post_bookmark_button.dart` | Create | Render localized selected/busy bookmark semantics and dispatch chooser versus immediate unsave. | FR-002, FR-003, FR-006, FR-024 | AT-001, AT-003, REG-001 |
| `app/lib/saved_posts/widgets/save_post_dialog.dart` | Create | Render No folder/current folder, incremental folders, explicit confirm, inline New folder, Retry, and no search. | FR-004–FR-006, FR-024 | AT-002, UT-013, IT-004 |
| `app/lib/saved_posts/widgets/saved_post_row.dart` | Create | Compose `PostSummary` with parent-owned saved time, Move, Unsave, and exact tap. | FR-011, FR-023 | AT-006, AT-012, IT-008 |
| `app/lib/saved_posts/widgets/saved_post_folder_dialogs.dart` | Create | Add/rename folder input and the Cancel/keep/remove delete dialog with safe focus/destructive semantics. | FR-013, FR-014, FR-024 | AT-007, IT-008 |
| `app/lib/saved_posts/navigation/saved_post_destination.dart` | Create | Infer top-level versus root-plus-exact-focus `PostThreadRoute` destinations. | FR-012 | AT-006, UT-007, REG-004 |
| `app/lib/saved_posts/pages/saved_posts_page.dart` | Create | Settings overview: folders first, independent Unfiled list, sort, add folder, empty/error/loading states, and retained scroll. | FR-008–FR-011, FR-013 | AT-004–AT-007, IT-008 |
| `app/lib/saved_posts/pages/saved_post_folder_page.dart` | Create | Separate named folder screen with sort, items, Rename/Delete overflow, title reconciliation, and pop behavior. | FR-010–FR-014 | AT-005–AT-007, IT-008 |
| `app/lib/shared/widgets/post_summary.dart` | Create | Define compact data adapters and action-free presentation for visible/protected/unavailable summaries. | FR-020–FR-023 | AT-010–AT-012, UT-008 |
| `app/lib/feed/widgets/post_card.dart` | Change | Insert bookmark before overflow and replace private quote content with `PostSummary` while preserving quote wrapper/policy. | FR-002, FR-021 | AT-001, AT-010, REG-001, REG-003 |
| `app/lib/notifications/widgets/notification_row.dart` | Change | Replace post-subject text with text-only `PostSummary`; retain row context and existing `_openPost` routing. | FR-022 | AT-011, REG-004 |
| `app/lib/router/route_locations.dart` | Change | Keep `saved` beneath Settings and add static `folder` child; no private route parameters. | FR-008, FR-013 | UT-011, IT-007 |
| `app/lib/router/router.dart` | Change | Replace `SavedRoute` with `SavedPostsRoute` nested under `SettingsRoute`; add generic `SavedPostFolderRoute` with redacted `$extra`. | FR-008, FR-013 | AT-004, UT-011, IT-007 |
| `app/lib/router/router.g.dart` | Change generated | Checked-in typed route generation for Saved overview/folder routes. | FR-008, NFR-001, NFR-005 | UT-011, IT-007 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add localized Saved posts tile that pushes the typed full-screen route. | FR-008, RULE-006 | AT-004, IT-007 |
| `app/lib/profile/widgets/profile_tab_bar.dart` | Change | Remove `ProfileTab.saved` and saved count plumbing. | FR-008, RULE-006 | AT-004, REG-002 |
| `app/lib/profile/pages/profile_page.dart` | Change | Remove the Saved placeholder tab branch and adjust tab count through enum values. | FR-008, RULE-006 | AT-004, REG-002 |
| `app/lib/profile/pages/saved_page.dart` | Delete | Remove the old `/profile/saved` placeholder implementation. | FR-008 | AT-004, REG-002 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Invalidate all saved feature repository/provider/controller families before account activation or active-session removal. | FR-019, RULE-009 | AT-009, IT-006, REG-009 |
| `app/lib/shared/api/providers/error_mapping_interceptor.dart` | Change | Add bounded categories for post saves, saved lists, folders, and folder detail without embedding identifiers. | FR-025, NFR-001 | UT-010, IT-012 |
| `app/lib/l10n/app_en.arb` | Change | Add all Saved/bookmark/folder/error/empty/sort/destructive/accessibility copy; remove obsolete profile Saved strings. | FR-024 | AT-001, AT-002, AT-004, AT-007, AT-013 |
| `app/lib/l10n/generated/app_localizations*.dart` | Change generated | Checked-in localization output. | FR-024, NFR-005 | AT-013 |

### AppView production files

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/saved_post_request.go` | Change | Add strict `deleteSaves` query parsing: absent/false preserve, true remove, repeated/empty/invalid/unknown reject. | FR-015 | UT-009, IT-009 |
| `appview/internal/api/saved_post.go` | Change | Add explicit delete mode to `SavedPostFolderStore`; parse mode before calling the store; retain standard envelope/204 semantics. | FR-015, FR-016 | AT-008, UT-012, IT-009 |
| `appview/internal/api/saved_post_store.go` | Change | Execute owner-scoped preserve/remove modes atomically and idempotently; no migration or PDS collaborator. | FR-015–FR-017, NFR-001 | IT-010–IT-012, REG-006–REG-008 |
| `appview/internal/routes/routes.go` | No behavioral change expected | Existing authenticated/write/no-body route remains canonical; only the handler/store contract changes. | FR-015 | IT-009, REG-006 |

### Test and support files

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/test/saved_posts/fakes/fake_saved_post_repository.dart` | Create | Programmable typed fake with call recording/completers and redacted diagnostics. | FR-007, FR-019 | IT-002–IT-008 |
| `app/test/feed/models/post_test.dart` | Change | First red test for saved viewer decoding/default/copy/clear. | FR-001 | UT-001 |
| `app/test/saved_posts/models/*_test.dart` | Create | DTO, folder validation/identity, overview, destination, and safe-error unit coverage. | FR-004, FR-007, FR-009, FR-012, FR-025, FR-026 | UT-002, UT-003, UT-006, UT-007, UT-010 |
| `app/test/saved_posts/data/*_test.dart` | Create | Dio contract and repository delegation coverage for every consumed operation. | FR-007, FR-015, FR-017 | IT-001, IT-002 |
| `app/test/saved_posts/providers/*_test.dart` | Create | URI reducer, pagination/reconciliation, dialog controller, mutations, and account-race coverage. | FR-004–FR-006, FR-010, FR-018, FR-019 | AT-003, AT-005, AT-009, UT-004, UT-005, UT-013, UT-014, IT-003–IT-006 |
| `app/test/saved_posts/widgets/*_test.dart` | Create | Chooser, bookmark, row, folder dialog, PostSummary ownership, semantics, and constrained-layout coverage. | FR-002–FR-006, FR-014, FR-023, FR-024 | AT-001–AT-003, AT-007, AT-012, AT-013 |
| `app/test/saved_posts/pages/*_test.dart` | Create | Overview/folder hierarchy, paging, sort, actions, errors, empty state, mutation reconciliation, and scroll restoration. | FR-009–FR-014 | AT-004–AT-007, IT-008 |
| `app/test/shared/widgets/post_summary_test.dart` | Create | Adapter/content/state tests and proof that compact summaries contain no engagement/folder logic. | FR-020–FR-023 | AT-010–AT-012, UT-008 |
| `app/test/feed/widgets/post_card_test.dart` | Change | Bookmark placement/branches, protected omission, action-layout regression, and quote extraction regression. | FR-002, FR-003, FR-021 | AT-001, AT-003, AT-010, REG-001, REG-003, REG-005 |
| `app/test/notifications/notifications_page_test.dart` and destination tests | Change | Cover every post-bearing category, unchanged context/filter/destination, and `PostSummary`. | FR-012, FR-022 | AT-011, REG-004 |
| `app/test/router/saved_posts_route_test.dart` | Create | Typed canonical route, static generic folder route, full-screen stack, redacted diagnostics, and back behavior. | FR-008, FR-013 | AT-004, UT-011, IT-007 |
| `app/test/settings/settings_page_test.dart` | Change | Assert localized Saved posts tile and typed navigation. | FR-008 | AT-004 |
| `app/test/profile/saved_page_test.dart` | Delete | Placeholder no longer exists. | FR-008 | REG-002 |
| Existing profile/account-boundary tests | Change | Assert no Saved tab on own/visited profiles and saved providers clear during activation. | FR-008, FR-019 | REG-002, REG-009 |
| `appview/internal/api/saved_post_folder_request_test.go` | Change | Extend strict query parsing cases. | FR-015 | UT-009 |
| `appview/internal/api/saved_post_test.go` | Change | Update fake store signature and cover absent/false/true/invalid/unknown handler behavior. | FR-015 | UT-012, IT-009, REG-006 |
| `appview/internal/api/saved_post_store_test.go` | Change | Cover preserve timestamps, remove hidden/all rows, ownership, concurrency/idempotency, and deterministic rollback. | FR-015–FR-017, NFR-001 | IT-010, IT-011, REG-006–REG-008 |
| `appview/internal/api/saved_post_response_test.go`, `saved_post_cursor_test.go`, `saved_post_policy_test.go`, and `saved_post_lifecycle_test.go` | Run unchanged; extend only for a focused assertion gap | Prove the delete extension leaves saved-list JSON, opaque pagination, hydration policy, and exact reply context unchanged. | NFR-005 | REG-007 |
| `appview/internal/api/saved_post_observability_test.go` | Change | Assert bounded diagnostics/metrics, sentinel redaction, author non-disclosure, and zero public/PDS effects. | FR-016, FR-025, NFR-001 | IT-012, AT-013, REG-008 |
| `appview/internal/routes/routes_test.go` | Change only if needed for assertions | Retain auth/device/write/no-body policy coverage for the unchanged route. | FR-015 | IT-009 |

No lexicon, migration, SQLC output, PDS client, firehose, notification schema, runtime dependency, or durable Flutter storage file is planned.

## 5. Services, Interfaces, And Data Flow

### Flutter wire and repository contracts

```text
enum SavedPostSort { newest, oldest }
enum SavedPostScopeKind { unfiled, folder }

final class SavedPostState {
  DateTime savedAt;
  String? folderId;
}

final class SavedPostItem {
  Post post;
  DateTime savedAt;
  String? folderId;
}

abstract interface class SavedPostRepository {
  Future<SavedPostState> save(Post post, {required String? folderId});
  Future<void> unsave(Post post);
  Future<SavedPostPage> list({
    required SavedPostScope scope,
    required SavedPostSort sort,
    String? cursor,
    int? limit,
  });
  Future<SavedPostFolderPage> listFolders({String? cursor, int? limit});
  Future<SavedPostFolder> createFolder(String name);
  Future<SavedPostFolder> renameFolder(String folderId, String name);
  Future<void> deleteFolder(String folderId, {required bool deleteSaves});
}
```

Serialization guardrails:

- `save`/`move` always POST `{"folderId": <id-or-null>}` so moving to Unfiled is explicit; the API client derives `{did}/{rkey}` from typed `Post` data and never sends an owner DID.
- `unsave` sends a no-body `DELETE /v1/posts/{did}/{rkey}/saves`.
- `list` sends either `unfiled=true` or `folderId=<opaque>`, plus `sort`, optional cursor, and bounded limit. Cursors/IDs are round-tripped without inspection.
- `deleteFolder(deleteSaves: false)` omits `deleteSaves`; `true` serializes exactly `deleteSaves=true`. Flutter never pages rows to implement the destructive option.
- API errors remain sealed `ApiException`/`AppError` classifications. UI copy is localized and never interpolates `ApiException.message`, folder values, URI, cursor, or owner/target identity.

### Save/move/unsave flow

```text
PostCard(post)
  -> SavedPostBookmarkButton(post)
    -> reads active AccountKey
    -> seeds accountSavedPostStateProvider(account) only if post.uri absent
    -> watches savedPostPresentationProvider(SavedPostKey(account, uri))

outlined bookmark tap
  -> open SavePostDialog(account, post, initialFolderId: null)
  -> dialog watches savedPostFoldersProvider(account)
  -> confirm calls accountSavedPostStateProvider(account).save(post, folderId)
  -> notifier consumes the repository's SavedPostState response internally
  -> notifier publishes the confirmed URI state or safe failure state
  -> dialog matches the selected URI's AsyncValue/presentation state
  -> if confirmed and the initiating ActiveAccountLease is still current:
       update any live source/destination saved list by server savedAt
       close silently
     otherwise:
       keep the dialog open for failure, or suppress active UI effects for a stale lease

filled bookmark / row Unsave
  -> account state stores exact confirmed snapshot and publishes unsaved overlay
  -> one repository DELETE; duplicate taps see pending URI and no-op
  -> 204 commits unsaved and removes from live source list
  -> failure restores exact snapshot and exposes only localized safe feedback

Move from saved row
  -> open same dialog with current folder/Unfiled preselected
  -> keep item in current list while POST is pending
  -> success uses server folderId/savedAt, then repositions live source/destination lists
  -> failure leaves confirmed list/overlay unchanged and retains attempted selection inline
```

### Folder mutation reconciliation (DR-002)

```text
create / rename / delete succeeds
  -> retain server-returned folder entity by opaque ID when still valid
  -> discard pre-mutation alphabetical cursor and incremental error
  -> refetch/restart page one from AppView server order
  -> dedupe page items solely by opaque ID
  -> keep newly created dialog selection by ID even if it sorts beyond page one
  -> update open-folder title from confirmed rename result
  -> clear deleted chooser selection and folder entity
  -> preserve overview ScrollController/PageStorage while route remains mounted
```

Folder deletion additionally reconciles known URI state:

- Keep saves: confirmed known entries assigned to that folder become saved + Unfiled without changing `savedAt`; refresh live deleted-folder and Unfiled resources after confirmation.
- Remove saves: confirmed known entries assigned to that folder become unsaved; refresh/remove live folder rows.
- On ambiguous failure: keep the last confirmed folder/list/URI state and offer localized retry. The AppView transaction is the authority for hidden rows.

### AppView delete contract

```text
enum SavedPostFolderDeleteMode {
  preserveSaves,
  removeSaves,
}

ParseSavedPostFolderDeleteQuery(url.Values)
  absent / deleteSaves=false -> preserveSaves
  deleteSaves=true           -> removeSaves
  empty/repeated/other value/unknown key -> FieldError(validation_failed)

SavedPostFolderStore.DeleteFolder(ctx, owner, folderId, mode)

DeleteFolder transaction:
  parse UUID; malformed -> idempotent nil
  BEGIN
  if removeSaves:
    DELETE saved_posts WHERE owner_did = owner AND folder_id = folderId
  DELETE saved_post_folders WHERE owner_did = owner AND id = folderId
    // preserve mode invokes existing composite FK ON DELETE SET NULL
  COMMIT
```

Missing/other-owner folders change zero rows and return `204`. Any failure before commit rolls back both save-row and folder work. The store receives no PDS, firehose, notification, or public-post dependency.

## 6. State, Providers, Controllers, Or DI

### Provider graph

```text
sessionRegistryProvider
  -> accountDioProvider(AccountKey) [existing fixed token + session generation]
    -> accountSavedPostApiClientProvider(AccountKey)
      -> accountSavedPostRepositoryProvider(AccountKey)
        -> accountSavedPostStateProvider(AccountKey)
          -> savedPostPresentationProvider(SavedPostKey)
        -> savedPostFoldersProvider(AccountKey)
        -> savedPostsProvider(SavedPostListKey)
        -> savePostDialogControllerProvider(SavePostDialogKey)

accountStateInvalidatorProvider [existing]
  -> invalidates all providers above before account activation/session removal
```

### Account URI state

Use an `AsyncNotifier` family whose state is `AsyncValue<AccountSavedPostStateMap>`. Its synchronous initial build may return the empty redacted map directly, and optimistic changes can still publish synchronously with `state = AsyncData(nextMap)` before awaiting the repository. Public mutation methods are commands returning `Future<void>`; they consume repository responses internally and expose pending, confirmed, rollback, and safe error outcomes through provider state instead of returning mutation result objects.

```text
@Riverpod(keepAlive: true)
class AccountSavedPostState extends _$AccountSavedPostState {
  FutureOr<AccountSavedPostStateMap> build(AccountKey account) => empty/redacted;

  void seedIfAbsent(Post post);
  Future<void> save(Post post, String? folderId);
  Future<void> move(SavedPostItem item, String? folderId);
  Future<void> unsave(Post post);
  void reconcileFolderDeletion(String folderId, DeleteFolderChoice choice);
}

@riverpod
AsyncValue<SavedPostPresentation> savedPostPresentation(
  Ref ref,
  SavedPostKey key,
) {
  final accountState = ref.watch(accountSavedPostStateProvider(key.account));
  return switch (accountState) {
    AsyncData(:final value) => AsyncData(value.forUri(key.uri)),
    AsyncLoading() => projectUriLoading(accountState, key.uri),
    AsyncError() => projectUriError(accountState, key.uri),
  };
}
```

`projectUriLoading` and `projectUriError` are pseudocode for preserving/projecting any previous account-map value while retaining the outer loading/error metadata; implementation may use a small private helper or the Riverpod API available in the pinned version.

Guardrails:

- `SavedPostKey`, map state, and folder/list keys have equality/hash implementations but redacted `toString`; do not let `dart_mappable` generate a private-value `toString`.
- Reducers and selectors exhaustively match `AsyncData`, `AsyncLoading`, and `AsyncError`. Where an `AsyncLoading`/`AsyncError` carries previous data, preserve and project that data rather than replacing confirmed bookmark state with a blank fallback.
- Expected per-URI mutation progress and failures remain inside that URI's presentation entry while the outer account provider stays usable; one save must not put every bookmark for the account into a global loading/error state.
- Seed only when a URI has no entry. Confirmed mutation state wins over stale copies still held by timeline/profile/search/thread/notification providers.
- Capture the initiating `ActiveAccountLease` before the first await. A keyed provider may reconcile only its own account; dialog close, navigation, messenger, and any active-screen side effect also require the same active lease at completion.
- A pending URI disables only that URI's conflicting bookmark/save/move/unsave controls. Folder create/rename/delete have separate operation guards.

### Folder and saved-list providers

Use generated `AsyncNotifier` families with explicit account/list keys. Keep initial failure in `AsyncError`, but represent incremental loading/error inside loaded state so confirmed rows remain visible.

```text
@riverpod
class SavedPostFolders extends _$SavedPostFolders {
  Future<SavedPostFolderListState> build(AccountKey account);
  Future<void> loadMore();
  Future<void> refresh();
  Future<SavedPostFolder?> create(String name);
  Future<SavedPostFolder?> rename(String id, String name);
  Future<bool> delete(String id, DeleteFolderChoice choice);
}

@riverpod
class SavedPosts extends _$SavedPosts {
  Future<SavedPostListState> build(SavedPostListKey key);
  Future<void> loadMore();
  Future<void> refresh();
  void removeConfirmed(AtUri uri);
  void upsertConfirmed(SavedPostItem item);
}
```

- A list key contains account, scope, and sort; the folder ID is stored but redacted from diagnostics.
- `loadMore` appends by canonical URI once, keeps the returned server order, and retains previous rows/cursor on failure.
- `invalid_cursor` triggers one page-one restart for only that scope/sort. If restart fails, retain the last confirmed rows and show Retry; do not loop automatically.
- Changing sort watches a different key with independent cursor/error state. The UI may keep both cached while mounted, but account invalidation clears both.
- Folder paging is independent of Unfiled paging. A Load more folders control remains above the Unfiled heading, so later folder pages cannot render below posts.

### Save dialog controller

Use one controller for save and move. It holds only UI orchestration state; mutations remain in the account URI seam and folder notifier.

```text
SavePostDialogState(
  selectedFolderId,       // null means No folder
  isCreatingFolder,
  createName,
  createErrorKind,
  isConfirming,
  confirmErrorKind,
)
```

Folder list failure belongs to `savedPostFoldersProvider` and does not disable the null selection or confirm button. The controller awaits `Future<void>` commands only for completion and derives success/failure from the selected URI's provider state; it does not receive a mutation outcome value. A successfully created folder is retained/selected independently of a later dialog cancellation.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Bookmark on full posts

`PostCard` continues to short-circuit to `_ProtectedPostCard`, so protected placeholders never build a bookmark. In the normal action row, insert a fixed tap-target `SavedPostBookmarkButton` after `Spacer` and before `_PostCardMenu`.

```text
Row
  public engagement actions
  Spacer
  SavedPostBookmarkButton(post)  // no count; outlined/filled; busy/selected semantics
  _PostCardMenu                  // existing adaptive More menu
```

The button is private save behavior only. `PostSummary` and `_ProtectedPostCard` never include it. Split AT-001 widget coverage into rendering/placement, unsaved-tap chooser, and saved-tap optimistic-unsave tests per DR-003.

### Save/move chooser

Use an `AlertDialog` or equivalent bounded dialog with:

- localized title for Save versus Move;
- `No folder` first and selected for a new save, or current assignment selected for a move;
- one-selection folder rows keyed by opaque ID, including duplicate display names;
- incremental Load more/Retry inside the folder region;
- inline `New folder` expansion with name field, validation, busy, and safe error;
- Cancel and one explicit Save/Move confirm action;
- no search;
- confirm remains busy/disabled and dialog stays open until success;
- folder-list errors do not block an Unfiled save;
- success closes silently; save/move failures stay inline with the attempted selection.

### Saved overview

`SavedPostsPage` is a `ConsumerStatefulWidget` with its own `ScrollController`/PageStorage identity. It always builds the root overview when pushed.

```text
Scaffold
  AppBar(
    title: Saved posts,
    actions: Add folder,
  )
  RefreshIndicator
    CustomScrollView(controller retained while folder route is pushed)
      Folders heading
      folder rows (server order, no counts)
      folder incremental spinner/error/Load more
      if unfiled rows exist:
        Unfiled heading + Newest/Oldest control
        SavedPostRow...
        unfiled incremental spinner/error/Load more
      else if folders empty:
        full Nothing saved yet state
```

If folders exist and Unfiled is empty, omit the entire Unfiled empty section. Initial errors are section-specific where possible; a folder error must not discard a confirmed Unfiled list, and vice versa.

### Folder screen

`SavedPostFolderRoute` uses static path `folder`, generic route name `saved-post-folder`, and redacted `$extra` carrying the confirmed folder. No ID/name appears in the URL or route/breadcrumb label.

```text
Scaffold
  AppBar(
    title: confirmed folder.name,
    actions: CraftskyContextMenuButton(Rename, Delete),
  )
  folder SavedPostRow list + Newest/Oldest + paging/error/empty states
```

A confirmed rename updates the page title immediately from the returned folder and restarts folder-list ordering. Delete always shows Cancel, keep saved posts, and remove saved posts—even for a visibly empty folder. Give Cancel `autofocus`; give remove-saves destructive color plus semantic hint. On confirmed success, pop to the still-mounted overview and reconcile it; on failure, remain on the folder page with confirmed state.

### Saved rows and navigation

`SavedPostRow` places saved time and Move/Unsave controls outside `PostSummary`. Use the established adaptive `CraftskyContextMenuButton` for row actions if inline controls do not fit at large text; either form must keep adequate tap targets and parent ownership.

- Top-level item tap: push `PostThreadRoute` for that post.
- Comment/reply tap: parse `post.reply.root.uri`, push the root route, and set `focus` to the exact saved `post.uri`.
- Do not navigate from unavailable content; show only localized safe state.

### Shared `PostSummary`

`PostSummaryData` has nullable fields rather than surface-specific modes. Adapters intentionally omit fields a surface does not currently show:

- Quote: author, text, first image, project title, created time, quote availability/reveal state; parent retains quote card chrome and author/post callbacks.
- Notification: bounded subject text only; parent retains actor/action header, avatar, icon/color, timestamp, follow control, filtering, unread treatment, and whole-row navigation.
- Saved row: author, text, first image, project title; parent retains `savedAt`, Move, Unsave, and exact-item navigation.

The widget must remain one level deep and never render nested quotes, full galleries/project cards, engagement counts, bookmark, More menu, folder name, or mutation state.

### Routes and localization

```text
ProfileRoute /profile
  SettingsRoute settings                    -> /profile/settings
    SavedPostsRoute saved                   -> /profile/settings/saved
      SavedPostFolderRoute folder           -> /profile/settings/saved/folder
```

Remove the sibling `/profile/saved` route and `ProfileTab.saved`. Add localized strings for all visible labels, hints, selected/busy/destructive semantics, folder validation, load/retry/error/empty states, sort options, saved time, and delete choices. Reuse existing generic Cancel/Retry/error strings where their meaning is exact.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Protected/blocked full post | Existing placeholder short-circuit; no bookmark or hidden data. | FR-001, FR-002 | AT-001, REG-005 |
| Unsaved bookmark | Outlined; opens chooser only; no request before confirm. | FR-002–FR-004, RULE-001 | AT-001, AT-002 |
| Saved bookmark | Filled/selected; immediate optimistic outline on tap; no dialog/Undo. | FR-003, FR-006, RULE-002 | AT-001, AT-003 |
| Duplicate tap/mutation | Per-URI pending state disables/serializes conflicting action; one request. | FR-006, NFR-003 | AT-003, UT-004, IT-004, IT-005 |
| Save/move pending | Keep dialog open and confirmed placement visible; one busy confirm action. | FR-006, NFR-003 | AT-002, AT-006, IT-004 |
| Save/move failure | Keep attempted selection/dialog open; localized safe inline error; no optimistic move. | FR-006, FR-025 | AT-002, AT-006, UT-010 |
| Unsave failure | Restore exact saved/folder/savedAt snapshot across selectors and keep list row; localized safe feedback. | FR-006, FR-018 | AT-003, UT-004, IT-005 |
| Folder list initial failure in chooser | Show inline Retry for folders; No folder remains selected/savable. | FR-004, FR-025 | AT-002, UT-013 |
| Folder create failure | Keep field/text editable, clear busy state, show safe inline error; do not close chooser. | FR-005, FR-025 | AT-002, UT-013 |
| Initial collection loading/error | Section/page progress; localized retry. Never display raw server values. | FR-010, FR-025 | AT-005, IT-008 |
| Incremental collection loading/error | Retain confirmed rows and cursor; show only that section's spinner/Retry. | FR-010 | AT-005, UT-005, IT-003 |
| Invalid cursor | Restart only the current account/scope/sort from page one; replace unsafe cursor and dedupe. | FR-010 | AT-005, UT-005, IT-003 |
| Duplicate/case-variant folders | Render each row; equality/selection/mutation uses opaque ID only. | FR-026, RULE-007 | AT-002, AT-007, UT-003 |
| Folder mutation during partial pagination | Retain confirmed entity where valid, discard cursor, restart server order, dedupe ID, clear deleted selection, preserve route scroll/title. | FR-005, FR-010, FR-013, FR-018 | AT-005, UT-005, IT-003, IT-008 |
| Folders exist, Unfiled empty | Render folders only; hide Unfiled empty section. | FR-009, RULE-010 | AT-004, UT-006 |
| No folders and no Unfiled items | Render full localized Nothing saved yet state. | FR-009, RULE-010 | AT-004, UT-006 |
| Exact saved reply | Open root thread with exact saved URI as focus. | FR-012 | AT-006, UT-007, REG-004 |
| Folder delete keep | One folder DELETE without `deleteSaves=true`; preserved saves become Unfiled with unchanged `savedAt`. | FR-014, FR-015, RULE-003, RULE-005 | AT-007, AT-008, IT-010, REG-006 |
| Folder delete remove | One folder DELETE with `deleteSaves=true`; all owned visible/hidden saves and folder removed atomically; no public/PDS effect. | FR-015–FR-017 | AT-008, IT-011, IT-012, REG-008 |
| Missing/malformed/other-owner folder | Store no-op and handler `204`; do not disclose existence. Invalid query is standard validation error. | FR-015, NFR-001 | UT-009, UT-012, IT-009–IT-012 |
| Transaction failure | Roll back save-row and folder changes; Flutter retains confirmed state and can retry. | FR-015, FR-025 | AT-008, IT-011 |
| Account switch during any operation | Feature invalidated before activation; stale completion cannot close/navigate/message/mutate the new active account. | FR-019, RULE-009 | AT-009, IT-006, REG-009 |
| Long Unicode/large text/narrow layout | Rune-count validation, bounded summary text, wrapping controls, adaptive menus, operable tap targets/focus. | FR-013, FR-020, FR-024 | AT-007, AT-010–AT-013, MAN-001, MAN-002 |
| Private-value API/diagnostic failure | Project to bounded endpoint/operation/error class; sentinel folder/URI/account/cursor values absent from UI, route, Sentry, logs, traces, metrics. | FR-025, NFR-001 | AT-013, UT-010, IT-012 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `app/test/feed/models/post_test.dart` | True/folder, true/null, false/null, omitted protected fields, malformed types, copy and explicit clear. | `Post` lacks saved viewer fields. |
| 2 | UT-002 | `app/test/saved_posts/models/saved_post_test.dart` | CamelCase folder/item/page/mutation payloads with opaque cursor and nullable folder. | Typed saved models/mappers do not exist. |
| 3 | UT-003 | `app/test/saved_posts/models/saved_post_folder_test.dart` | TD-003 whitespace/Unicode/slash/control/duplicate/case-variant fixtures. | Client validation and ID identity do not exist. |
| 4 | IT-001 | `app/test/saved_posts/data/saved_post_api_client_test.dart` | Dio adapter for all methods, bodies, scopes/sorts/cursors, explicit null, no-body deletes, and `deleteSaves`. | API client does not exist. |
| 5 | IT-002 | `app/test/saved_posts/data/saved_post_repository_test.dart` | Fake API client/repository contract for every typed operation. | Repository seam does not exist. |
| 6 | UT-004 | `app/test/saved_posts/providers/saved_post_state_provider_test.dart` | Same URI with confirmed/pending/save/move/optimistic/failure/stale snapshot events, exhaustive `AsyncData`/`AsyncLoading`/`AsyncError` projection (including previous data), `Future<void>` commands, and two accounts. | Account URI reducer/provider does not exist. |
| 7 | UT-013 | `app/test/saved_posts/providers/save_post_dialog_controller_test.dart` | Default/current selection, list failure, duplicate IDs, create success/failure, cancel. | Dialog controller does not exist. |
| 8 | IT-004 | `app/test/saved_posts/providers/saved_post_mutation_provider_test.dart` | ProviderContainer, controlled save/move/create futures, duplicate taps, success/failure. | Confirmation/busy/selection behavior is absent. |
| 9 | IT-005 | Same mutation provider suite | Same URI selected by multiple consumers and controlled unsave. | Optimistic update/rollback and stale-snapshot protection are absent. |
| 10 | AT-001 | `app/test/feed/widgets/post_card_test.dart` | Separate full-post variants and protected/compact fixtures; semantics/order inspection. | Bookmark is absent. |
| 11 | AT-002 | `app/test/saved_posts/widgets/save_post_dialog_test.dart` | Multi-page duplicate folders, folder failure, inline create, save completer. | Chooser UI is absent. |
| 12 | AT-003 | State-provider and `post_card_test.dart` | Rapid double tap, successful/failed DELETE, multiple selectors/saved row. | One-shot optimistic unsave is absent. |
| 13 | UT-005 | `app/test/saved_posts/providers/saved_posts_pagination_test.dart` | Initial/next/failure/invalid cursor plus create/rename/delete across partial alphabetical cursor. | Independent merge/restart/reconciliation logic is absent. |
| 14 | IT-003 | `app/test/saved_posts/providers/saved_posts_provider_test.dart` | Controlled folder, Unfiled, and per-folder pages for both sorts; concurrent loads and mutations. | Resource isolation and safe folder-list restart are absent. |
| 15 | UT-006 | `app/test/saved_posts/models/saved_posts_overview_test.dart` | Folder/unfiled/foldered/empty and mismatched `createdAt`/`savedAt` order. | Overview projection is absent. |
| 16 | UT-007 | Saved and notification destination inference suites | Top-level, direct comment, nested reply root/parent/exact URIs. | Saved destination helper is absent. |
| 17 | UT-011 | `app/test/router/saved_posts_route_test.dart` | Typed locations, route names, redacted args. | Canonical/static routes do not exist. |
| 18 | AT-004 | Router, Settings, and overview widget suites | Signed-in router harness; own/visited profiles; folder/unfiled/empty fixtures. | Settings entry/canonical hierarchy is absent. |
| 19 | AT-005 | Provider plus overview/folder page suites | TD-004 independent pages, failure/retry/refresh/invalid cursor and mutations. | Page UI/providers do not exist. |
| 20 | AT-006 | Overview/folder page suites | Saved top-level/comment/reply, move success/failure, unsave. | Row actions and exact navigation are absent. |
| 21 | AT-007 | Overview/folder page suites | Add/rename/delete, duplicate names, static folder route, focus/semantics. | Folder management UI is absent. |
| 22 | IT-007 | `app/test/router/saved_posts_route_test.dart` | Push Settings -> overview -> folder -> back twice and inspect matched locations/names. | Full-screen typed stack is absent. |
| 23 | IT-008 | `app/test/saved_posts/pages/*_test.dart` | Broad widget harness with partial folder pagination, sort, actions, errors, focus, and scroll. | Collection screens are absent. |
| 24 | UT-009 | `appview/internal/api/saved_post_folder_request_test.go` | Absent/false/true/empty/mixed/repeated/invalid/unknown query forms. | Delete query parser does not exist. |
| 25 | UT-012 | `appview/internal/api/saved_post_error_test.go` / `saved_post_test.go` | Missing/cross-owner, validation, and store failures. | Delete mode/error mapping is not represented. |
| 26 | IT-009 | `saved_post_test.go`, `routes_test.go` | `httptest` authenticated route with recording mode store. | Handler cannot select modes or reject bad query. |
| 27 | IT-010 | `appview/internal/api/saved_post_store_test.go` | Real Postgres, multiple owners, preserved timestamps, repeat/concurrent deletion. | Store signature/mode coverage is absent. |
| 28 | IT-011 | Same real-Postgres suite | Visible/hidden saved rows and deterministic folder-delete failure inside transaction. | Remove mode and rollback are absent. |
| 29 | IT-012 | Store/observability + Flutter redaction suites | Alice/Bob/post-author, forbidden collaborators, private sentinels. | Full mode privacy/redaction proof is absent. |
| 30 | AT-008 | Handler/store/observability suites | Compose IT-009–IT-012 outcomes, including unchanged public rows. | Atomic end-to-end acceptance is not proven. |
| 31 | UT-008 | `app/test/shared/widgets/post_summary_test.dart` | Post/quote adapters; visible/protected/unavailable; optional fields; first image only. | Shared summary does not exist. |
| 32 | AT-010 | `post_card_test.dart`, `post_summary_test.dart` | TD-006 quote states and callbacks. | Quote still uses private preview widget. |
| 33 | AT-011 | `notifications_page_test.dart`, destination suite | TD-007 every post-bearing category and destination. | Notification subject still uses inline text. |
| 34 | AT-012 | `saved_post_row_test.dart`, `post_summary_test.dart` | Rich summary plus savedAt/Move/Unsave siblings. | Saved row does not exist. |
| 35 | IT-006 / AT-009 | `account_saved_posts_provider_test.dart` | Alice/Bob fixed clients and delayed list/dialog/save/move/unsave/folder operations. | Saved providers are not registered/account-guarded. |
| 36 | UT-010 | `saved_post_error_test.dart`, existing Sentry sanitizer tests | API errors containing TD-010 sentinels. | Saved error projection/categories are absent. |
| 37 | AT-013 | Relevant Flutter semantics/layout tests and AppView observability tests | TD-010/TD-011, keyboard traversal, text scale 2+, narrow viewport. | Quality/privacy/accessibility boundary is incomplete. |
| 38 | REG-001–REG-005 | Existing post/profile/quote/notification suites | Run unchanged behavior assertions after bookmark/summary/route work. | Any shared UI/model regression becomes visible. |
| 39 | REG-006, REG-007, REG-008, REG-009 | Existing saved AppView and account-boundary suites | Absent-query, response/cursor/policy/reply context, public-row snapshot, activation/cancellation. | Any backend/account regression becomes visible. |
| 40 | MAN-001 / MAN-002 | iOS VoiceOver or Android TalkBack plus narrow large-text target | Follow the approved manual steps after automated gates pass. | Platform announcement/layout issues may remain despite widget tests. |

Focused command progression:

```text
cd app && flutter test test/feed/models/post_test.dart --plain-name "UT-001 saved viewer state"
cd app && flutter test test/saved_posts/models test/saved_posts/data
cd app && flutter test test/saved_posts/providers
cd app && flutter test test/feed/widgets/post_card_test.dart test/saved_posts/widgets
cd app && flutter test test/router/saved_posts_route_test.dart test/settings/settings_page_test.dart test/saved_posts/pages
cd appview && go test ./internal/api ./internal/routes -run 'SavedPost|SavedPostFolder'
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./internal/api ./internal/routes -run 'SavedPost|SavedPostFolder'
cd app && flutter test test/shared/widgets/post_summary_test.dart test/feed/widgets/post_card_test.dart test/notifications/notifications_page_test.dart
```

## 10. Sequencing And Guardrails

- First TDD step: add `UT-001` in `app/test/feed/models/post_test.dart` for `viewerHasSaved` / `viewerSavedFolderId` decode, protected omission defaults, copy preservation, and explicit nullable clear.
- Dependencies between work items:
  - The `Post` fields must exist before saved DTOs and PostSummary adapters can compile against canonical state.
  - Typed API/repository contracts must exist before URI/folder/list providers.
  - The account URI seam must pass save/move/unsave reducer tests before bookmark/dialog UI is wired.
  - Folder pagination/reconciliation must pass UT-005/IT-003 before overview/folder mutation widgets depend on it.
  - Saved route data must exist before generated router output and page route tests.
  - The strict AppView query/parser/store modes must pass before Flutter exposes remove-saves.
  - `PostSummary` extraction lands after saved behavior is stable so quote/notification failures are isolated from mutation failures.
- Guardrails:
  - Private saves/folders remain AppView Postgres data. Do not create PDS records, lexicons, firehose events, public counts, author notifications, or client durable caches.
  - Never enumerate a folder's hydrated rows to implement delete-with-saves.
  - Never use display name as folder identity or expose folder counts/search.
  - Never include folder ID/name, saved URI, cursor, DID pair, or owner-target relation in provider/route `toString`, breadcrumb, log, Sentry context, metric label, analytics dimension, or user-facing raw error.
  - Keep save/move confirmation-driven and unsave optimistic; do not add Undo or success snackbars.
  - Keep compact summary actions parent-owned; do not turn `PostCard` into a multi-mode compact widget or put bookmark/engagement logic into `PostSummary`.
  - Preserve quote moderation/reveal policy and notification routing/category behavior exactly.
  - Keep cursors opaque and list state bounded; no N+1 per-item post/folder lookup.
  - Use existing Dio, Riverpod, dart_mappable, go_router, localization, context-menu, error, Sentry, `httptest`, pgx, and real-Postgres facilities. No runtime dependency is planned.
  - Do not edit generated files manually; run Flutter localization/build-runner generation after source changes.
  - Real-Postgres IT-010–IT-012 must report actual passes with `TEST_DATABASE_URL`; a skipped case is not evidence.
  - Before completion, inspect `app/pubspec.yaml`, `app/pubspec.lock`, `appview/go.mod`, and `appview/go.sum` diffs explicitly per DR-004.
- Out of scope:
  - Lexicon/PDS/firehose changes; nested/shared/smart folders; tags/notes/pins/manual ordering/search/bulk/drag-and-drop; folder/save counts; offline queues; analytics events; a generic normalized post cache; composer preview migrations; OpenAPI/client generation; new dependencies.

Final automated gates after focused tests:

```text
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && flutter gen-l10n
just app-analyze
just app-test
just fmt
just test
git diff --check -- app appview docs/changes/2026-07-21-flutter-saved-posts
git diff -- app/pubspec.yaml app/pubspec.lock appview/go.mod appview/go.sum
```

`just fmt` writes Go formatting as part of implementation verification; it is not run during this planning stage.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking, resolved | Existing post providers hold independent `Post` copies and there is no normalized cache. | Direct cache mutation would duplicate rules and allow bookmark drift. | Use one account URI map plus per-URI selectors; canonical posts seed absent entries and confirmed mutations remain authoritative until account/feature invalidation. |
| CPQ-002 | Non-blocking, resolved | A folder screen needs an opaque ID but route diagnostics must expose neither ID nor name. | Putting the ID in path/query or route name leaks private organization data. | Use static `/profile/settings/saved/folder`, generic route name, and a redacted typed `$extra`; render a localized safe error/back action if extra is absent. |
| CPQ-003 | Non-blocking, resolved | A create/rename can sort outside the currently loaded folder page. | Local insertion could corrupt server order or stale cursor state. | Retain the confirmed entity/selection by ID, discard the cursor, and restart page one; let normal pagination reveal off-page rows. Cover before/after-cursor cases in UT-005/IT-003/IT-008. |
| CPQ-004 | Non-blocking, resolved | `PostSummary` consumers currently show different subsets. | Surface flags could recreate a complex multi-mode widget. | Make data fields optional and let adapters omit unavailable fields; keep outer chrome, metadata, callbacks, and actions in parents. Use one representative image/project title per ASM-005. |
| CPQ-005 | Non-blocking, resolved | Deterministically proving rollback after save-row deletion is harder than testing a failure before the transaction. | A weak test could miss partial remove-saves commits. | In real-Postgres tests, inject failure at folder deletion (for example a temporary trigger) after the save-row statement and assert all rows remain after rollback. |
| CPQ-006 | Non-blocking, resolved | Dart and Go must agree on Unicode folder-name validation without a new dependency. | Client could reject a server-valid name or accept one the server rejects. | Mirror trim, rune count, slash/backslash, and Unicode control checks with the approved TD-003 fixtures; server validation remains authoritative and its field error stays localized/safe. |

Blocking open questions: None.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-07-21-flutter-saved-posts/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-07-21-flutter-saved-posts/05-implementation-plan.md`
- Start with test: `UT-001` in `app/test/feed/models/post_test.dart` for saved viewer decode/default/copy/clear.
- Focused command: `cd app && flutter test test/feed/models/post_test.dart --plain-name "UT-001 saved viewer state"`
- Next tests: `UT-002`, `UT-003`, `IT-001`, and `IT-002` for typed saved/folder/API/repository contracts before provider or widget work.
- Notes:
  - Preserve red-green-refactor order and record actual test evidence/deviations in `05-implementation-plan.md`.
  - Generate and include required mapper/provider/router/localization outputs during implementation, not by hand.
  - Do not stage, commit, push, or create a PR without explicit user authorization.
