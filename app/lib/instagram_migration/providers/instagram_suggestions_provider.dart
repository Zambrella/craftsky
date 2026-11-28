import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
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
  new({
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
  final Set<Did> _followedElsewhere = {};
  int _refreshSequence = 0;
  int _actionRevision = 0;

  @override
  Future<InstagramSuggestionReviewState> build(
    ActiveAccountLease lease,
  ) async {
    // Ordinary follow buttons publish through this shared mutation provider.
    // Remove accepted follows immediately, while the projection catches up.
    ref.listen(toggleFollowProfileProvider, (previous, next) {
      if (_isCurrent &&
          next.hasValue &&
          next.value != null &&
          next != previous) {
        final profile = next.requireValue!;
        if (profile.viewerIsFollowing) {
          _followedElsewhere.add(profile.did);
          final current = state.value;
          if (current != null) {
            state = AsyncData(
              current.copyWith(
                items: current.items
                    .where(
                      (item) => item.target.did != profile.did.toString(),
                    )
                    .toList(growable: false),
              ),
            );
          }
        } else {
          _followedElsewhere.remove(profile.did);
          ref.invalidateSelf();
        }
      }
    });
    final repository = await ref.watch(
      instagramMigrationRepositoryProvider(lease).future,
    );
    _operation.requireCurrent(discarded: const InstagramOperationDiscarded());
    final page = await _operation.step(
      repository.listSuggestions,
      discarded: const InstagramOperationDiscarded(),
    );
    _operation.requireCurrent(discarded: const InstagramOperationDiscarded());
    return _fromPage(page);
  }

  Future<void> refresh() async {
    final sequence = ++_refreshSequence;
    final revision = _actionRevision;
    if (!state.hasValue) state = const AsyncLoading();
    bool isLatest() =>
        _isCurrent &&
        sequence == _refreshSequence &&
        revision == _actionRevision;
    try {
      final repository = await _repository();
      final page = await _operation.step(
        repository.listSuggestions,
        discarded: const InstagramOperationDiscarded(),
      );
      if (!isLatest()) return;
      state = AsyncData(
        _fromPage(page).copyWith(busyIds: state.value?.busyIds),
      );
    } on InstagramOperationDiscarded {
      return;
    } on Object catch (error, stackTrace) {
      if (!isLatest()) return;
      state = AsyncError(error, stackTrace);
    }
  }

  Future<bool> loadMore() async {
    final cursor = state.value?.cursor;
    if (cursor == null) return false;
    final sequence = _refreshSequence;
    try {
      final repository = await _repository();
      final page = await _operation.step(
        () => repository.listSuggestions(cursor: cursor),
        discarded: const InstagramOperationDiscarded(),
      );
      _operation.requireCurrent(discarded: const InstagramOperationDiscarded());
      final latest = state.value;
      if (latest == null ||
          latest.cursor != cursor ||
          sequence != _refreshSequence) {
        return false;
      }
      final seen = latest.items.map((item) => item.suggestionId).toSet();
      state = AsyncData(
        InstagramSuggestionReviewState(
          items: [
            ...latest.items,
            ..._fromPage(page).items
                .where((item) => seen.add(item.suggestionId)),
          ],
          cursor: page.cursor,
          busyIds: latest.busyIds,
          hasActionError: latest.hasActionError,
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
    if (!_isCurrent ||
        current == null ||
        current.busyIds.contains(suggestionId)) {
      return false;
    }
    final suggestion = current.items
        .where((item) => item.suggestionId == suggestionId)
        .firstOrNull;
    if (suggestion == null) return false;

    final targetDid = Did.parse(suggestion.target.did);
    final endpoint =
        '/v1/migrations/instagram/suggestions/'
        '${Uri.encodeComponent(suggestionId)}/accept';
    const immutableBody = 'POST\n';
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final scope = followProfileMutationScope(ref, targetDid);
    final pending = controller.retryToken(
      scope: scope,
      endpoint: endpoint,
      immutableBody: immutableBody,
    );
    if (pending == null &&
        controller.operationFor(scope)?.status == PdsMutationStatus.ambiguous) {
      // A follow started elsewhere must be retried at its original endpoint.
      return false;
    }
    final token =
        pending ??
        controller.begin(
          scope: scope,
          endpoint: endpoint,
          immutableBody: immutableBody,
          operationKey: newPdsMutationOperationKey(),
        );
    final operationKey = token.operationKey;
    final retry = PdsMutationRetry(
      ref: ref,
      controller: controller,
      token: token,
      isCurrent: () => _isCurrent,
    );
    late final repositoryFuture = _repository();

    state = AsyncData(
      current.copyWith(
        busyIds: {...current.busyIds, suggestionId},
        hasActionError: false,
      ),
    );
    while (true) {
      late final InstagramSuggestionActionResult result;
      try {
        final repository = await repositoryFuture;
        _operation.requireCurrent(
          discarded: const InstagramOperationDiscarded(),
        );
        result = await repository.acceptSuggestion(
          suggestionId,
          operationKey: operationKey,
        );
      } on PdsMutationAmbiguousException catch (error) {
        final decision = await retry.wait(error);
        if (!_isCurrent) return false;
        switch (decision) {
          case PdsMutationRetryDecision.obsolete:
            if (_isCurrent) _clearBusy(suggestionId);
            return false;
          case PdsMutationRetryDecision.exhausted:
            _completeAction(suggestionId, accepted: false);
            return false;
          case PdsMutationRetryDecision.retry:
            continue;
        }
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

  Future<bool> dismiss(String suggestionId) async {
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
      await _operation.step(
        () => repository.dismissSuggestion(suggestionId),
        discarded: const InstagramOperationDiscarded(),
      );
      _operation.requireCurrent(discarded: const InstagramOperationDiscarded());
      _completeAction(suggestionId, accepted: true);
      return true;
    } on InstagramOperationDiscarded {
      return false;
    } on Object {
      if (!_isCurrent) return false;
      _completeAction(suggestionId, accepted: false);
      return false;
    }
  }

  Future<InstagramMigrationRepository> _repository() => _operation.step(
    () => ref.read(instagramMigrationRepositoryProvider(lease).future),
    discarded: const InstagramOperationDiscarded(),
  );

  void _completeAction(String suggestionId, {required bool accepted}) {
    final latest = state.value;
    if (latest == null) return;
    _actionRevision++;
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

  ActiveAccountOperation get _operation =>
      ActiveAccountOperation.fromLease(ref, lease);

  bool get _isCurrent => _operation.isCurrent;

  InstagramSuggestionReviewState _fromPage(InstagramSuggestionPage page) =>
      InstagramSuggestionReviewState(
        items: page.items
            .where((item) {
              final did = Did.parse(item.target.did);
              final overlay = ref
                  .read(pdsRecordOperationControllerProvider)
                  .overlayFor(followProfileMutationScope(ref, did));
              return overlay?.optimisticValue != true &&
                  !_followedElsewhere.contains(did);
            })
            .toList(growable: false),
        cursor: page.cursor,
      );
}
