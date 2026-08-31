import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'owner_business_events_provider.g.dart';

@riverpod
class OwnerBusinessEvents extends _$OwnerBusinessEvents {
  var _generation = 0;

  @override
  Future<BusinessEventListState> build(OwnerEventFilter filter) async {
    _generation = 0;
    final ownership = captureActiveAccountOperation(ref);
    final read = _captureRead(ownership, const []);
    final page = await ref
        .watch(businessRepositoryProvider)
        .listOwnerEvents(filter);
    if (!isActiveAccountOperationCurrent(ref, ownership)) {
      throw StateError('Active account changed');
    }
    if (read != null && !_isReadCurrent(read)) {
      return _retainedState(
        const BusinessEventListState(items: [], cursor: null),
        read,
      );
    }
    final result = _pageState(page, generation: _generation, read: read);
    if (result.isStale) {
      return _retainedState(
        const BusinessEventListState(items: [], cursor: null),
        read!,
      );
    }
    return result.state;
  }

  Future<void> retryInitial() async {
    if (state.hasValue) return;
    ref.invalidateSelf();
    try {
      await future;
    } on Object {
      // The provider publishes the retry error.
    }
  }

  Future<void> loadMore() async {
    final current = state.value;
    if (current == null || current.cursor == null || current.isLoadingMore) {
      return;
    }
    final generation = _generation;
    final ownership = captureActiveAccountOperation(ref);
    final read = _captureRead(ownership, current.items);
    state = AsyncData(
      current.copyWith(isLoadingMore: true, incrementalError: null),
    );
    try {
      final page = await ref
          .read(businessRepositoryProvider)
          .listOwnerEvents(filter, cursor: current.cursor);
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (read != null && !_isReadCurrent(read)) {
        state = AsyncData(_retainedState(current, read));
        return;
      }
      final reconciliation = _reconcile(
        [...current.items, ...page.items],
        read,
      );
      if (reconciliation.isStale) {
        state = AsyncData(_retainedState(current, read!));
        return;
      }
      state = AsyncData(
        current.copyWith(
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
      if (read != null && !_isReadCurrent(read)) {
        state = AsyncData(_retainedState(current, read));
        return;
      }
      if (_isInvalidCursor(error)) {
        await _restartAfterInvalidCursor(current);
        return;
      }
      _markFailure(error, read);
      state = AsyncData(
        current.copyWith(isLoadingMore: false, incrementalError: error),
      );
    }
  }

  Future<void> refresh() => _restart(state.value);

  Future<void> _restartAfterInvalidCursor(
    BusinessEventListState current,
  ) => _restart(current);

  Future<void> _restart(BusinessEventListState? current) async {
    if (current == null || current.isRefreshing) return;
    final generation = ++_generation;
    final ownership = captureActiveAccountOperation(ref);
    final read = _captureRead(ownership, current.items);
    state = AsyncData(
      current.copyWith(
        isLoadingMore: false,
        isRefreshing: true,
        incrementalError: null,
        refreshError: null,
        refreshGeneration: generation,
      ),
    );
    try {
      final page = await ref
          .read(businessRepositoryProvider)
          .listOwnerEvents(filter);
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (read != null && !_isReadCurrent(read)) {
        state = AsyncData(_retainedState(current, read));
        return;
      }
      final result = _pageState(page, generation: generation, read: read);
      if (result.isStale) {
        state = AsyncData(_retainedState(current, read!));
        return;
      }
      state = AsyncData(result.state);
    } on Object catch (error) {
      if (generation != _generation ||
          !isActiveAccountOperationCurrent(ref, ownership)) {
        return;
      }
      if (read != null && !_isReadCurrent(read)) {
        state = AsyncData(_retainedState(current, read));
        return;
      }
      _markFailure(error, read);
      state = AsyncData(
        current.copyWith(
          isLoadingMore: false,
          isRefreshing: false,
          incrementalError: null,
          refreshError: error,
          refreshGeneration: generation,
        ),
      );
    }
  }

  ({BusinessEventListState state, bool isStale}) _pageState(
    BusinessEventPage page, {
    required int generation,
    required _OwnerEventRead? read,
  }) {
    final reconciliation = _reconcile(page.items, read);
    return (
      state: BusinessEventListState(
        items: List.unmodifiable(reconciliation.events),
        cursor: page.cursor,
        refreshGeneration: generation,
      ),
      isStale: reconciliation.isStale,
    );
  }

  BusinessEventListReconciliation _reconcile(
    Iterable<BusinessEvent> events,
    _OwnerEventRead? read,
  ) {
    final values = events.toList();
    if (read == null) return (events: _dedupe(values), isStale: false);
    final lease = read.lease;
    final owner = values.firstOrNull?.did ?? lease.account.did;
    return reconcileBusinessEventList(
      controller: ref.read(businessProjectionOverlayProvider.notifier),
      lease: lease,
      fence: read.fence,
      owner: owner,
      authoritative: values,
      acceptedFilter: (event) {
        final upcoming =
            event.status.value == 'scheduled' &&
            event.endsAt.isAfter(DateTime.now().toUtc());
        return filter == OwnerEventFilter.upcoming ? upcoming : !upcoming;
      },
    );
  }

  void _markFailure(Object error, _OwnerEventRead? read) {
    if (read == null) return;
    final lease = read.lease;
    markBusinessEventReadFailure(
      controller: ref.read(businessProjectionOverlayProvider.notifier),
      lease: lease,
      fence: read.fence,
      owner: lease.account.did,
      error: error,
    );
  }

  bool _isReadCurrent(_OwnerEventRead read) => ref
      .read(businessProjectionOverlayProvider.notifier)
      .isReadCurrent(read.fence);

  BusinessEventListState _retainedState(
    BusinessEventListState current,
    _OwnerEventRead staleRead,
  ) {
    final freshRead = (
      lease: staleRead.lease,
      fence: ref
          .read(businessProjectionOverlayProvider.notifier)
          .captureRead(staleRead.lease),
    );
    return current.copyWith(
      items: List.unmodifiable(_reconcile(current.items, freshRead).events),
      isLoadingMore: false,
      isRefreshing: false,
    );
  }

  _OwnerEventRead? _captureRead(
    ActiveAccountLease? ownership,
    List<BusinessEvent> events,
  ) {
    final lease = _lease(ownership, events);
    if (lease == null) return null;
    return (
      lease: lease,
      fence: ref
          .read(businessProjectionOverlayProvider.notifier)
          .captureRead(lease),
    );
  }

  AccountSessionLease? _lease(
    ActiveAccountLease? ownership,
    List<BusinessEvent> events,
  ) {
    if (ownership != null) return ownership.session;
    if (events.isNotEmpty) {
      return AccountSessionLease(
        account: AccountKey(events.first.did.toString()),
        sessionGeneration: 0,
      );
    }
    for (final overlay in ref.read(businessProjectionOverlayProvider).values) {
      if (overlay.lease.sessionGeneration == 0) return overlay.lease;
    }
    return null;
  }
}

typedef _OwnerEventRead = ({
  AccountSessionLease lease,
  BusinessProjectionReadFence fence,
});

bool _isInvalidCursor(Object error) =>
    error is ApiBadRequest && error.code == 'invalid_cursor';

List<BusinessEvent> _dedupe(Iterable<BusinessEvent> events) {
  final identities = <String>{};
  return [
    for (final event in events)
      if (identities.add('${event.did}/${event.rkey}')) event,
  ];
}
