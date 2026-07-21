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
    Set<String> selectedIds = const {},
    Set<String> busyIds = const {},
    this.hasActionError = false,
  }) : items = List.unmodifiable(items),
       selectedIds = Set.unmodifiable(selectedIds),
       busyIds = Set.unmodifiable(busyIds);

  final List<InstagramSuggestion> items;
  final String? cursor;
  final Set<String> selectedIds;
  final Set<String> busyIds;
  final bool hasActionError;

  InstagramSuggestionReviewState copyWith({
    List<InstagramSuggestion>? items,
    String? cursor,
    Set<String>? selectedIds,
    Set<String>? busyIds,
    bool? hasActionError,
  }) => InstagramSuggestionReviewState(
    items: items ?? this.items,
    cursor: cursor ?? this.cursor,
    selectedIds: selectedIds ?? this.selectedIds,
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

  void select(String suggestionId, {required bool selected}) {
    final current = state.value;
    if (current == null ||
        !current.items.any(
          (item) =>
              item.suggestionId == suggestionId &&
              item.state == InstagramSuggestionState.pending,
        )) {
      return;
    }
    final next = {...current.selectedIds};
    if (selected) {
      next.add(suggestionId);
    } else {
      next.remove(suggestionId);
    }
    state = AsyncData(current.copyWith(selectedIds: next));
  }

  void selectAllReviewed() {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        selectedIds: {
          for (final item in current.items)
            if (item.state == InstagramSuggestionState.pending)
              item.suggestionId,
        },
      ),
    );
  }

  void clearSelection() {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(current.copyWith(selectedIds: const {}));
  }

  Future<bool> dismiss(String suggestionId) async {
    final current = state.value;
    if (current == null || current.busyIds.contains(suggestionId)) return false;
    _setBusy(current, suggestionId);
    try {
      final repository = await _repository();
      await repository.dismissSuggestion(suggestionId);
      ensureInstagramOperationCurrent(ref, lease);
      final now = state.value;
      if (now == null) return false;
      state = AsyncData(
        now.copyWith(
          items: now.items
              .where((item) => item.suggestionId != suggestionId)
              .toList(growable: false),
          selectedIds: {...now.selectedIds}..remove(suggestionId),
          busyIds: {...now.busyIds}..remove(suggestionId),
          hasActionError: false,
        ),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      _setActionError(suggestionId);
      return false;
    }
  }

  Future<bool> accept(String suggestionId) async {
    final current = state.value;
    if (current == null || current.busyIds.contains(suggestionId)) return false;
    _setBusy(current, suggestionId);
    try {
      final repository = await _repository();
      final result = await repository.acceptSuggestion(suggestionId);
      ensureInstagramOperationCurrent(ref, lease);
      if (_isCompletedAction(result.state)) {
        final now = state.value;
        if (now == null) return false;
        state = AsyncData(
          now.copyWith(
            items: now.items
                .where((item) => item.suggestionId != suggestionId)
                .toList(growable: false),
            selectedIds: {...now.selectedIds}..remove(suggestionId),
            busyIds: {...now.busyIds}..remove(suggestionId),
            hasActionError: false,
          ),
        );
        return true;
      }
      final page = await repository.listSuggestions();
      ensureInstagramOperationCurrent(ref, lease);
      state = AsyncData(_fromPage(page));
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      _setActionError(suggestionId);
      return false;
    }
  }

  Future<int> acceptSelected() async {
    final ids = state.value?.selectedIds.toList(growable: false) ?? const [];
    var accepted = 0;
    try {
      for (final id in ids) {
        if (await accept(id)) accepted++;
        ensureInstagramOperationCurrent(ref, lease);
      }
    } on InstagramOperationDiscarded {
      return accepted;
    }
    return accepted;
  }

  Future<void> refresh() async {
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
          selectedIds: current.selectedIds,
          busyIds: current.busyIds,
        ),
      );
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      return false;
    }
  }

  void _setBusy(
    InstagramSuggestionReviewState current,
    String suggestionId,
  ) {
    state = AsyncData(
      current.copyWith(
        busyIds: {...current.busyIds, suggestionId},
        hasActionError: false,
      ),
    );
  }

  void _setActionError(String suggestionId) {
    if (!_isCurrent) return;
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      current.copyWith(
        busyIds: {...current.busyIds}..remove(suggestionId),
        hasActionError: true,
      ),
    );
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

bool _isCompletedAction(InstagramSuggestionState state) => switch (state) {
  InstagramSuggestionState.accepted ||
  InstagramSuggestionState.alreadyFollowing ||
  InstagramSuggestionState.dismissed ||
  InstagramSuggestionState.invalidated => true,
  _ => false,
};
