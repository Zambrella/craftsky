import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Default debounce duration for mention/hashtag autocomplete.
final facetAutocompleteDebounceProvider = Provider<Duration>(
  (ref) => const Duration(milliseconds: 300),
);

/// Mock-backed account suggestions for this Flutter-only slice.
final accountSuggestionRepositoryProvider =
    Provider<AccountSuggestionRepository>(
      (ref) => const MockAccountSuggestionRepository(
        accounts: [
          AccountSuggestion(
            did: 'did:plc:alice',
            handle: 'alice.craftsky.social',
            displayName: 'Alice',
            avatar: null,
            isCraftskyProfile: true,
            viewerIsFollowing: true,
          ),
          AccountSuggestion(
            did: 'did:plc:alicia',
            handle: 'alicia.craftsky.social',
            displayName: 'Alicia',
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
      (ref) => const MockHashtagSuggestionRepository(
        hashtags: [
          HashtagSuggestion(tag: 'SockKAL', postsLast28Days: 128),
          HashtagSuggestion(tag: 'sockmending', postsLast28Days: 12),
        ],
      ),
    );

/// Facet generator backed by the local/mock mention resolver seam.
final facetGeneratorProvider = Provider<FacetGenerator>(
  (ref) => FacetGenerator(
    mentionResolver: ref.watch(accountSuggestionRepositoryProvider),
  ),
);
