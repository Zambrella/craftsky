# Coding Plan: Flutter Facets UI

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`, high risk)
- User handoff notes for this stage:
  - Treat the three prior documents as source of truth.
  - Preserve the Flutter-only boundary: no AppView implementation, migrations, lexicon changes, PDS calls, or external identity lookup calls.
  - Use the acceptance-test suggested order unless a stronger dependency ordering is identified.
  - Explicitly include implementation-review checks for `descriptionFacets` compatibility handling and third-party helper usage.

## 2. Implementation Strategy

Build a shared Flutter rich-text/facet module under `app/lib/shared/rich_text/`, then wire it into the existing feed and profile seams. This keeps AT Protocol facet parsing/generation, rendering, tap actions, and autocomplete behavior reusable between post text and profile descriptions while preserving the current repository/provider architecture.

The implementation should be pure Flutter/Dart for this slice:

- Generate AT Protocol-compatible raw facet JSON in Flutter at submit/save time.
- Use mock/injected Flutter repositories for mention/hashtag suggestions and handle-to-DID resolution.
- Pass post `facets` through the existing post repository/API/fake path.
- Pass profile `descriptionFacets` through the profile repository/API/fake path even though the current AppView rejects that field; this is an accepted compatibility risk, not a reason to gate the Flutter send.
- Render incoming raw facets defensively from `Post.facets` and new optional `Profile.descriptionFacets` metadata.
- Use `bluesky_text` only for local byte-safe entity detection if implementation inspection confirms the chosen APIs do not make external identity calls. Do not use helper methods that resolve handles externally.

No stronger dependency ordering was found than the acceptance-test order. Use it with one small refinement: introduce minimal shared data contracts before the first generator test so subsequent tests can compile.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Shared rich-text module | No current module; `Post.facets` is raw JSON only | Add facet generation, raw-facet normalization, span building, tap action mapping, and reusable autocomplete editor/controller | BR-001, FR-001, FR-004, FR-005, FR-006, FR-007, FR-008, FR-012, FR-013, NFR-001, NFR-002, NFR-005, RULE-001, RULE-002, RULE-003, RULE-005, RULE-006, RULE-007, RULE-008, RULE-009 | UT-001 through UT-020, AT-003 through AT-008 |
| Post composer | `PostComposerSheet` uses `BrandTextField`, stores `_text`, calls `createPostProvider.create(text, reply, images)` | Replace body input with shared facet autocomplete editor, recompute facets from current text on submit, pass facets through provider/repository/API/fake | FR-001, FR-002, FR-006, FR-008, FR-010, FR-011, FR-012, NFR-004, RULE-002 | AT-001, AT-003, AT-004, AT-007, IT-003, IT-012, REG-001, REG-002, REG-003 |
| Post rendering | `PostCard` renders `Text(post.text)` | Render `FacetedText` when `post.facets` is available; fall back safely for null/invalid facets | FR-004, FR-013, NFR-005, RULE-009 | AT-005, AT-006, IT-008, REG-006 |
| Post write data path | `PostRepository.create`, `ApiPostRepository.create`, `PostApiClient.createPost`, `FakePostRepository.onCreate` accept text/reply/images | Add optional `List<Map<String, dynamic>>? facets` and include `facets` in JSON body only when non-empty | FR-002 | IT-001, IT-002, IT-003, AT-001 |
| Profile edit flow | `EditProfileDialog` uses `BrandTextField` bio field and `saveProfileProvider.save(description, ...)` | Replace bio input with shared facet autocomplete editor, recompute `descriptionFacets` on save, preserve dirty/validation/image behavior | FR-003, FR-007, FR-008, NFR-004, RULE-004 | AT-002, AT-003, AT-004, IT-004, IT-006, IT-011, IT-012, REG-004, REG-005 |
| Profile model/rendering | `Profile` has `description` only; `ProfileBio` renders plain `Text` | Add optional `descriptionFacets`; pass to `ProfileBio`; render with `FacetedText` defensively | FR-005, FR-013, NFR-005, RULE-009 | AT-005, IT-009, profile model tests |
| Profile write data path | `ProfileRepository.updateMe`, `ApiProfileRepository`, `ProfileApiClient.updateMyProfile`, `FakeProfileRepository` accept display/profile/media fields | Add optional `descriptionFacets` through each seam and include `descriptionFacets` in request body when non-empty | FR-003, RULE-004 | IT-004, IT-006, IT-011, AT-002 |
| Routing/actions | Typed `SearchRoute` has no query state; profile route exists; `url_launcher` is used only in auth | Add optional `tag` query parameter to `SearchRoute`/`SearchPage`; add shared/fakeable link launcher seam for facet links; mention actions navigate by visible handle | FR-013 | UT-014, IT-010, AT-006 |
| Dependencies/codegen | Riverpod/go_router/dart_mappable generated files are committed; `bluesky_text` is not present | Implementation may add `bluesky_text` and run `flutter pub get`; provider/model/router changes require build_runner-generated files | NFR-001, RULE-001 | GAP-002, UT-017, IT-007 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/pubspec.yaml`, `app/pubspec.lock` | Change, if helper accepted | Add `bluesky_text` only if implementation inspection confirms local detection APIs can be used without external identity resolution | NFR-001, RULE-001 | GAP-002, UT-017, IT-007 |
| `app/lib/shared/rich_text/models/facet_models.dart` | Create | Local models for raw AT Protocol facets, byte ranges, normalized render segments, account suggestions, hashtag suggestions, and tap intents | FR-001, FR-004, FR-005, FR-010, FR-013 | UT-009, UT-010, UT-012, UT-013, UT-014 |
| `app/lib/shared/rich_text/facet_generator.dart` | Create | Generate raw `app.bsky.richtext.facet`-compatible JSON from current text using byte-safe entity detection plus injected Craftsky mention resolver | FR-001, FR-012, NFR-001, RULE-002, RULE-003, RULE-007, RULE-008 | UT-001 through UT-008, UT-018, AT-001, AT-002 |
| `app/lib/shared/rich_text/faceted_text_model.dart` | Create | Parse/normalize incoming raw facet JSON for rendering; sort valid ranges; reject invalid UTF-8 boundaries; ignore unsupported features | FR-004, FR-005, RULE-009 | UT-009, UT-010, AT-005 |
| `app/lib/shared/rich_text/faceted_text_span_builder.dart` | Create | Build theme-aware `TextSpan`s from normalized facets; faceted spans use `Theme.colorScheme.primary` | NFR-005 | UT-016, AT-005 |
| `app/lib/shared/rich_text/widgets/faceted_text.dart` | Create | Render rich text on post/profile surfaces and dispatch tap actions safely | FR-004, FR-005, FR-013 | AT-005, AT-006, IT-008, IT-009, IT-010 |
| `app/lib/shared/rich_text/facet_action_handler.dart` | Create | Map mention/link/hashtag taps to profile route, fakeable URL launcher, and search route; swallow failures without crashing | FR-013 | UT-014, IT-010, AT-006 |
| `app/lib/shared/rich_text/providers/facet_action_providers.dart` | Create | Provider for link launcher callback backed by `url_launcher.launchUrl`; tests override it | FR-013 | IT-010, AT-006 |
| `app/lib/shared/rich_text/data/facet_suggestion_repository.dart` | Create | Interfaces for account suggestions, hashtag suggestions, and handle resolution | FR-011, RULE-001, RULE-002 | UT-017, IT-007, AT-008 |
| `app/lib/shared/rich_text/data/mock_facet_suggestion_repository.dart` | Create | Flutter-only mock data, filtering/sorting, hashtag counts, and local DID resolution | BR-002, FR-010, FR-011, RULE-001 | UT-012, UT-013, IT-005, IT-007, AT-008 |
| `app/lib/shared/rich_text/providers/facet_suggestion_providers.dart` and `.g.dart` | Create | Riverpod provider graph for suggestion repositories, resolver, generator, debounce duration, and auto-disposed debounced suggestion request families | FR-011, NFR-002, RULE-001 | IT-005, IT-007, AT-008 |
| `app/lib/shared/rich_text/facet_autocomplete_controller.dart` | Create | Pure Dart controller/helpers for active-token detection, stale-token guards, and token replacement; async debounce belongs in Riverpod suggestion providers | FR-006, FR-007, FR-008, NFR-002, RULE-005, RULE-006 | UT-011, UT-015, UT-019, UT-020, AT-007 |
| `app/lib/shared/rich_text/widgets/facet_autocomplete_editor.dart` | Create | Reusable editor that composes `BrandTextField`, a facet-aware text controller, and visible/semantic suggestion dropdown | FR-006, FR-007, FR-008, FR-009, FR-010, NFR-003, NFR-005 | AT-003, AT-004, AT-007, IT-005 |
| `app/lib/shared/rich_text/widgets/facet_suggestion_tiles.dart` | Create | Account tile with avatar/display name/handle and hashtag tile with 28-day count label; mention no-results state | FR-009, FR-010, NFR-003 | UT-012, UT-013, AT-003, AT-004 |
| `app/lib/feed/widgets/post_composer_sheet.dart` | Change | Use autocomplete editor for body; generate facets on submit from final text; preserve image/reply/discard behavior | FR-001, FR-002, FR-006, FR-008, NFR-004 | AT-001, AT-003, AT-004, IT-012, REG-001 through REG-003 |
| `app/lib/feed/providers/create_post_provider.dart` and `.g.dart` if needed | Change | Add optional `facets` argument and pass to repository; preserve cache-prepend behavior | FR-002 | IT-003, AT-001 |
| `app/lib/feed/data/post_repository.dart` | Change | Add optional raw `facets` parameter to `create` | FR-002 | IT-002, IT-003 |
| `app/lib/feed/data/api_post_repository.dart` | Change | Forward `facets` to API client | FR-002 | IT-002 |
| `app/lib/feed/data/post_api_client.dart` | Change | Include non-empty `facets` in `/v1/posts` JSON body alongside text/reply/images | FR-002 | IT-001 |
| `app/test/feed/fakes/fake_post_repository.dart` | Change | Capture/pass optional `facets` in `onCreate` | FR-002 | IT-003, AT-001 |
| `app/lib/feed/widgets/post_card.dart` | Change | Render `FacetedText` for `post.text`/`post.facets`; ensure facet taps do not crash or unintentionally invoke card taps | FR-004, FR-013 | AT-005, AT-006, IT-008, REG-006 |
| `app/lib/profile/models/profile.dart`, `.mapper.dart` | Change | Add optional `List<Map<String, dynamic>>? descriptionFacets`; run build_runner | FR-005, FR-003 | IT-009, IT-011 |
| `app/lib/profile/widgets/profile_bio.dart` | Change | Accept facets and render `FacetedText`; keep empty-description behavior | FR-005 | AT-005, IT-009 |
| `app/lib/profile/widgets/profile_meta_section.dart` | Change | Pass `profile.descriptionFacets` into `ProfileBio` | FR-005 | IT-009 |
| `app/lib/profile/pages/edit_profile_dialog.dart` | Change | Use autocomplete editor for bio, generate `descriptionFacets` at save, keep FormBuilder dirty/validation/focus behavior | FR-003, FR-007, FR-008, NFR-004 | AT-002, AT-003, AT-004, REG-004, REG-005 |
| `app/lib/profile/providers/save_profile_provider.dart` and `.g.dart` if needed | Change | Add optional `descriptionFacets` argument and forward to repository | FR-003, RULE-004 | IT-011, AT-002 |
| `app/lib/profile/data/profile_repository.dart` | Change | Add optional raw `descriptionFacets` parameter to `updateMe` | FR-003 | IT-011 |
| `app/lib/profile/data/api_profile_repository.dart` | Change | Forward `descriptionFacets` to API client | FR-003 | IT-004, IT-011 |
| `app/lib/profile/data/profile_api_client.dart` | Change | Include non-empty `descriptionFacets` in `/v1/profiles/me` body; preserve avatar/banner clear semantics | FR-003, RULE-004 | IT-004, IT-006 |
| `app/test/profile/fakes/fake_profile_repository.dart` | Change | Capture/pass optional `descriptionFacets` in `onUpdateMe` | FR-003 | AT-002, IT-011 |
| `app/lib/router/router.dart`, `router.g.dart`, `route_locations.dart` if needed | Change | Add optional `tag` query parameter to `SearchRoute`; preserve existing `/search` location | FR-013 | IT-010, router tests |
| `app/lib/search/pages/search_page.dart` | Change | Accept/display or hold optional hashtag context without implementing results | FR-013 | IT-010, `search_page_test.dart` |
| `app/lib/l10n/*.arb`, generated localizations | Change if needed | Add labels such as `No results`, hashtag 28-day count, autocomplete semantics | FR-009, FR-010, NFR-003 | AT-003, AT-004, MAN-001 |
| `app/test/shared/rich_text/*_test.dart` | Create | Unit/widget/provider tests listed in `02-acceptance-tests.md` | All rich-text requirements | UT-001 through UT-020, AT-003 through AT-008 |
| Existing feed/profile/router/search tests | Change/extend | Add payload/rendering/regression cases without weakening existing assertions | FR-002, FR-003, FR-004, FR-005, FR-013, NFR-004 | IT-001 through IT-012, REG-001 through REG-006 |

## 5. Services, Interfaces, And Data Flow

### 5.1 Core raw facet shape

Use raw `Map<String, dynamic>` at repository/API boundaries to match existing `Post.facets` and avoid generating lexicon-derived Flutter classes in this slice.

```text
// Pseudo-shape only.
facet = {
  'index': {'byteStart': int, 'byteEnd': int},
  'features': [
    {'$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:...'}
    // or {'$type': 'app.bsky.richtext.facet#link', 'uri': 'https://...'}
    // or {'$type': 'app.bsky.richtext.facet#tag', 'tag': 'SockKAL'}
  ],
}
```

Generated ranges must be byte offsets, inclusive start/exclusive end, non-overlapping, and valid UTF-8 boundaries. Renderer input may be invalid and must be normalized defensively.

### 5.2 Facet generator

Create a small pure Dart service responsible for converting final text to raw facets.

```text
abstract interface class MentionResolver {
  Future<String?> didForHandle(String handle); // handle without leading @
}

class FacetGenerator {
  FacetGenerator({required MentionResolver mentionResolver});

  Future<List<Map<String, dynamic>>> generate(String text);
}
```

Generation rules:

1. Analyze the current final text at submit/save time. Do not trust stale autocomplete selection ranges.
2. Use `bluesky_text.BlueskyText(text).entities` only if it satisfies the helper guardrails below. Prefer its `Entity.value` and `ByteIndices` for local handle/link/tag detection. Do **not** use `Entity.toFacet()` or any helper that resolves handles externally.
3. Mentions:
   - Detect syntactically valid visible `@handle` tokens.
   - Resolve handle-to-DID only through injected/mock `MentionResolver` data.
   - Generate mention facets only for resolved Craftsky accounts.
   - Leave unknown handles as plain text with no warning.
4. Links:
   - Recognize explicit `http://`/`https://` URLs and bare domains.
   - Normalize bare-domain facet `uri` to `https://...` while leaving visible text unchanged.
   - Exclude common trailing sentence punctuation and unmatched closing brackets/parentheses from both range and URI.
   - Do not implement hidden markdown targets.
5. Hashtags:
   - Preserve user-typed casing in facet `tag` value.
   - Exclude leading `#`.
   - Include Unicode letters, Unicode digits, and underscore only; stop before hyphen or emoji for this slice.
6. Overlaps:
   - Prefer the earlier/wider link range when a hashtag-like fragment appears inside a URL.
   - Drop overlapping lower-priority generated entities rather than emitting invalid overlaps.

### 5.3 Incoming facet normalization for rendering

Create a separate normalizer so rendering can tolerate future/malformed AppView/PDS data without coupling to generation assumptions.

```text
class FacetedTextModel {
  static List<NormalizedFacetRange> fromRaw({
    required String text,
    required List<Map<String, dynamic>>? rawFacets,
  });
}

sealed class FacetFeature { mention(did), link(uri), tag(tag) }
class NormalizedFacetRange { int charStart; int charEnd; FacetFeature feature; }
```

Renderer normalization rules:

- Convert byte ranges to character ranges only when both ends map to UTF-8 boundaries.
- Drop only the invalid facet whose byte range is malformed, out of range, or splits a multibyte character.
- Sort valid ranges by start offset.
- Ignore unsupported/unknown feature variants.
- When multiple supported features appear on one facet range, use the first supported feature in incoming feature order.
- For overlapping incoming ranges, keep deterministic valid ranges and ignore/drop conflicting later ranges so rendering never throws.

### 5.4 Suggestion and resolver repositories

Keep mock data behind interfaces that can later be backed by AppView endpoints without changing editor widgets.

```text
abstract interface class AccountSuggestionRepository implements MentionResolver {
  Future<List<AccountSuggestion>> searchAccounts(String query);
}

abstract interface class HashtagSuggestionRepository {
  Future<List<HashtagSuggestion>> searchHashtags(String query);
}

class AccountSuggestion {
  String did;
  String handle;
  String? displayName;
  String? avatar;
  bool isCraftskyProfile;
  bool viewerIsFollowing;
}

class HashtagSuggestion {
  String tag; // canonical/display casing, no leading # in storage model is acceptable
  int postsLast28Days;
}
```

Mock account repository rules:

- Filter to matching `isCraftskyProfile == true` accounts only.
- Sort followed matching accounts before non-followed matching accounts.
- Expose display name, avatar, handle, DID for UI and resolver use.

Mock hashtag repository rules:

- Filter by query after `#`.
- Preserve repository display/canonical casing for insertion.
- Expose 28-day count for UI.

### 5.5 Tap action flow

Keep action dispatch safe and fakeable.

```text
enum FacetTapIntent { mention(handle), link(uri), hashtag(tag) }

class FacetActionHandler {
  Future<void> handle(BuildContext context, FacetTapIntent intent);
}
```

- Mention: derive destination from visible faceted text (`@alice.craftsky.social` -> `alice.craftsky.social`), validate syntactically enough to avoid bad routes, then navigate with `UserProfileRoute(handle: handle)` or equivalent route API. DID metadata is not used for navigation in this slice.
- Link: call an injectable launcher provider backed by `url_launcher.launchUrl`. Failures must not crash.
- Hashtag: navigate to `/search?tag=SockKAL` via typed `SearchRoute(tag: tag)` or an equivalent query-parameter route. Search results remain out of scope.

## 6. State, Providers, Controllers, Or DI

Use existing Riverpod provider conventions (`@riverpod`, generated `.g.dart`) for shared services and mocks. For async autocomplete lookups, follow Riverpod's official debounce/cancel pattern from `https://riverpod.dev/docs/how_to/cancel#debouncing-requests`: use auto-disposed provider-family requests, `ref.onDispose`, an awaited debounce delay, and cancellation before the repository call starts if the provider was disposed during the delay.

Suggested graph:

```text
facetAutocompleteDebounceProvider -> Duration(milliseconds: 300)

accountSuggestionRepositoryProvider -> MockAccountSuggestionRepository(seed data)
hashtagSuggestionRepositoryProvider -> MockHashtagSuggestionRepository(seed data)
mentionResolverProvider -> accountSuggestionRepositoryProvider

facetGeneratorProvider -> FacetGenerator(mentionResolverProvider)
facetUrlLauncherProvider -> (Uri uri) => url_launcher.launchUrl(uri, ...)

accountSuggestionsProvider(query) -> auto-disposed debounced repository search
hashtagSuggestionsProvider(query) -> auto-disposed debounced repository search
```

Provider choices:

- Use `Provider`/`@Riverpod(keepAlive: true)` for repositories and stateless services.
- Use `FutureProvider`/generated `Future` provider families for `accountSuggestions(query)` and `hashtagSuggestions(query)` so each active query has its own auto-disposed request lifecycle.
- Keep `FacetAutocompleteEditor` local state responsible for caret/token detection, selected active query, focus, and rendering the current `AsyncValue`; let Riverpod own async debounce/cancel for suggestion lookups.
- Keep the debounce duration provider overrideable in widget tests. Tests can set it to a short duration or zero where appropriate.
- Do not inject `Dio`, AppView clients, PDS clients, or atproto identity clients into suggestion/resolver providers for this slice.

Debounced suggestion provider sketch:

```text
@riverpod
Future<List<AccountSuggestion>> accountSuggestions(
  Ref ref,
  String query,
) async {
  var didDispose = false;
  ref.onDispose(() => didDispose = true);

  await Future<void>.delayed(ref.watch(facetAutocompleteDebounceProvider));
  if (didDispose) throw Exception('Cancelled');

  return ref.watch(accountSuggestionRepositoryProvider).searchAccounts(query);
}
```

Notes for implementation:

- The provider should not be `keepAlive`; disposal is how superseded queries and unmounted editors are cancelled.
- Throwing after disposal is acceptable per the Riverpod docs; Riverpod will ignore that disposed provider result.
- Because repositories are mock/local in this slice, no HTTP client cancellation is needed, but the same `ref.onDispose` pattern prevents stale lookups from applying.
- If future AppView-backed autocomplete is added, this provider is where request cancellation should be attached to the network client/cancel token.

Autocomplete controller responsibilities:

```text
ActiveToken? detectActiveToken(TextEditingValue value);
TextEditingValue replaceActiveToken({
  required TextEditingValue current,
  required ActiveToken token,
  required String replacementWithSingleTrailingSpace,
});
```

Detection rules:

- Activate only at start of text or after whitespace/opening punctuation such as `(`.
- Require at least one query character after `@`/`#`.
- Do not activate inside words, email addresses, URL fragments, or bare trigger characters.
- Hide/ignore stale results when caret or token changes before debounce completes; the editor should stop watching the old provider family key and watch the new/empty key so Riverpod disposes the old request.

Editable primary-color token styling:

- Prefer a `FacetTextEditingController extends TextEditingController` that overrides `buildTextSpan` so the existing `BrandTextField` can keep using a normal `TextField` while active mention/hashtag tokens use `Theme.colorScheme.primary`.
- If broader styling is needed, extend `BrandTextField` minimally to accept a specialized controller/field builder rather than duplicating the full Craftsky text-field style.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Shared editor composition

`FacetAutocompleteEditor` should wrap `BrandTextField` and render suggestions below the field (or via an anchored overlay if straightforward). Keep visible text and semantics testable.

```text
FacetAutocompleteEditor(
  label,
  hintText,
  controller,
  focusNode,
  minLines,
  maxLines,
  enabled,
  errorText,
  helperText,
  onChanged,
)
```

UI behavior:

- Mention dropdown: avatar, display name, handle; followed before non-followed; `No results` state when query returns no matches.
- Hashtag dropdown: canonical/display `#tag`, visible 28-day post count; hide dropdown when query returns no matches.
- Selection replaces only the active token and appends exactly one trailing space.
- Focus remains in the editor; caret moves after the trailing space.
- Link facets are generated at submit/save/render time and never show an autocomplete dropdown.

### 7.2 Post composer

Replace only the body text field in `PostComposerSheet`; keep:

- `_controller`, `_focusNode`, `_text`, `_initialText` draft tracking.
- Empty/overlong validation and character helper.
- Reply prefill `@author.handle `.
- Image selection, alt-text warning, order, upload, and reply image restrictions.
- Discard confirmation.

On submit:

```text
final trimmedText = _text.trim();
final facets = await ref.read(facetGeneratorProvider).generate(trimmedText);
await ref.read(createPostProvider.notifier).create(
  text: trimmedText,
  reply: _replyFor(widget.replyTarget),
  images: ...,
  facets: facets.isEmpty ? null : facets,
);
```

### 7.3 Profile editor

Replace only the bio `BrandTextField` with `FacetAutocompleteEditor` inside the existing `FormBuilderField<String>`.

Keep:

- FormBuilder state and field names.
- Shared focus-node behavior that avoids validation stealing focus.
- Dirty-state and max-length validation.
- Avatar/banner draft, upload, clear, and save semantics.
- Full atomic profile update behavior.

On save, generate facets from the final trimmed bio and pass `descriptionFacets` through `saveProfileProvider.save`. Do not add a Flutter compatibility gate for the current AppView rejection.

### 7.4 Rendered text

Use `FacetedText` in:

- `PostCard` body.
- `ProfileBio`.

Keep plain rendering when facets are null/empty/invalid. Faceted spans use `Theme.colorScheme.primary`; non-faceted spans use existing text styles.

### 7.5 Routes

Change `SearchRoute` from a no-argument route to an optional query-parameter route.

```text
class SearchRoute extends GoRouteData with $SearchRoute {
  const SearchRoute({this.tag});
  final String? tag;
  Widget build(...) => SearchPage(tag: tag);
}
```

`SearchPage` may simply display/hold the tag context for now. Do not implement search results.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Bare `@` or `#` | No query and no dropdown | RULE-005 | UT-011, UT-019, AT-007 |
| Trigger inside word/email/URL fragment | No query and no dropdown | RULE-005 | UT-011, UT-019, AT-007 |
| Caret moves while debounce pending | Editor watches a different/no provider-family key, causing Riverpod to dispose the old debounced request; hide stale dropdown | NFR-002 | UT-015, AT-007 |
| Mention query has no results | Show visible/semantic `No results`; do not block typing | FR-006, FR-011 | AT-003 |
| Hashtag query has no results | Hide dropdown; allow free typed hashtag | FR-008, FR-011 | AT-004 |
| Unknown manually typed mention | No mention facet, text unchanged, no warning/block | RULE-002 | UT-001, UT-007, AT-001, AT-002 |
| Manually typed resolvable mention | Generate mention facet from local resolver, even without autocomplete selection | RULE-002 | UT-008, AT-001, AT-002 |
| Multibyte text before entity | Generated byte offsets use UTF-8 bytes and valid boundaries | NFR-001 | UT-002, AT-001 |
| Link followed by punctuation/unmatched closer | Exclude punctuation/closer from range and URI | RULE-008 | UT-005, AT-001 |
| Hashtag with hyphen/emoji | Generate only Unicode letter/digit/underscore run; preserve casing | RULE-007 | UT-018, AT-004 |
| Incoming facet unsupported feature | Ignore unsupported feature; render remaining valid text safely | RULE-009 | UT-009, AT-005 |
| Incoming facet splits UTF-8 character | Drop only invalid facet, no exception | RULE-009 | UT-010, AT-005 |
| Link launcher failure | Swallow/surface safely without crash | FR-013 | UT-014, AT-006, MAN-004 |
| Current AppView rejects `descriptionFacets` | Existing save error path handles failure; tests/docs mark as known compatibility risk | FR-003, RULE-004 | IT-006, MAN-003 |
| Existing composer/profile behavior | Preserve existing tests and add facet cases without weakening assertions | NFR-004 | REG-001 through REG-006 |

## 9. Test Implementation Plan

Use the suggested order from `02-acceptance-tests.md`.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 0 | compile scaffolding | Minimal shared model/interface files | Empty models/providers only as needed for first tests | New tests do not compile until shared module exists |
| 1 | UT-002 | `app/test/shared/rich_text/facet_generator_test.dart` | `🧶 café @alice.craftsky.social #Mending` with local resolver | No `FacetGenerator` / wrong byte offsets |
| 2 | UT-001 | Same | Alice DID local resolver plus unknown handle | Mention facets not generated/filtered correctly |
| 3 | UT-004 | Same | HTTP, HTTPS, bare domain | Link facets missing or URI normalization absent |
| 4 | UT-005 | Same | `craftsky.social, (https://example.com/path).` | Link ranges include punctuation/closers |
| 5 | UT-018 | Same | Unicode/hyphen/emoji hashtag examples | Hashtag parser too broad/narrow |
| 6 | UT-003 | Same | URL fragment plus standalone hashtag | Generated facets overlap |
| 7 | UT-009 | `app/test/shared/rich_text/faceted_text_model_test.dart` | Unsorted/overlapping/unsupported/multi-feature/out-of-range raw facets | Normalizer missing or throws |
| 8 | UT-010 | Same | Bad byte range into emoji plus good range | Invalid facet not isolated |
| 9 | UT-016 | `app/test/shared/rich_text/faceted_text_span_builder_test.dart` | Theme primary color and sample normalized ranges | Faceted spans not styled primary |
| 10 | UT-011 | `app/test/shared/rich_text/facet_autocomplete_controller_test.dart` | Boundary/minimum query cases | Token detection absent/incorrect |
| 11 | UT-015 | Same plus provider tests where useful | Fake/injected debounce and rapid edits using Riverpod auto-dispose provider families | Queries fire too early/often or disposed queries still apply |
| 12 | UT-020 | Same | Token replacement in middle/end of text | Replacement changes surrounding text or caret |
| 13 | UT-012 | `app/test/shared/rich_text/mock_account_suggestion_repository_test.dart` | Followed/non-followed/non-Craftsky accounts | Filtering/sorting/display data missing |
| 14 | UT-013 | `app/test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart` | Hashtags with 28-day counts | Count/casing unavailable |
| 15 | IT-001 | `app/test/feed/data/post_api_client_test.dart` | DioAdapter expecting `facets` in `/v1/posts` body | Client signature/body lacks facets |
| 16 | IT-002 | `app/test/feed/data/post_repository_test.dart` | API client fake/mock behind `ApiPostRepository` | Repository does not forward facets |
| 17 | IT-003 | `app/test/feed/providers/create_post_provider_test.dart` | Fake repository captures facets | Provider does not pass facets |
| 18 | IT-004 | `app/test/profile/data/profile_api_client_test.dart` | DioAdapter expecting `descriptionFacets` body | Client signature/body lacks descriptionFacets |
| 19 | AT-003 | Shared editor + feed/profile widget tests | Provider-overridden account suggestions and debounce | Mention dropdown absent |
| 20 | AT-004 | Shared editor + feed/profile widget tests | Provider-overridden hashtag suggestions | Hashtag dropdown/count/insert absent |
| 21 | AT-001 | `app/test/feed/widgets/post_composer_sheet_facets_test.dart` | Composer submits mixed entities to fake repo | Submit path omits generated facets |
| 22 | AT-002 | `app/test/profile/edit_profile_dialog_facets_test.dart` | Profile editor saves mixed entities to fake repo | Save path omits descriptionFacets |
| 23 | AT-005 | `app/test/shared/rich_text/faceted_text_test.dart`, `post_card_test.dart`, `profile_bio_test.dart` | Valid/malformed facets and theme | Rendering not rich/safe |
| 24 | AT-006 / UT-014 / IT-010 | `faceted_text_actions_test.dart`, router/search tests | Fake router/launcher, failing launcher | Tap intents absent or crash |
| 25 | REG-001 through REG-006 | Existing composer/profile/post tests | Existing fixtures plus one facets-with-images case | Regression in validation/images/replies/dirty/save/card rendering |

Focused commands for the TDD builder:

```text
cd app && flutter test test/shared/rich_text/facet_generator_test.dart
cd app && flutter test test/shared/rich_text/faceted_text_model_test.dart test/shared/rich_text/faceted_text_span_builder_test.dart
cd app && flutter test test/shared/rich_text/facet_autocomplete_controller_test.dart test/shared/rich_text/mock_account_suggestion_repository_test.dart test/shared/rich_text/mock_hashtag_suggestion_repository_test.dart
cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/create_post_provider_test.dart
cd app && flutter test test/profile/data/profile_api_client_test.dart test/profile/edit_profile_dialog_facets_test.dart
cd app && flutter test test/feed/widgets/post_composer_sheet_facets_test.dart test/profile/edit_profile_dialog_facets_test.dart test/shared/rich_text/facet_autocomplete_editor_test.dart
cd app && flutter test test/shared/rich_text/faceted_text_test.dart test/shared/rich_text/faceted_text_actions_test.dart test/feed/widgets/post_card_test.dart test/profile/widgets/profile_bio_test.dart
```

Codegen/dependency commands when needed:

```text
cd app && flutter pub get
cd app && dart run build_runner build --delete-conflicting-outputs
```

## 10. Sequencing And Guardrails

- First TDD step: `UT-002` in `app/test/shared/rich_text/facet_generator_test.dart` for UTF-8 byte offsets with emoji/multibyte text before mention/link/hashtag facets.
- Dependency ordering:
  1. Shared facet models/interfaces.
  2. Facet generation.
  3. Renderer normalization and span styling.
  4. Autocomplete token detection/replacement, Riverpod debounced suggestion providers, and mock repositories.
  5. Post/profile repository/API/provider/fake payload propagation.
  6. Post/profile editor widgets.
  7. Rendered surfaces and tap actions.
  8. Regression tests and manual checks.
- Guardrails:
  - Flutter-only: no AppView code, migrations, SQL, lexicon, Tap/PDS, or external identity lookup calls.
  - Do not store PDS tokens or introduce a PDS client.
  - Do not call AppView for autocomplete in this slice.
  - Do not use `bluesky_text.Entity.toFacet()` or any helper path that can resolve handles externally; construct facets manually from local entity byte indices and injected resolver data.
  - Keep raw facet payloads camelCase where AppView JSON expects it (`facets`, `descriptionFacets`, `byteStart`, `byteEnd`).
  - Recompute facets from current final text at submit/save time.
  - Implement autocomplete debounce/cancel through Riverpod auto-disposed provider families using `ref.onDispose`, not only ad hoc widget timers.
  - Preserve existing image/reply/profile atomic-update semantics.
  - Keep `descriptionFacets` live-send incompatibility visible in test names/comments and implementation review.
- Out of scope:
  - AppView profile `descriptionFacets` support.
  - AppView autocomplete endpoints.
  - Lexicon changes.
  - Migrations/persistence changes.
  - PDS/external identity resolution.
  - Hashtag search results.
  - Website preview cards or markdown links.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking risk | Profile `descriptionFacets` are intentionally sent before AppView supports them | Live profile saves with generated `descriptionFacets` can fail with the current AppView | Keep Flutter send; isolate in API/repository tests; ensure existing error UI handles failure; follow-up backend/API slice required |
| CPQ-002 | Non-blocking risk | Third-party helper usage may accidentally perform external handle resolution | Violates Flutter-only architecture and RULE-001 | Implementation review must inspect imports/API calls; use only local entity/byte-index APIs; do not call `toFacet()` or identity endpoints; tests must run without network clients |
| CPQ-003 | Non-blocking risk | `bluesky_text` built-in parsing may not exactly match Craftsky rules for bare domains, trailing punctuation, or hashtag character limits | Generated facets may fail acceptance edge cases | Wrap/post-filter helper entities; add custom byte-range adjustment where helper behavior diverges; keep UT-004/UT-005/UT-018 as authority |
| CPQ-004 | Non-blocking risk | `BrandTextField` does not expose every low-level `TextField` option | Rich editable styling/autocomplete may require careful composition | Prefer specialized `TextEditingController.buildTextSpan` and wrapper composition; only minimally extend `BrandTextField` if needed |
| CPQ-005 | Non-blocking risk | Gesture recognizers in `FacetedText` inside tappable post cards may compete with card tap | Facet taps may also navigate to thread or fail to invoke facet action | Add widget tests for facet tap vs card tap; adjust recognizers/InkWell composition if needed |
| CPQ-006 | Non-blocking risk | Search route tag context is ahead of real search results | Users may land on a placeholder search page | Only preserve route/query context in this slice; search results remain follow-up work |
| CPQ-007 | Non-blocking risk | Model/provider/router changes require generated files | Build/test failures if codegen is missed | Run `dart run build_runner build --delete-conflicting-outputs` after modifying `@riverpod`, `@TypedGoRoute`, or `@MappableClass` files |
| CPQ-008 | Non-blocking risk | Debounce implemented only in widget-local timers instead of Riverpod request lifecycle | Stale autocomplete requests may apply after caret changes/unmounts and diverge from project Riverpod patterns | Use Riverpod's documented `ref.onDispose` debounce/cancel approach for suggestion provider families; keep widget/controller code limited to token state and rendering |

## 12. Implementation-Review Checklist

The implementation reviewer should explicitly verify these items before approving the implementation stage.

### `descriptionFacets` compatibility handling

- [ ] Flutter sends `descriptionFacets` through `EditProfileDialog` -> `saveProfileProvider` -> `ProfileRepository.updateMe` -> `ApiProfileRepository` -> `ProfileApiClient.updateMyProfile` -> fake repository seams.
- [ ] The production API-client test includes `descriptionFacets` in the `/v1/profiles/me` JSON body without changing AppView code.
- [ ] `IT-006` or equivalent simulates the current AppView `unexpected_field`/bad-request behavior and verifies Flutter uses the existing save-error path without crashing.
- [ ] No temporary Flutter gate silently strips `descriptionFacets` for compatibility.
- [ ] The code or test comments keep the follow-up AppView/API requirement visible.

### Third-party helper and Flutter-only boundary

- [ ] If `bluesky_text` is added, usage is limited to local text/entity/byte-index analysis such as `BlueskyText(...).entities`, `handles`, `links`, `tags`, and `ByteIndices`.
- [ ] The code does **not** call `Entity.toFacet()`, `bluesky` identity APIs, PDS APIs, `resolveHandle`, `resolveIdentity`, `resolveDid`, or any network-backed helper for mention resolution.
- [ ] The shared rich-text module does not import `dio`, AppView API clients, PDS clients, OAuth/token packages, or generated AppView/backend code.
- [ ] Mention DID mapping uses only the injected/mock `MentionResolver` seam.
- [ ] Autocomplete provider overrides in tests pass without AppView, PDS, or external network setup.
- [ ] No files under `appview/`, `lexicon/`, migrations, or backend docs are changed for this slice.

## 13. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-06-01-flutter-facets-ui/04-coding-plan.md`
- TDD execution plan: create/update `05-implementation-plan.md` only if the next workflow stage requires it; otherwise implement directly from the approved documents.
- Start with test: `UT-002` in `app/test/shared/rich_text/facet_generator_test.dart`.
- First focused command: `cd app && flutter test test/shared/rich_text/facet_generator_test.dart`
- Source of truth: `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this coding plan.
- Notes:
  - Keep this slice Flutter-only.
  - Use the acceptance-test suggested order.
  - Treat live profile `descriptionFacets` failure as a known compatibility risk until a future AppView/API slice lands.
  - Preserve all existing composer/profile/post-card behavior while adding facets.
