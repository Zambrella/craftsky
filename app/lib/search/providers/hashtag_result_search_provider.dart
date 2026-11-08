import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/content_read_guard.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'hashtag_result_search_provider.g.dart';

@riverpod
class HashtagResultSearch extends _$HashtagResultSearch {
  @override
  Future<HashtagSearchResultsState> build(
    HashtagResultSearchQuery query, {
    BrowsingContext? audience,
  }) async {
    final read = await ContentReadGuard.captureWhenReady(ref);
    read.requireCurrent(discarded: const ApiCanceled());
    if (audience != null && read.context != audience) {
      throw const ContentReadDiscarded();
    }

    final page = await ref
        .watch(searchRepositoryProvider)
        .searchHashtags(q: query.q, limit: searchResultsPageLimit);
    read.requireCurrent();
    return HashtagSearchResultsState(items: page.items, cursor: page.cursor);
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ContentReadGuard.capture(ref);
    state = const AsyncLoading<HashtagSearchResultsState>();
    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(searchRepositoryProvider)
          .searchHashtags(
            q: query.q,
            limit: searchResultsPageLimit,
            cursor: current.cursor,
          );
      return HashtagSearchResultsState(
        items: appendUniqueHashtags(current.items, page.items),
        cursor: page.cursor,
      );
    });
    accountOperation.publish(() => state = next);
  }
}
