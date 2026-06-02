import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Default debounce duration for mention/hashtag autocomplete.
final facetAutocompleteDebounceProvider = Provider<Duration>(
  (ref) => const Duration(milliseconds: 300),
);

/// Mock-backed account suggestions for this Flutter-only slice.
final accountSuggestionRepositoryProvider = Provider<AccountSuggestionRepository>(
  (ref) => MockAccountSuggestionRepository(
    accounts: [
      const AccountSuggestion(
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
        displayName: 'Alice',
        avatar: null,
        isCraftskyProfile: true,
        viewerIsFollowing: true,
      ),
      const AccountSuggestion(
        did: 'did:plc:alicia',
        handle: 'alicia.craftsky.social',
        displayName: 'Alicia',
        avatar: null,
        isCraftskyProfile: true,
        viewerIsFollowing: false,
      ),
      const AccountSuggestion(
        did: 'did:plc:long-display-name',
        handle: 'longname.craftsky.social',
        displayName:
            'This Is An Extremely Long Display Name For Visual Overflow Testing In The Composer Suggestions',
        avatar: null,
        isCraftskyProfile: true,
        viewerIsFollowing: false,
      ),
    ],
  ),
);

/// Mock-backed hashtag suggestions for this Flutter-only slice.
final hashtagSuggestionRepositoryProvider =
    Provider<HashtagSuggestionRepository>(
      (ref) => MockHashtagSuggestionRepository(
        hashtags: [
          const HashtagSuggestion(tag: 'SockKAL', postsLast28Days: 128),
          const HashtagSuggestion(tag: 'sockmending', postsLast28Days: 12),
          ...List.generate(
            50,
            (index) => HashtagSuggestion(
              tag: 'example${index + 1}',
              postsLast28Days: 50 - index,
            ),
          ),
        ],
      ),
    );

/// Facet generator backed by the local/mock mention resolver seam.
final facetGeneratorProvider = Provider<FacetGenerator>(
  (ref) => FacetGenerator(
    mentionResolver: ref.watch(accountSuggestionRepositoryProvider),
  ),
);
