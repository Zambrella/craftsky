import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/rich_text/data/appview_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Default debounce duration for mention/hashtag autocomplete.
final facetAutocompleteDebounceProvider = Provider<Duration>(
  (ref) => const Duration(milliseconds: 300),
);

/// AppView-backed account suggestions.
final accountSuggestionRepositoryProvider =
    Provider<AccountSuggestionRepository>(
      (ref) => AppViewAccountSuggestionRepository(ref.watch(dioProvider)),
    );

/// AppView-backed hashtag suggestions.
final hashtagSuggestionRepositoryProvider =
    Provider<HashtagSuggestionRepository>(
      (ref) => AppViewHashtagSuggestionRepository(ref.watch(dioProvider)),
    );

/// Facet generator backed by the local/mock mention resolver seam.
final facetGeneratorProvider = Provider<FacetGenerator>(
  (ref) => FacetGenerator(
    mentionResolver: ref.watch(accountSuggestionRepositoryProvider),
  ),
);
