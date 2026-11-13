import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'recent_searches_provider.g.dart';

@Riverpod(keepAlive: true)
Future<RecentSearchPage> recentSearchPage(Ref ref) =>
    ref.watch(searchRepositoryProvider).listRecentSearches();

@riverpod
class SaveRecentSearch extends _$SaveRecentSearch {
  @override
  FutureOr<RecentSearchItem?> build() => null;

  Future<RecentSearchItem?> save(SaveRecentSearchRequest request) async {
    final accountOperation = ActiveAccountOperation.capture(ref);
    state = const AsyncLoading();
    final next = await AsyncValue.guard(
      () => ref.read(searchRepositoryProvider).saveRecentSearch(request),
    );
    accountOperation.publish(() {
      state = next;
      if (next.hasValue) {
        ref
          ..invalidate(recentSearchPageProvider)
          ..invalidate(blankSearchProvider);
      }
    });
    return next.value;
  }
}

@riverpod
class DeleteRecentSearch extends _$DeleteRecentSearch {
  @override
  FutureOr<void> build() => null;

  Future<void> delete(String id) async {
    final accountOperation = ActiveAccountOperation.capture(ref);
    state = const AsyncLoading();
    final next = await AsyncValue.guard(
      () => ref.read(searchRepositoryProvider).deleteRecentSearch(id),
    );
    accountOperation.publish(() {
      state = next;
      if (!next.hasError) {
        ref
          ..invalidate(recentSearchPageProvider)
          ..invalidate(blankSearchProvider);
      }
    });
  }
}
