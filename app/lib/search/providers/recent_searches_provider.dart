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

  Future<RecentSearchItem> save(SaveRecentSearchRequest request) async {
    final accountOperation = ActiveAccountOperation.capture(ref);
    state = const AsyncLoading();
    final saved = await ref
        .read(searchRepositoryProvider)
        .saveRecentSearch(request);
    if (!accountOperation.isCurrent) return saved;
    ref
      ..invalidate(recentSearchPageProvider)
      ..invalidate(blankSearchProvider);
    state = AsyncData(saved);
    return saved;
  }
}

@riverpod
class DeleteRecentSearch extends _$DeleteRecentSearch {
  @override
  FutureOr<void> build() => null;

  Future<void> delete(String id) async {
    final accountOperation = ActiveAccountOperation.capture(ref);
    state = const AsyncLoading();
    await ref.read(searchRepositoryProvider).deleteRecentSearch(id);
    if (!accountOperation.isCurrent) return;
    ref
      ..invalidate(recentSearchPageProvider)
      ..invalidate(blankSearchProvider);
    state = const AsyncData(null);
  }
}
