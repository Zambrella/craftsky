import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum BusinessProjectionRecordType { declaration, event }

@immutable
final class BusinessProjectionKey {
  const BusinessProjectionKey({
    required this.account,
    required this.owner,
    required this.recordType,
    required this.rkey,
  });

  factory BusinessProjectionKey.declaration(AccountKey account, Did owner) =>
      BusinessProjectionKey(
        account: account,
        owner: owner,
        recordType: BusinessProjectionRecordType.declaration,
        rkey: RecordKey.parse('self'),
      );

  factory BusinessProjectionKey.event(
    AccountKey account,
    Did owner,
    RecordKey rkey,
  ) => BusinessProjectionKey(
    account: account,
    owner: owner,
    recordType: BusinessProjectionRecordType.event,
    rkey: rkey,
  );

  final AccountKey account;
  final Did owner;
  final BusinessProjectionRecordType recordType;
  final RecordKey rkey;

  @override
  bool operator ==(Object other) =>
      other is BusinessProjectionKey &&
      other.account == account &&
      other.owner == owner &&
      other.recordType == recordType &&
      other.rkey == rkey;

  @override
  int get hashCode => Object.hash(account, owner, recordType, rkey);

  @override
  String toString() => 'BusinessProjectionKey(<redacted>)';
}

@immutable
final class BusinessProjectionRetryMetadata {
  const BusinessProjectionRetryMetadata({
    required this.failureCount,
    required this.error,
  });

  final int failureCount;
  final Object error;
}

@immutable
final class BusinessProjectionReconciliation<T> {
  const BusinessProjectionReconciliation.applied({
    required this.view,
    this.overlay,
  }) : isStale = false;

  const BusinessProjectionReconciliation.stale({
    required this.view,
    this.overlay,
  }) : isStale = true;

  final T? view;
  final BusinessProjectionOverlay<T>? overlay;
  final bool isStale;
}

@immutable
final class BusinessProjectionOverlay<T> {
  const BusinessProjectionOverlay._({
    required this.lease,
    required this.requestGeneration,
    required this.preWriteCid,
    required this.acceptedCid,
    required this.acceptedView,
    required this.isDelete,
    this.retryMetadata,
  });

  const BusinessProjectionOverlay.upsert({
    required AccountSessionLease lease,
    required int requestGeneration,
    required Cid? preWriteCid,
    required Cid acceptedCid,
    required T acceptedView,
  }) : this._(
         lease: lease,
         requestGeneration: requestGeneration,
         preWriteCid: preWriteCid,
         acceptedCid: acceptedCid,
         acceptedView: acceptedView,
         isDelete: false,
       );

  const BusinessProjectionOverlay.delete({
    required AccountSessionLease lease,
    required int requestGeneration,
    required Cid deletedCid,
    required Cid acceptedCid,
  }) : this._(
         lease: lease,
         requestGeneration: requestGeneration,
         preWriteCid: deletedCid,
         acceptedCid: acceptedCid,
         acceptedView: null,
         isDelete: true,
       );

  final AccountSessionLease lease;
  final int requestGeneration;
  final Cid? preWriteCid;
  final Cid acceptedCid;
  final T? acceptedView;
  final bool isDelete;
  final BusinessProjectionRetryMetadata? retryMetadata;

  BusinessProjectionReconciliation<T> reconcile({
    required Cid? authoritativeCid,
    required T? authoritativeView,
  }) {
    if (isDelete) {
      if (authoritativeCid == preWriteCid) {
        return BusinessProjectionReconciliation.applied(
          view: null,
          overlay: this,
        );
      }
      return BusinessProjectionReconciliation.applied(
        view: authoritativeView,
      );
    }
    if (authoritativeCid == preWriteCid) {
      return BusinessProjectionReconciliation.applied(
        view: acceptedView,
        overlay: this,
      );
    }
    return BusinessProjectionReconciliation.applied(view: authoritativeView);
  }

  BusinessProjectionOverlay<T> withFailure(Object error) =>
      BusinessProjectionOverlay._(
        lease: lease,
        requestGeneration: requestGeneration,
        preWriteCid: preWriteCid,
        acceptedCid: acceptedCid,
        acceptedView: acceptedView,
        isDelete: isDelete,
        retryMetadata: BusinessProjectionRetryMetadata(
          failureCount: (retryMetadata?.failureCount ?? 0) + 1,
          error: error,
        ),
      );

  BusinessProjectionOverlay<R> cast<R>() => BusinessProjectionOverlay<R>._(
    lease: lease,
    requestGeneration: requestGeneration,
    preWriteCid: preWriteCid,
    acceptedCid: acceptedCid,
    acceptedView: acceptedView as R?,
    isDelete: isDelete,
    retryMetadata: retryMetadata,
  );
}

@immutable
final class BusinessProjectionReadFence {
  const BusinessProjectionReadFence({
    required this.lease,
    required this.accountGeneration,
    required this.recordGenerations,
  });

  final AccountSessionLease lease;
  final int accountGeneration;
  final Map<BusinessProjectionKey, int> recordGenerations;
}

enum BusinessProjectionReloadDecision {
  nothingToDiscard,
  warningRequired,
  discarded,
}

final businessProjectionOverlayProvider =
    NotifierProvider<
      BusinessProjectionOverlayController,
      Map<BusinessProjectionKey, BusinessProjectionOverlay<Object>>
    >(BusinessProjectionOverlayController.new);

final class BusinessProjectionOverlayController
    extends
        Notifier<
          Map<BusinessProjectionKey, BusinessProjectionOverlay<Object>>
        > {
  final _requestGenerations = <BusinessProjectionKey, int>{};
  final _requestLeases = <BusinessProjectionKey, AccountSessionLease>{};
  final _accountGenerations = <AccountKey, int>{};

  @override
  Map<BusinessProjectionKey, BusinessProjectionOverlay<Object>> build() =>
      const {};

  void advanceAccountBoundary() {
    final accounts = <AccountKey>{
      ..._accountGenerations.keys,
      ..._requestGenerations.keys.map((key) => key.account),
      ...state.keys.map((key) => key.account),
    };
    _requestGenerations.updateAll((_, generation) => generation + 1);
    _requestLeases.clear();
    accounts.forEach(_advanceAccount);
    state = const {};
  }

  int beginMutation(BusinessProjectionKey key, AccountSessionLease lease) {
    if (key.account != lease.account) return -1;
    final generation = (_requestGenerations[key] ?? 0) + 1;
    _requestGenerations[key] = generation;
    _requestLeases[key] = lease;
    _advanceAccount(key.account);
    return generation;
  }

  bool acceptUpsert<T>({
    required BusinessProjectionKey key,
    required AccountSessionLease lease,
    required int requestGeneration,
    required Cid? preWriteCid,
    required Cid acceptedCid,
    required T acceptedView,
  }) {
    if (!_isCurrent(key, lease, requestGeneration)) return false;
    state = {
      ...state,
      key: BusinessProjectionOverlay<Object>.upsert(
        lease: lease,
        requestGeneration: requestGeneration,
        preWriteCid: preWriteCid,
        acceptedCid: acceptedCid,
        acceptedView: acceptedView as Object,
      ),
    };
    return true;
  }

  bool acceptDelete({
    required BusinessProjectionKey key,
    required AccountSessionLease lease,
    required int requestGeneration,
    required Cid deletedCid,
    required Cid acceptedCid,
  }) {
    if (!_isCurrent(key, lease, requestGeneration)) return false;
    state = {
      ...state,
      key: BusinessProjectionOverlay<Object>.delete(
        lease: lease,
        requestGeneration: requestGeneration,
        deletedCid: deletedCid,
        acceptedCid: acceptedCid,
      ),
    };
    return true;
  }

  BusinessProjectionReadFence captureRead(AccountSessionLease lease) =>
      BusinessProjectionReadFence(
        lease: lease,
        accountGeneration: _accountGenerations[lease.account] ?? 0,
        recordGenerations: Map.unmodifiable({
          for (final entry in _requestGenerations.entries)
            if (entry.key.account == lease.account) entry.key: entry.value,
        }),
      );

  bool isReadCurrent(BusinessProjectionReadFence fence) =>
      (_accountGenerations[fence.lease.account] ?? 0) ==
      fence.accountGeneration;

  bool hasOverlay(BusinessProjectionKey key, AccountSessionLease lease) =>
      state[key]?.lease == lease;

  bool isRecordReadCurrent(
    BusinessProjectionKey key,
    BusinessProjectionReadFence fence,
  ) =>
      isReadCurrent(fence) &&
      (fence.recordGenerations[key] ?? 0) == (_requestGenerations[key] ?? 0);

  BusinessProjectionReconciliation<T> reconcile<T>({
    required BusinessProjectionKey key,
    required BusinessProjectionReadFence fence,
    required Cid? authoritativeCid,
    required T? authoritativeView,
  }) {
    if (key.account != fence.lease.account ||
        !isRecordReadCurrent(key, fence)) {
      final current = state[key];
      final retained = current?.lease == fence.lease
          ? current?.cast<T>()
          : null;
      return BusinessProjectionReconciliation.stale(
        view: retained?.acceptedView,
        overlay: retained,
      );
    }
    final untyped = state[key];
    if (untyped == null) {
      return BusinessProjectionReconciliation.applied(
        view: authoritativeView,
      );
    }
    if (untyped.lease != fence.lease) {
      return const BusinessProjectionReconciliation.stale(view: null);
    }
    final overlay = untyped.cast<T>();
    final result = overlay.reconcile(
      authoritativeCid: authoritativeCid,
      authoritativeView: authoritativeView,
    );
    if (result.overlay == null) {
      final next = {...state}..remove(key);
      state = next;
      _requestGenerations[key] = (_requestGenerations[key] ?? 0) + 1;
    }
    return result;
  }

  bool markReadFailure({
    required BusinessProjectionKey key,
    required BusinessProjectionReadFence fence,
    required Object error,
  }) {
    if (key.account != fence.lease.account ||
        !isRecordReadCurrent(key, fence)) {
      return false;
    }
    final overlay = state[key];
    if (overlay == null || overlay.lease != fence.lease) return false;
    state = {...state, key: overlay.withFailure(error)};
    return true;
  }

  BusinessProjectionReloadDecision discardForExplicitReload({
    required BusinessProjectionKey key,
    required AccountSessionLease lease,
    required bool confirmed,
  }) {
    final overlay = state[key];
    if (overlay == null || overlay.lease != lease) {
      return BusinessProjectionReloadDecision.nothingToDiscard;
    }
    if (!confirmed) return BusinessProjectionReloadDecision.warningRequired;
    state = {...state}..remove(key);
    _requestGenerations[key] = (_requestGenerations[key] ?? 0) + 1;
    _advanceAccount(key.account);
    return BusinessProjectionReloadDecision.discarded;
  }

  Iterable<MapEntry<BusinessProjectionKey, BusinessProjectionOverlay<T>>>
  overlaysFor<T>({
    required AccountSessionLease lease,
    required BusinessProjectionRecordType recordType,
    Did? owner,
  }) sync* {
    for (final entry in state.entries) {
      if (entry.key.account == lease.account &&
          entry.key.recordType == recordType &&
          (owner == null || entry.key.owner == owner) &&
          entry.value.lease == lease) {
        yield MapEntry(entry.key, entry.value.cast<T>());
      }
    }
  }

  bool _isCurrent(
    BusinessProjectionKey key,
    AccountSessionLease lease,
    int requestGeneration,
  ) =>
      key.account == lease.account &&
      requestGeneration >= 0 &&
      _requestLeases[key] == lease &&
      _requestGenerations[key] == requestGeneration;

  void _advanceAccount(AccountKey account) {
    _accountGenerations[account] = (_accountGenerations[account] ?? 0) + 1;
  }
}

typedef BusinessEventListReconciliation = ({
  List<BusinessEvent> events,
  bool isStale,
});

BusinessEventListReconciliation reconcileBusinessEventList({
  required BusinessProjectionOverlayController controller,
  required AccountSessionLease lease,
  required BusinessProjectionReadFence fence,
  required Did owner,
  required Iterable<BusinessEvent> authoritative,
  bool Function(BusinessEvent event)? acceptedFilter,
}) {
  final seen = <BusinessProjectionKey>{};
  final events = <BusinessEvent>[];
  var isStale = false;
  for (final event in authoritative) {
    final key = BusinessProjectionKey.event(
      lease.account,
      event.did,
      event.rkey,
    );
    final hadOverlay = controller.hasOverlay(key, lease);
    final reconciliation = controller.reconcile<BusinessEvent>(
      key: key,
      fence: fence,
      authoritativeCid: event.cid,
      authoritativeView: event,
    );
    final resolved = reconciliation.view;
    isStale = isStale || reconciliation.isStale;
    if (!reconciliation.isStale || reconciliation.overlay != null) {
      seen.add(key);
    }
    if (resolved != null &&
        (!hadOverlay || acceptedFilter == null || acceptedFilter(resolved))) {
      events.add(resolved);
    }
  }
  for (final entry in controller.overlaysFor<BusinessEvent>(
    lease: lease,
    recordType: BusinessProjectionRecordType.event,
    owner: owner,
  )) {
    final event = entry.value.acceptedView;
    if (seen.add(entry.key) &&
        !entry.value.isDelete &&
        event != null &&
        (acceptedFilter == null || acceptedFilter(event))) {
      events.add(event);
    }
  }
  final identities = <String>{};
  return (
    events: [
      for (final event in events)
        if (identities.add('${event.did}/${event.rkey}')) event,
    ],
    isStale: isStale,
  );
}

void markBusinessEventReadFailure({
  required BusinessProjectionOverlayController controller,
  required AccountSessionLease lease,
  required BusinessProjectionReadFence fence,
  required Did owner,
  required Object error,
}) {
  final keys = [
    for (final entry in controller.overlaysFor<BusinessEvent>(
      lease: lease,
      recordType: BusinessProjectionRecordType.event,
      owner: owner,
    ))
      entry.key,
  ];
  for (final key in keys) {
    controller.markReadFailure(key: key, fence: fence, error: error);
  }
}
