import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_business_events_provider.g.dart';

@immutable
final class ProfileBusinessEventsTarget {
  const ProfileBusinessEventsTarget({
    required this.account,
    required this.owner,
  });

  final AccountKey account;
  final AtIdentifier owner;

  @override
  bool operator ==(Object other) =>
      other is ProfileBusinessEventsTarget &&
      other.account == account &&
      other.owner == owner;

  @override
  int get hashCode => Object.hash(account, owner);

  @override
  String toString() => 'ProfileBusinessEventsTarget(<redacted>)';
}

@immutable
final class BusinessEventListState {
  const BusinessEventListState({
    required this.items,
    required this.cursor,
    this.isLoadingMore = false,
    this.isRefreshing = false,
    this.incrementalError,
    this.refreshError,
    this.refreshGeneration = 0,
  });

  final List<BusinessEvent> items;
  final String? cursor;
  final bool isLoadingMore;
  final bool isRefreshing;
  final Object? incrementalError;
  final Object? refreshError;
  final int refreshGeneration;

  bool get hasMore => cursor != null;

  BusinessEventListState copyWith({
    List<BusinessEvent>? items,
    Object? cursor = _unset,
    bool? isLoadingMore,
    bool? isRefreshing,
    Object? incrementalError = _unset,
    Object? refreshError = _unset,
    int? refreshGeneration,
  }) => BusinessEventListState(
    items: items ?? this.items,
    cursor: identical(cursor, _unset) ? this.cursor : cursor as String?,
    isLoadingMore: isLoadingMore ?? this.isLoadingMore,
    isRefreshing: isRefreshing ?? this.isRefreshing,
    incrementalError: identical(incrementalError, _unset)
        ? this.incrementalError
        : incrementalError,
    refreshError: identical(refreshError, _unset)
        ? this.refreshError
        : refreshError,
    refreshGeneration: refreshGeneration ?? this.refreshGeneration,
  );
}

const _unset = Object();

@riverpod
class ProfileBusinessEvents extends _$ProfileBusinessEvents {
  var _generation = 0;

  @override
  Future<BusinessEventListState> build(
    ProfileBusinessEventsTarget target,
  ) async {
    _generation = 0;
    final ownership = captureActiveAccountOperation(ref);
    if (ownership != null && ownership.session.account != target.account) {
      throw StateError('Active account changed');
    }
    final lease = _lease(ownership);
    final readFence = ref
        .read(businessProjectionOverlayProvider.notifier)
        .captureRead(lease);
    final page = await ref
        .watch(businessRepositoryProvider)
        .listProfileEvents(target.owner);
    if (!isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('Active account changed');
    }
    if (!_isReadCurrent(readFence)) {
      final retained = _reconcile(
        const [],
        lease,
        ref.read(businessProjectionOverlayProvider.notifier).captureRead(lease),
      );
      return BusinessEventListState(
        items: List.unmodifiable(retained.events),
        cursor: null,
      );
    }
    final reconciliation = _reconcile(page.items, lease, readFence);
    if (reconciliation.isStale) {
      return const BusinessEventListState(items: [], cursor: null);
    }
    return BusinessEventListState(
      items: List.unmodifiable(reconciliation.events),
      cursor: page.cursor,
    );
  }

  Future<void> retryInitial() async {
    if (state.hasValue) return;
    ref.invalidateSelf();
    try {
      await future;
    } on Object {
      // The provider publishes the retry error; callers need not catch it.
    }
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || current.cursor == null || current.isLoadingMore) {
      return;
    }
    final generation = _generation;
    final ownership = captureActiveAccountOperation(ref);
    final lease = _lease(ownership);
    final readFence = ref
        .read(businessProjectionOverlayProvider.notifier)
        .captureRead(lease);
    state = AsyncData(
      current.copyWith(isLoadingMore: true, incrementalError: null),
    );
    try {
      final page = await ref
          .read(businessRepositoryProvider)
          .listProfileEvents(target.owner, cursor: current.cursor);
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (!_isReadCurrent(readFence)) {
        _retainAfterStaleRead(current, lease);
        return;
      }
      final latest = state.value;
      if (latest == null) return;
      final reconciliation = _reconcile(
        [...current.items, ...page.items],
        lease,
        readFence,
      );
      if (reconciliation.isStale) {
        _retainAfterStaleRead(current, lease);
        return;
      }
      state = AsyncData(
        latest.copyWith(
          items: List.unmodifiable(reconciliation.events),
          cursor: page.cursor,
          isLoadingMore: false,
          incrementalError: null,
        ),
      );
    } on Object catch (error) {
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (!_isReadCurrent(readFence)) {
        _retainAfterStaleRead(current, lease);
        return;
      }
      _markFailure(error, lease, readFence);
      state = AsyncData(
        current.copyWith(isLoadingMore: false, incrementalError: error),
      );
    }
  }

  Future<void> refresh() async {
    final current = state.value;
    if (current == null || current.isRefreshing) return;
    final generation = ++_generation;
    final ownership = captureActiveAccountOperation(ref);
    final lease = _lease(ownership);
    final readFence = ref
        .read(businessProjectionOverlayProvider.notifier)
        .captureRead(lease);
    state = AsyncData(
      current.copyWith(
        isRefreshing: true,
        refreshError: null,
        refreshGeneration: generation,
      ),
    );
    try {
      final page = await ref
          .read(businessRepositoryProvider)
          .listProfileEvents(target.owner);
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (!_isReadCurrent(readFence)) {
        _retainAfterStaleRead(current, lease);
        return;
      }
      final reconciliation = _reconcile(page.items, lease, readFence);
      if (reconciliation.isStale) {
        _retainAfterStaleRead(current, lease);
        return;
      }
      state = AsyncData(
        BusinessEventListState(
          items: List.unmodifiable(reconciliation.events),
          cursor: page.cursor,
          refreshGeneration: generation,
        ),
      );
    } on Object catch (error) {
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (!_isReadCurrent(readFence)) {
        _retainAfterStaleRead(current, lease);
        return;
      }
      _markFailure(error, lease, readFence);
      state = AsyncData(
        current.copyWith(isRefreshing: false, refreshError: error),
      );
    }
  }

  BusinessEventListReconciliation _reconcile(
    Iterable<BusinessEvent> events,
    AccountSessionLease lease,
    BusinessProjectionReadFence readFence,
  ) {
    final values = events.toList();
    final owner = _ownerDid(values);
    if (owner == null) return (events: _dedupe(values), isStale: false);
    return reconcileBusinessEventList(
      controller: ref.read(businessProjectionOverlayProvider.notifier),
      lease: lease,
      fence: readFence,
      owner: owner,
      authoritative: values,
      acceptedFilter: (event) =>
          event.status.value == 'scheduled' &&
          event.endsAt.isAfter(DateTime.now().toUtc()),
    );
  }

  void _markFailure(
    Object error,
    AccountSessionLease lease,
    BusinessProjectionReadFence readFence,
  ) {
    final owner = _ownerDid(const []);
    if (owner == null) return;
    markBusinessEventReadFailure(
      controller: ref.read(businessProjectionOverlayProvider.notifier),
      lease: lease,
      fence: readFence,
      owner: owner,
      error: error,
    );
  }

  bool _isReadCurrent(BusinessProjectionReadFence fence) =>
      ref.read(businessProjectionOverlayProvider.notifier).isReadCurrent(fence);

  void _retainAfterStaleRead(
    BusinessEventListState current,
    AccountSessionLease lease,
  ) {
    final fence = ref
        .read(businessProjectionOverlayProvider.notifier)
        .captureRead(lease);
    state = AsyncData(
      current.copyWith(
        items: List.unmodifiable(
          _reconcile(current.items, lease, fence).events,
        ),
        isLoadingMore: false,
        isRefreshing: false,
      ),
    );
  }

  AccountSessionLease _lease(ActiveAccountLease? ownership) =>
      ownership?.session ??
      AccountSessionLease(account: target.account, sessionGeneration: 0);

  Did? _ownerDid(Iterable<BusinessEvent> events) {
    final first = events.firstOrNull;
    if (first != null) return first.did;
    try {
      return Did.parse(target.owner.toString());
    } on Object {
      return null;
    }
  }

  List<BusinessEvent> _dedupe(Iterable<BusinessEvent> events) {
    final identities = <String>{};
    return [
      for (final event in events)
        if (identities.add('${event.did}/${event.rkey}')) event,
    ];
  }
}
