import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'instagram_suggestions_provider.g.dart';

@immutable
final class InstagramSuggestionReviewState {
  InstagramSuggestionReviewState({
    required List<InstagramSuggestion> items,
    required this.cursor,
    Set<String> busyIds = const {},
    this.hasActionError = false,
  }) : items = List.unmodifiable(items),
       busyIds = Set.unmodifiable(busyIds);

  final List<InstagramSuggestion> items;
  final String? cursor;
  final Set<String> busyIds;
  final bool hasActionError;

  InstagramSuggestionReviewState copyWith({
    List<InstagramSuggestion>? items,
    String? cursor,
    Set<String>? busyIds,
    bool? hasActionError,
  }) => InstagramSuggestionReviewState(
    items: items ?? this.items,
    cursor: cursor ?? this.cursor,
    busyIds: busyIds ?? this.busyIds,
    hasActionError: hasActionError ?? this.hasActionError,
  );

  @override
  String toString() => 'InstagramSuggestionReviewState([REDACTED])';
}

@riverpod
class InstagramSuggestions extends _$InstagramSuggestions {
  @override
  Future<InstagramSuggestionReviewState> build(
    ActiveAccountLease lease,
  ) async {
    final repository = await ref.watch(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    final page = await repository.listSuggestions();
    ensureInstagramOperationCurrent(ref, lease);
    return _fromPage(page);
  }

  Future<void> refresh() async {
    state = const AsyncLoading<InstagramSuggestionReviewState>();
    try {
      final repository = await _repository();
      final page = await repository.listSuggestions();
      ensureInstagramOperationCurrent(ref, lease);
      state = AsyncData(_fromPage(page));
    } on InstagramOperationDiscarded {
      return;
    } on Object catch (error, stackTrace) {
      if (!_isCurrent) return;
      state = AsyncError(error, stackTrace);
    }
  }

  Future<bool> loadMore() async {
    final current = state.value;
    if (current?.cursor == null) return false;
    try {
      final repository = await _repository();
      final page = await repository.listSuggestions(cursor: current!.cursor);
      ensureInstagramOperationCurrent(ref, lease);
      final seen = current.items.map((item) => item.suggestionId).toSet();
      state = AsyncData(
        InstagramSuggestionReviewState(
          items: [
            ...current.items,
            ...page.items.where((item) => seen.add(item.suggestionId)),
          ],
          cursor: page.cursor,
        ),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      return false;
    }
  }

  Future<bool> accept(String suggestionId) => _act(
    suggestionId,
    (repository) async {
      final result = await repository.acceptSuggestion(suggestionId);
      return result.state == InstagramSuggestionState.followed ||
          result.state == InstagramSuggestionState.alreadyFollowing;
    },
  );

  Future<bool> dismiss(String suggestionId) => _act(
    suggestionId,
    (repository) async {
      await repository.dismissSuggestion(suggestionId);
      return true;
    },
  );

  Future<bool> _act(
    String suggestionId,
    Future<bool> Function(InstagramMigrationRepository repository) action,
  ) async {
    final current = state.value;
    if (current == null || current.busyIds.contains(suggestionId)) return false;
    state = AsyncData(
      current.copyWith(
        busyIds: {...current.busyIds, suggestionId},
        hasActionError: false,
      ),
    );
    try {
      final repository = await _repository();
      final completed = await action(repository);
      ensureInstagramOperationCurrent(ref, lease);
      if (!completed) {
        await refresh();
        return _isCurrent;
      }
      final latest = state.value;
      if (latest == null) return false;
      state = AsyncData(
        latest.copyWith(
          items: latest.items
              .where((item) => item.suggestionId != suggestionId)
              .toList(growable: false),
          busyIds: {...latest.busyIds}..remove(suggestionId),
          hasActionError: false,
        ),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      if (!_isCurrent) return false;
      final latest = state.value;
      if (latest != null) {
        state = AsyncData(
          latest.copyWith(
            busyIds: {...latest.busyIds}..remove(suggestionId),
            hasActionError: true,
          ),
        );
      }
      return false;
    }
  }

  Future<InstagramMigrationRepository> _repository() async {
    final repository = await ref.read(
      instagramMigrationRepositoryProvider(lease).future,
    );
    ensureInstagramOperationCurrent(ref, lease);
    return repository;
  }

  bool get _isCurrent {
    if (!ref.mounted) return false;
    try {
      ensureInstagramOperationCurrent(ref, lease);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    }
  }
}

InstagramSuggestionReviewState _fromPage(InstagramSuggestionPage page) =>
    InstagramSuggestionReviewState(items: page.items, cursor: page.cursor);
