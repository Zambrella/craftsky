import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/profile/providers/follow_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
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

  Future<bool> accept(String suggestionId) async {
    final current = state.value;
    if (current == null || current.busyIds.contains(suggestionId)) return false;
    final suggestion = current.items
        .where((item) => item.suggestionId == suggestionId)
        .firstOrNull;
    if (suggestion == null) return false;

    ensureInstagramOperationCurrent(ref, lease);
    final targetDid = Did.parse(suggestion.target.did);
    final endpoint =
        '/v1/migrations/instagram/suggestions/'
        '${Uri.encodeComponent(suggestionId)}/accept';
    const immutableBody = 'POST\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final token = controller.beginOrRetry(
      scope: followProfileMutationScope(ref, targetDid),
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;

    state = AsyncData(
      current.copyWith(
        busyIds: {...current.busyIds, suggestionId},
        hasActionError: false,
      ),
    );
    while (true) {
      late final InstagramSuggestionActionResult result;
      try {
        final repository = await _repository();
        result = await repository.acceptSuggestion(
          suggestionId,
          operationKey: operationKey,
        );
      } on PdsMutationAmbiguousException catch (error) {
        if (!_isCurrent) return false;
        if (!controller.markAmbiguous(
          token,
          retryAfterSeconds: error.retryAfterSeconds,
        )) {
          _clearBusy(suggestionId);
          return false;
        }
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) {
          _completeAction(suggestionId, accepted: false);
          return false;
        }
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!_isCurrent) return false;
        if (!controller.canRetry(
          token,
          operationKey: operationKey,
          endpoint: endpoint,
          immutableBody: immutableBody,
        )) {
          _clearBusy(suggestionId);
          return false;
        }
        continue;
      } on InstagramOperationDiscarded {
        return false;
      } on Object {
        if (!_isCurrent) return false;
        if (!controller.markFailed(token)) {
          _clearBusy(suggestionId);
          return false;
        }
        _completeAction(suggestionId, accepted: false);
        return false;
      }

      if (!_isCurrent) return false;
      final accepted =
          result.state == InstagramSuggestionState.followed ||
          result.state == InstagramSuggestionState.alreadyFollowing;
      if (!accepted) {
        if (!controller.markFailed(token)) {
          _clearBusy(suggestionId);
          return false;
        }
        _completeAction(suggestionId, accepted: false);
        return false;
      }
      const reconciliation = PdsSetReconciliation(active: true);
      if (!controller.markAccepted(
        token,
        optimisticValue: true,
        agrees: (value) => value is bool && reconciliation.agrees(value),
        refresh: () => invalidateFollowProfileReads(ref, targetDid),
      )) {
        _clearBusy(suggestionId);
        return false;
      }
      _completeAction(suggestionId, accepted: true);
      return true;
    }
  }

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

  void _completeAction(String suggestionId, {required bool accepted}) {
    final latest = state.value;
    if (latest == null) return;
    state = AsyncData(
      latest.copyWith(
        items: accepted
            ? latest.items
                  .where((item) => item.suggestionId != suggestionId)
                  .toList(growable: false)
            : latest.items,
        busyIds: {...latest.busyIds}..remove(suggestionId),
        hasActionError: !accepted,
      ),
    );
  }

  void _clearBusy(String suggestionId) {
    final latest = state.value;
    if (latest == null) return;
    state = AsyncData(
      latest.copyWith(
        busyIds: {...latest.busyIds}..remove(suggestionId),
      ),
    );
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
