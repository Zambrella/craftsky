import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/providers/hashtag_search_provider.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:craftsky_app/shared/browsing/content_read_guard.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_search_provider.g.dart';

@riverpod
class PostSearch extends _$PostSearch {
  @override
  Future<SearchPostResultsState> build(
    PostSearchQuery query, {
    BrowsingContext? audience,
  }) async {
    final read = await ContentReadGuard.captureWhenReady(ref);
    read.requireCurrent(discarded: const ApiCanceled());
    if (audience != null && read.context != audience) {
      throw const ContentReadDiscarded();
    }

    listenForPostEdits(ref, () {
      final current = state.value;
      if (current == null) return;
      state = AsyncData(
        current.copyWith(items: applyLoadedPostEdits(ref, current.items)),
      );
    });
    if (!ref.watch(isGuestBrowsingProvider)) {
      ref.watch(activeContentLanguagePolicyProvider);
    }
    final page = await ref
        .watch(searchRepositoryProvider)
        .searchPosts(
          q: query.q,
          limit: searchResultsPageLimit,
        );
    read.requireCurrent();
    return SearchPostResultsState(
      items: applyPostRecordListOverlays(
        ref,
        page.items,
        includes: (_) => false,
      ),
      cursor: page.cursor,
    );
  }

  Future<void> loadMore() async {
    if (!state.hasValue || state.isLoading) return;
    final current = state.requireValue;
    if (!current.hasMore) return;
    final accountOperation = ContentReadGuard.capture(ref);
    state = const AsyncLoading<SearchPostResultsState>();
    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(searchRepositoryProvider)
          .searchPosts(
            q: query.q,
            limit: searchResultsPageLimit,
            cursor: current.cursor,
          );
      accountOperation.requireCurrent();
      return SearchPostResultsState(
        items: appendUniquePosts(
          current.items,
          applyPostRecordListOverlays(ref, page.items, includes: (_) => false),
        ),
        cursor: page.cursor,
      );
    });
    accountOperation.publish(
      () => state = next.whenData(
        (value) =>
            value.copyWith(items: applyLoadedPostEdits(ref, value.items)),
      ),
    );
  }
}
