import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_interaction_list_state.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_interaction_lists_provider.g.dart';

const postInteractionListPageLimit = 20;

enum PostInteractionAccountKind { likes, reposts }

@riverpod
class PostInteractionAccounts extends _$PostInteractionAccounts {
  var _generation = 0;
  var _continuationInFlight = false;

  @override
  Future<PostInteractionAccountsState> build(
    Did did,
    RecordKey rkey,
    PostInteractionAccountKind kind,
  ) async {
    final generation = ++_generation;
    final ownership = (await ref.watch(
      sessionRegistryProvider.future,
    )).activeLease;
    final page = await _list();
    if (generation != _generation ||
        !isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('Active account changed');
    }
    return _accountState(page);
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || _continuationInFlight) return;

    _continuationInFlight = true;
    final generation = _generation;
    final ownership = captureActiveAccountOperation(ref);
    final cursor = current.cursor!;
    state = const AsyncLoading<PostInteractionAccountsState>();
    try {
      final page = await _list(cursor: cursor);
      if (!_isCurrent(generation, ownership)) return;
      state = AsyncData(
        PostInteractionAccountsState(
          items: _appendAccounts(current.items, page.items),
          cursor: page.cursor,
          totalCount: page.totalCount,
        ),
      );
    } on Object catch (error, stackTrace) {
      if (!_isCurrent(generation, ownership)) return;
      if (_isInvalidCursor(error)) {
        await _restart(current);
      } else {
        state = AsyncError<PostInteractionAccountsState>(error, stackTrace);
      }
    } finally {
      _continuationInFlight = false;
    }
  }

  Future<void> refresh() async {
    final current = state.value;
    if (current == null) {
      ref.invalidateSelf();
      try {
        await future;
      } on Object {
        // The provider publishes the retry error.
      }
      return;
    }
    await _restart(current);
  }

  Future<void> _restart(PostInteractionAccountsState current) async {
    final generation = ++_generation;
    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading<PostInteractionAccountsState>();
    try {
      final page = await _list();
      if (!_isCurrent(generation, ownership)) return;
      state = AsyncData(_accountState(page));
    } on Object catch (error, stackTrace) {
      if (!_isCurrent(generation, ownership)) return;
      state = AsyncError<PostInteractionAccountsState>(error, stackTrace);
    }
  }

  Future<ProfileAccountPage> _list({String? cursor}) => switch (kind) {
    PostInteractionAccountKind.likes =>
      ref
          .read(postRepositoryProvider)
          .listLikes(
            did,
            rkey,
            cursor: cursor,
            limit: postInteractionListPageLimit,
          ),
    PostInteractionAccountKind.reposts =>
      ref
          .read(postRepositoryProvider)
          .listReposts(
            did,
            rkey,
            cursor: cursor,
            limit: postInteractionListPageLimit,
          ),
  };

  bool _isCurrent(int generation, ActiveAccountLease? ownership) =>
      generation == _generation &&
      isActiveAccountOperationCurrent(ref, ownership);
}

@riverpod
class PostQuotes extends _$PostQuotes {
  var _generation = 0;
  var _continuationInFlight = false;

  @override
  Future<PostQuotesState> build(Did did, RecordKey rkey) async {
    ref.watch(activeContentLanguagePolicyProvider);
    final generation = ++_generation;
    final ownership = (await ref.watch(
      sessionRegistryProvider.future,
    )).activeLease;
    final page = await _list();
    if (generation != _generation ||
        !isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('Active account changed');
    }
    return _quoteState(page);
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || !current.hasMore || _continuationInFlight) return;

    _continuationInFlight = true;
    final generation = _generation;
    final ownership = captureActiveAccountOperation(ref);
    final cursor = current.cursor!;
    state = const AsyncLoading<PostQuotesState>();
    try {
      final page = await _list(cursor: cursor);
      if (!_isCurrent(generation, ownership)) return;
      final latest = state.value ?? current;
      state = AsyncData(
        PostQuotesState(
          items: _appendQuotes(latest.items, page.items),
          cursor: page.cursor,
        ),
      );
    } on Object catch (error, stackTrace) {
      if (!_isCurrent(generation, ownership)) return;
      if (_isInvalidCursor(error)) {
        await _restart(current);
      } else {
        state = AsyncError<PostQuotesState>(error, stackTrace);
      }
    } finally {
      _continuationInFlight = false;
    }
  }

  Future<void> refresh() async {
    final current = state.value;
    if (current == null) {
      ref.invalidateSelf();
      try {
        await future;
      } on Object {
        // The provider publishes the retry error.
      }
      return;
    }
    await _restart(current);
  }

  void replace(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      PostQuotesState(
        items: [
          for (final item in current.items)
            if (item.uri == post.uri) post else item,
        ],
        cursor: current.cursor,
      ),
    );
  }

  void remove(AtUri uri) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      PostQuotesState(
        items: current.items.where((post) => post.uri != uri).toList(),
        cursor: current.cursor,
      ),
    );
  }

  Future<void> _restart(PostQuotesState current) async {
    final generation = ++_generation;
    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading<PostQuotesState>();
    try {
      final page = await _list();
      if (!_isCurrent(generation, ownership)) return;
      state = AsyncData(_quoteState(page));
    } on Object catch (error, stackTrace) {
      if (!_isCurrent(generation, ownership)) return;
      state = AsyncError<PostQuotesState>(error, stackTrace);
    }
  }

  Future<PostPage> _list({String? cursor}) => ref
      .read(postRepositoryProvider)
      .listQuotes(
        did,
        rkey,
        cursor: cursor,
        limit: postInteractionListPageLimit,
      );

  bool _isCurrent(int generation, ActiveAccountLease? ownership) =>
      generation == _generation &&
      isActiveAccountOperationCurrent(ref, ownership);
}

PostInteractionAccountsState _accountState(ProfileAccountPage page) =>
    PostInteractionAccountsState(
      items: _dedupeAccounts(page.items),
      cursor: page.cursor,
      totalCount: page.totalCount,
    );

PostQuotesState _quoteState(PostPage page) => PostQuotesState(
  items: _dedupeQuotes(page.items),
  cursor: page.cursor,
);

List<ProfileAccountSummary> _dedupeAccounts(
  Iterable<ProfileAccountSummary> items,
) {
  final seen = <Did>{};
  return [
    for (final item in items)
      if (seen.add(item.did)) item,
  ];
}

List<ProfileAccountSummary> _appendAccounts(
  List<ProfileAccountSummary> current,
  List<ProfileAccountSummary> next,
) {
  final seen = current.map((item) => item.did).toSet();
  return [
    ...current,
    for (final item in next)
      if (seen.add(item.did)) item,
  ];
}

List<Post> _dedupeQuotes(Iterable<Post> items) {
  final seen = <AtUri>{};
  return [
    for (final item in items)
      if (seen.add(item.uri)) item,
  ];
}

List<Post> _appendQuotes(List<Post> current, List<Post> next) {
  final seen = current.map((post) => post.uri).toSet();
  return [
    ...current,
    for (final post in next)
      if (seen.add(post.uri)) post,
  ];
}

bool _isInvalidCursor(Object error) =>
    error is ApiBadRequest && error.code == 'invalid_cursor';
