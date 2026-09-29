import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_restore_controller.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-007 restore preserves known assignment and leaves new license open',
    () async {
      final known = _pair('1', SubscriptionTier.plus, assigned: 'did:plc:bob');
      final fresh = _pair('2', SubscriptionTier.business);
      final api = _ScriptedApi([
        _state(7, 7, pairs: [known]),
        _state(7, 7, pairs: [known]),
        _state(8, 8, pairs: [known, fresh]),
      ]);
      final service = _RestoreService();

      final result = await _controller(api, service).restore();

      expect(service.restoreCalls, 1);
      expect(result.outcome, SubscriptionRestoreOutcome.completed);
      expect(result.requiresAssignment, isTrue);
      expect(
        result.assignmentLicense?.id,
        '40000000-0000-4000-8000-000000000002',
      );
      expect(
        result.state?.licenses.first.assignedDid,
        Did.parse('did:plc:bob'),
      );
      expect(result.state?.licenses.last.assignedDid, isNull);
    },
  );

  test('IT-007 inaccessible restored license is not assignable', () async {
    final inaccessible = _pair(
      '2',
      SubscriptionTier.business,
      givesAccess: false,
    );
    final api = _ScriptedApi([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, pairs: [inaccessible]),
    ]);

    final result = await _controller(api, _RestoreService()).restore();

    expect(result.outcome, SubscriptionRestoreOutcome.completed);
    expect(result.assignmentLicense, isNull);
  });

  test('AT-009 unchanged restore completes after reconciliation', () async {
    final api = _ScriptedApi([_state(7, 7), _state(7, 7), _state(8, 8)]);

    final result = await _controller(api, _RestoreService()).restore();

    expect(result.outcome, SubscriptionRestoreOutcome.completed);
    expect(result.requiresAssignment, isFalse);
  });

  test('AT-014 reconciled restore anomaly requires support', () async {
    final anomaly = _pair(
      '1',
      SubscriptionTier.plus,
      anomaly: 'duplicate_tier',
    );
    final api = _ScriptedApi([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, pairs: [anomaly]),
    ]);

    final result = await _controller(api, _RestoreService()).restore();

    expect(result.outcome.name, 'anomaly');
    expect(result.state, isNotNull);
    expect(result.retry, isNull);
  });

  test('IT-007 orphan restored subscription requires support', () async {
    const orphan = BillingSubscription(
      id: '30000000-0000-4000-8000-000000000009',
      productId: 'craftsky_business',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    );
    final api = _ScriptedApi([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, extraSubscriptions: [orphan]),
    ]);

    final result = await _controller(api, _RestoreService()).restore();

    expect(result.outcome, SubscriptionRestoreOutcome.anomaly);
    expect(result.assignmentLicense, isNull);
    expect(result.retry, isNull);
  });

  test('IT-007 wrong RevenueCat identity never restores', () async {
    final service = _RestoreService(
      appUserId: '20000000-0000-4000-8000-000000000099',
    );
    final api = _ScriptedApi([_state(7, 7)]);

    final result = await _controller(api, service).restore();

    expect(result.outcome, SubscriptionRestoreOutcome.providerError);
    expect(service.restoreCalls, 0);
  });

  test('IT-014 AppView UUID mismatch never restores', () async {
    final service = _RestoreService();

    await expectLater(
      _controller(
        _ScriptedApi([
          _state(
            7,
            7,
            revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
          ),
        ]),
        service,
      ).restore(),
      throwsA(isA<BillingOwnerGuardException>()),
    );
    expect(service.restoreCalls, 0);
  });

  test(
    'IT-014 confirmed restore remains pending after route disposal',
    () async {
      var current = true;
      final service = _RestoreService(onRestore: () => current = false);

      final result = await _controller(
        _ScriptedApi([_state(7, 7)]),
        service,
        isCurrent: () => current,
      ).restore();

      expect(result.outcome, SubscriptionRestoreOutcome.pending);
      expect(result.retry, isNotNull);
    },
  );

  test('IT-014 stale pre-restore owner read is cancelled', () async {
    var registry = _registry();
    final api = _ScriptedApi(
      [_state(7, 7)],
      onRead: (read) {
        if (read == 1) registry = _beneficiaryActiveRegistry();
      },
    );
    final service = _RestoreService();

    final result = await _controller(
      api,
      service,
      readRegistry: () => registry,
    ).restore();

    expect(result.outcome, SubscriptionRestoreOutcome.cancelled);
    expect(service.restoreCalls, 0);
  });

  test(
    'IT-014 confirmed restore owner race preserves pending intent',
    () async {
      var registry = _registry();
      final api = _ScriptedApi(
        [_state(7, 7), _state(7, 7)],
        onRead: (read) {
          if (read == 2) registry = _beneficiaryActiveRegistry();
        },
      );

      final result = await _controller(
        api,
        _RestoreService(),
        readRegistry: () => registry,
      ).restore();

      expect(result.outcome, SubscriptionRestoreOutcome.pending);
      expect(result.retry, isNotNull);
      expect(api.reconciliationCalls, 0);
    },
  );

  test(
    'AT-007 post-restore read failure retries reconciliation only',
    () async {
      final api = _ScriptedApi(
        [_state(7, 7), _state(7, 7), _state(8, 8)],
        failReadNumbers: {2},
      );
      final service = _RestoreService();

      final result = await _controller(api, service).restore();
      expect(result.outcome, SubscriptionRestoreOutcome.pending);
      expect(result.retry, isNotNull);

      final retried = await result.retry!();
      expect(retried.outcome, SubscriptionRestoreOutcome.completed);
      expect(api.reconciliationCalls, 1);
      expect(service.restoreCalls, 1);
    },
  );

  test('AT-007 post-restore reconciliation failure stays retryable', () async {
    final api = _ScriptedApi(
      [_state(7, 7), _state(7, 7), _state(7, 7), _state(8, 8)],
      failReconciliationNumbers: {1},
    );
    final service = _RestoreService();

    final result = await _controller(api, service).restore();
    expect(result.outcome, SubscriptionRestoreOutcome.pending);

    final retried = await result.retry!();
    expect(retried.outcome, SubscriptionRestoreOutcome.completed);
    expect(api.reconciliationCalls, 2);
    expect(service.restoreCalls, 1);
  });

  test(
    'IT-010 resumed foreground does not discard a suspended restore poll',
    () async {
      var foregroundChecks = 0;
      final result = await _controller(
        _ScriptedApi([_state(7, 7), _state(7, 7), _state(8, 8)]),
        _RestoreService(),
        isForeground: () {
          foregroundChecks++;
          return foregroundChecks != 5;
        },
      ).restore();

      expect(result.outcome, SubscriptionRestoreOutcome.pending);
      expect(result.retry, isNotNull);
    },
  );

  test(
    'IT-010 pending restore reacquires owner after account switching',
    () async {
      var registry = _beneficiaryActiveRegistry();
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final api = _ScriptedApi(
        [_state(7, 7), _state(7, 7), _state(8, 8)],
        failReadNumbers: {2},
      );
      final service = _RestoreService();
      final controller = _controller(
        api,
        service,
        readRegistry: () => registry,
      );

      final pending = await controller.restore();
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:bob'))!,
      );
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final retried = await pending.retry!();

      expect(retried.outcome, SubscriptionRestoreOutcome.completed);
      expect(api.reconciliationCalls, 1);
      expect(service.restoreCalls, 1);
    },
  );

  test(
    'IT-014 confirmed restore survives owner switch during native call',
    () async {
      var registry = _registry();
      var confirmed = false;
      final api = _ScriptedApi([_state(7, 7)]);
      final result = await _controller(
        api,
        _RestoreService(
          onRestore: () => registry = _beneficiaryActiveRegistry(),
        ),
        readRegistry: () => registry,
        onProviderConfirmed: (_) => confirmed = true,
      ).restore();

      expect(result.outcome, SubscriptionRestoreOutcome.pending);
      expect(result.retry, isNotNull);
      expect(confirmed, isTrue);
      expect(api.reconciliationCalls, 0);
    },
  );
}

SubscriptionRestoreController _controller(
  _ScriptedApi api,
  _RestoreService service, {
  bool Function()? isCurrent,
  bool Function()? isForeground,
  SessionRegistry Function()? readRegistry,
  void Function(BillingState)? onProviderConfirmed,
}) => SubscriptionRestoreController(
  ownerGuard: BillingOwnerGuard(readRegistry ?? _registry),
  api: api,
  revenueCat: service,
  wait: (_) async {},
  isCurrent: isCurrent ?? () => true,
  isForeground: isForeground ?? () => true,
  onProviderConfirmed: onProviderConfirmed,
  maxReconciliationAttempts: 3,
);

SessionRegistry _registry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    )
    .reserveBillingOwner('did:plc:alice')
    .completeBillingOwner(
      'did:plc:alice',
      '20000000-0000-4000-8000-000000000001',
    );

SessionRegistry _beneficiaryActiveRegistry() => _registry().upsertAndActivate(
  token: 'bob-token',
  did: 'did:plc:bob',
  handle: 'bob.test',
);

BillingState _state(
  int requested,
  int reconciled, {
  List<({BillingSubscription subscription, BillingLicense license})> pairs =
      const [],
  List<BillingSubscription> extraSubscriptions = const [],
  String revenueCatAppUserId = '20000000-0000-4000-8000-000000000001',
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: revenueCatAppUserId,
  requestedGeneration: requested,
  reconciledGeneration: reconciled,
  reconciliationStale: requested > reconciled,
  subscriptions: [
    ...pairs.map((value) => value.subscription),
    ...extraSubscriptions,
  ],
  licenses: pairs.map((value) => value.license).toList(),
);

({BillingSubscription subscription, BillingLicense license}) _pair(
  String suffix,
  SubscriptionTier tier, {
  String? assigned,
  String anomaly = 'none',
  bool givesAccess = true,
}) {
  final subscriptionId = '30000000-0000-4000-8000-00000000000$suffix';
  return (
    subscription: BillingSubscription(
      id: subscriptionId,
      productId: 'craftsky_${tier.name}',
      store: 'app_store',
      status: 'active',
      givesAccess: givesAccess,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: anomaly,
    ),
    license: BillingLicense(
      id: '40000000-0000-4000-8000-00000000000$suffix',
      subscriptionId: subscriptionId,
      tier: tier,
      assignedDid: assigned == null ? null : Did.parse(assigned),
      assignable: true,
      anomaly: anomaly,
    ),
  );
}

final class _RestoreService implements RevenueCatService {
  _RestoreService({
    this.appUserId = '20000000-0000-4000-8000-000000000001',
    this.onRestore,
  });

  final String appUserId;
  final void Function()? onRestore;
  int restoreCalls = 0;

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async =>
      RevenueCatIdentity.identified(appUserId);

  @override
  Future<void> restorePurchases() async {
    restoreCalls++;
    onRestore?.call();
  }

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) => throw UnimplementedError();

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) =>
      throw UnimplementedError();

  @override
  Future<void> identify(String appViewUuid) => throw UnimplementedError();

  @override
  Future<DirectPaywallResult> presentPaywall(RevenueCatOffering offering) =>
      throw UnimplementedError();
}

final class _ScriptedApi implements SubscriptionApi {
  _ScriptedApi(
    this.states, {
    this.failReadNumbers = const {},
    this.failReconciliationNumbers = const {},
    this.onRead,
  });

  final List<BillingState> states;
  final Set<int> failReadNumbers;
  final Set<int> failReconciliationNumbers;
  final void Function(int read)? onRead;
  int billingReads = 0;
  int reconciliationCalls = 0;

  @override
  Future<BillingState> getBillingAccount() async {
    billingReads++;
    onRead?.call(billingReads);
    if (failReadNumbers.contains(billingReads)) throw StateError('unavailable');
    return states.removeAt(0);
  }

  @override
  Future<void> requestReconciliation() async {
    reconciliationCalls++;
    if (failReconciliationNumbers.contains(reconciliationCalls)) {
      throw StateError('unavailable');
    }
  }

  @override
  Future<BillingAssignment> assign(String licenseId, String targetDid) =>
      throw UnimplementedError();

  @override
  Future<BillingState> ensureBillingAccount() => throw UnimplementedError();

  @override
  Future<SubscriptionAccess> getAccess() => throw UnimplementedError();

  @override
  Future<void> unassign(String licenseId) => throw UnimplementedError();
}
