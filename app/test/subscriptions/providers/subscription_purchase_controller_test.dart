import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_purchase_controller.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-006 new purchase completes unassigned and requests assignment',
    () async {
      final result = await _runPurchase([
        _state(7, 7),
        _state(7, 7),
        _state(8, 8, licenses: [_pair('new')]),
      ]);

      expect(result.outcome, SubscriptionPurchaseOutcome.completed);
      expect(result.requiresAssignment, isTrue);
      expect(
        result.assignmentLicense?.id,
        '40000000-0000-4000-8000-000000000002',
      );
    },
  );

  test('IT-006 inaccessible new license requires support', () async {
    final inaccessible = _pair('new', givesAccess: false);

    final result = await _runPurchase([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, licenses: [inaccessible]),
    ]);

    expect(result.outcome, SubscriptionPurchaseOutcome.anomaly);
    expect(result.assignmentLicense, isNull);
    expect(result.retry, isNull);
  });

  test('IT-006 orphan subscription requires support', () async {
    const orphan = BillingSubscription(
      id: '30000000-0000-4000-8000-000000000009',
      productId: 'craftsky_plus',
      store: 'app_store',
      status: 'active',
      givesAccess: true,
      pendingPayment: false,
      autoRenewalStatus: 'will_renew',
      anomaly: 'none',
    );

    final result = await _runPurchase([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, extraSubscriptions: [orphan]),
    ]);

    expect(result.outcome, SubscriptionPurchaseOutcome.anomaly);
    expect(result.assignmentLicense, isNull);
    expect(result.retry, isNull);
  });

  test('IT-006 same-subscription recovery preserves assignment', () async {
    final dormant = _pair(
      'same',
      givesAccess: false,
      renewal: 'will_not_renew',
      assigned: 'did:plc:bob',
    );
    final recovered = _pair('same', assigned: 'did:plc:bob');

    final result = await _runPurchase([
      _state(7, 7, licenses: [dormant]),
      _state(7, 7, licenses: [dormant]),
      _state(8, 8, licenses: [recovered]),
    ]);

    expect(result.outcome, SubscriptionPurchaseOutcome.completed);
    expect(result.requiresAssignment, isFalse);
    expect(result.state?.licenses.single.assignedDid, Did.parse('did:plc:bob'));
  });

  test(
    'AT-002 post-lapse new license does not replace dormant assignment',
    () async {
      final old = _pair(
        'old',
        givesAccess: false,
        renewal: 'will_not_renew',
        assigned: 'did:plc:bob',
      );
      final fresh = _pair('new');

      final result = await _runPurchase([
        _state(7, 7, licenses: [old]),
        _state(7, 7, licenses: [old]),
        _state(8, 8, licenses: [old, fresh]),
      ]);

      expect(result.outcome, SubscriptionPurchaseOutcome.completed);
      expect(result.requiresAssignment, isTrue);
      expect(
        result.state?.licenses.first.assignedDid,
        Did.parse('did:plc:bob'),
      );
      expect(result.state?.licenses.last.assignedDid, isNull);
    },
  );

  test('AT-014 reconciled purchase anomaly requires support', () async {
    final anomaly = _pair('new', anomaly: 'duplicate_tier');

    final result = await _runPurchase([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, licenses: [anomaly]),
    ]);

    expect(result.outcome.name, 'anomaly');
    expect(result.state, isNotNull);
    expect(result.retry, isNull);
  });

  test('IT-006 ineligible tier never presents checkout', () async {
    final api = _ScriptedApi([
      _state(7, 7, licenses: [_pair('active')]),
    ]);
    var presentations = 0;
    final controller = _controller(
      api,
      presentPaywall: (_, _) async {
        presentations++;
        return PaywallOutcome.reconcile;
      },
    );

    final result = await controller.purchase(SubscriptionTier.plus);

    expect(result.outcome, SubscriptionPurchaseOutcome.notEligible);
    expect(presentations, 0);
  });

  test('REG-004 paywall cancellation performs no AppView mutation', () async {
    final api = _ScriptedApi([_state(7, 7)]);
    final result = await _controller(
      api,
      presentPaywall: (_, _) async => PaywallOutcome.cancelled,
    ).purchase(SubscriptionTier.plus);

    expect(result.outcome, SubscriptionPurchaseOutcome.cancelled);
    expect(api.reconciliationCalls, 0);
    expect(api.states, isEmpty);
  });

  test(
    'AT-006 unavailable offering remains distinct from provider failure',
    () async {
      final api = _ScriptedApi([_state(7, 7)]);
      final result = await _controller(
        api,
        presentPaywall: (_, _) async => PaywallOutcome.offeringUnavailable,
      ).purchase(SubscriptionTier.plus);

      expect(result.outcome, SubscriptionPurchaseOutcome.offeringUnavailable);
    },
  );

  test('IT-010 confirmed purchase is pending after route disposal', () async {
    var current = true;
    final api = _ScriptedApi([_state(7, 7)]);
    final result = await _controller(
      api,
      isCurrent: () => current,
      presentPaywall: (_, _) async {
        current = false;
        return PaywallOutcome.reconcile;
      },
    ).purchase(SubscriptionTier.plus);

    expect(result.outcome, SubscriptionPurchaseOutcome.pending);
    expect(result.retry, isNotNull);
    expect(api.reconciliationCalls, 0);
  });

  test('IT-014 confirmed purchase is pending after owner removal', () async {
    var registry = _ownerRegistry();
    var confirmed = false;
    final api = _ScriptedApi([_state(7, 7)]);
    final result = await _controller(
      api,
      readRegistry: () => registry,
      presentPaywall: (_, _) async {
        registry = registry.remove('did:plc:alice');
        return PaywallOutcome.reconcile;
      },
      onProviderConfirmed: (_, _) => confirmed = true,
    ).purchase(SubscriptionTier.plus);

    expect(result.outcome, SubscriptionPurchaseOutcome.pending);
    expect(result.retry, isNotNull);
    expect(confirmed, isTrue);
    expect(api.reconciliationCalls, 0);
  });

  test('IT-014 AppView UUID mismatch never presents paywall', () async {
    var presentations = 0;
    final controller = _controller(
      _ScriptedApi([
        _state(
          7,
          7,
          revenueCatAppUserId: '20000000-0000-4000-8000-000000000099',
        ),
      ]),
      presentPaywall: (_, _) async {
        presentations++;
        return PaywallOutcome.reconcile;
      },
    );

    await expectLater(
      controller.purchase(SubscriptionTier.plus),
      throwsA(isA<BillingOwnerGuardException>()),
    );
    expect(presentations, 0);
  });

  test('IT-006 Plus purchase ignores concurrent Business license', () async {
    final business = _pair('new', tier: SubscriptionTier.business);

    final result = await _runPurchase([
      _state(7, 7),
      _state(7, 7),
      _state(8, 8, licenses: [business]),
    ]);

    expect(result.outcome, SubscriptionPurchaseOutcome.pending);
    expect(result.assignmentLicense, isNull);
  });

  test(
    'IT-006 recovered assignment requires exact self-access predicates',
    () async {
      final dormant = _pair(
        'same',
        givesAccess: false,
        renewal: 'will_not_renew',
        assigned: 'did:plc:bob',
      );
      final recovered = _pair('same', assigned: 'did:plc:bob');
      final mismatches = [
        SubscriptionAccess(
          did: Did.parse('did:plc:alice'),
          effectiveTier: SubscriptionTier.plus,
          givesAccess: true,
          assignedTier: SubscriptionTier.plus,
        ),
        SubscriptionAccess(
          did: Did.parse('did:plc:bob'),
          effectiveTier: SubscriptionTier.business,
          givesAccess: true,
          assignedTier: SubscriptionTier.business,
        ),
        SubscriptionAccess(
          did: Did.parse('did:plc:bob'),
          effectiveTier: SubscriptionTier.free,
          givesAccess: false,
          assignedTier: SubscriptionTier.plus,
        ),
      ];

      for (final access in mismatches) {
        final api = _ScriptedApi([
          _state(7, 7, licenses: [dormant]),
          _state(7, 7, licenses: [dormant]),
          _state(8, 8, licenses: [recovered]),
        ]);
        final result = await _controller(
          api,
          readAccess: (_) async => access,
        ).purchase(SubscriptionTier.plus);

        expect(result.outcome, SubscriptionPurchaseOutcome.pending);
      }
    },
  );

  test(
    'AT-007 post-purchase read failure retries reconciliation only',
    () async {
      final api = _ScriptedApi(
        [
          _state(7, 7),
          _state(7, 7),
          _state(8, 8, licenses: [_pair('new')]),
        ],
        failReadNumbers: {2},
      );
      var presentations = 0;
      final result = await _controller(
        api,
        presentPaywall: (_, _) async {
          presentations++;
          return PaywallOutcome.reconcile;
        },
      ).purchase(SubscriptionTier.plus);

      expect(result.outcome, SubscriptionPurchaseOutcome.pending);
      expect(result.retry, isNotNull);
      expect(api.reconciliationCalls, 0);

      final retried = await result.retry!();
      expect(retried.outcome, SubscriptionPurchaseOutcome.completed);
      expect(retried.requiresAssignment, isTrue);
      expect(api.reconciliationCalls, 1);
      expect(presentations, 1);
    },
  );

  test(
    'IT-014 confirmed purchase owner race preserves pending intent',
    () async {
      for (final staleRead in [1, 2]) {
        var registry = _ownerRegistry();
        var presentations = 0;
        final api = _ScriptedApi(
          [_state(7, 7), _state(7, 7)],
          onRead: (read) {
            if (read == staleRead) registry = _beneficiaryActiveRegistry();
          },
        );

        final result = await _controller(
          api,
          readRegistry: () => registry,
          presentPaywall: (_, _) async {
            presentations++;
            return PaywallOutcome.reconcile;
          },
        ).purchase(SubscriptionTier.plus);

        expect(
          result.outcome,
          staleRead == 1
              ? SubscriptionPurchaseOutcome.cancelled
              : SubscriptionPurchaseOutcome.pending,
        );
        expect(presentations, staleRead == 1 ? 0 : 1);
        expect(api.reconciliationCalls, 0);
      }
    },
  );

  test('AT-007 reconciliation POST failure retries without checkout', () async {
    var presentations = 0;
    final api = _ScriptedApi(
      [
        _state(7, 7),
        _state(7, 7),
        _state(7, 7),
        _state(8, 8, licenses: [_pair('new')]),
      ],
      failReconciliationNumbers: {1},
    );
    final result = await _controller(
      api,
      presentPaywall: (_, _) async {
        presentations++;
        return PaywallOutcome.reconcile;
      },
    ).purchase(SubscriptionTier.plus);

    expect(result.outcome, SubscriptionPurchaseOutcome.pending);
    final retried = await result.retry!();
    expect(retried.outcome, SubscriptionPurchaseOutcome.completed);
    expect(api.reconciliationCalls, 2);
    expect(presentations, 1);
  });

  test('AT-007 reconciliation timeout remains pending and retryable', () async {
    var presentations = 0;
    final api = _ScriptedApi([
      _state(7, 7),
      _state(7, 7),
      _state(7, 7),
      _state(7, 7),
      _state(7, 7),
    ]);
    final result = await _controller(
      api,
      presentPaywall: (_, _) async {
        presentations++;
        return PaywallOutcome.reconcile;
      },
    ).purchase(SubscriptionTier.plus);

    expect(result.outcome, SubscriptionPurchaseOutcome.pending);
    api.states.addAll([
      _state(7, 7),
      _state(8, 8, licenses: [_pair('new')]),
    ]);
    final retried = await result.retry!();
    expect(retried.outcome, SubscriptionPurchaseOutcome.completed);
    expect(presentations, 1);
  });

  test(
    'IT-010 resumed foreground does not discard a suspended purchase poll',
    () async {
      var foregroundChecks = 0;
      final result = await _controller(
        _ScriptedApi([
          _state(7, 7),
          _state(7, 7),
          _state(8, 8, licenses: [_pair('new')]),
        ]),
        isForeground: () {
          foregroundChecks++;
          return foregroundChecks != 5;
        },
      ).purchase(SubscriptionTier.plus);

      expect(result.outcome, SubscriptionPurchaseOutcome.pending);
      expect(result.retry, isNotNull);
    },
  );

  test(
    'IT-010 pending purchase reacquires owner after account switching',
    () async {
      var registry = _ownerRegistry().upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final api = _ScriptedApi(
        [
          _state(7, 7),
          _state(7, 7),
          _state(8, 8, licenses: [_pair('new')]),
        ],
        failReadNumbers: {2},
      );
      var presentations = 0;
      final controller = _controller(
        api,
        readRegistry: () => registry,
        presentPaywall: (_, _) async {
          presentations++;
          return PaywallOutcome.reconcile;
        },
      );

      final pending = await controller.purchase(SubscriptionTier.plus);
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:bob'))!,
      );
      registry = registry.activate(
        registry.leaseFor(AccountKey('did:plc:alice'))!,
      );
      final retried = await pending.retry!();

      expect(retried.outcome, SubscriptionPurchaseOutcome.completed);
      expect(retried.requiresAssignment, isTrue);
      expect(api.reconciliationCalls, 1);
      expect(presentations, 1);
    },
  );
}

Future<SubscriptionPurchaseResult> _runPurchase(List<BillingState> states) {
  final api = _ScriptedApi(states);
  return _controller(api).purchase(SubscriptionTier.plus);
}

SubscriptionPurchaseController _controller(
  _ScriptedApi api, {
  Future<PaywallOutcome> Function(BillingOwnerLease, SubscriptionTier)?
  presentPaywall,
  bool Function()? isCurrent,
  bool Function()? isForeground,
  Future<SubscriptionAccess> Function(Did)? readAccess,
  SessionRegistry Function()? readRegistry,
  void Function(SubscriptionTier, BillingState)? onProviderConfirmed,
}) {
  final registry = readRegistry ?? _ownerRegistry;
  return SubscriptionPurchaseController(
    ownerGuard: BillingOwnerGuard(registry),
    api: api,
    presentPaywall: presentPaywall ?? (_, _) async => PaywallOutcome.reconcile,
    wait: (_) async {},
    isCurrent: isCurrent ?? () => true,
    isForeground: isForeground ?? () => true,
    onProviderConfirmed: onProviderConfirmed,
    readAccess:
        readAccess ??
        (did) async => SubscriptionAccess(
          did: did,
          effectiveTier: SubscriptionTier.plus,
          givesAccess: true,
          assignedTier: SubscriptionTier.plus,
        ),
    maxReconciliationAttempts: 3,
  );
}

SessionRegistry _ownerRegistry() => SessionRegistry.empty()
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

SessionRegistry _beneficiaryActiveRegistry() =>
    _ownerRegistry().upsertAndActivate(
      token: 'bob-token',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );

BillingState _state(
  int requested,
  int reconciled, {
  List<({BillingSubscription subscription, BillingLicense license})> licenses =
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
    ...licenses.map((value) => value.subscription),
    ...extraSubscriptions,
  ],
  licenses: licenses.map((value) => value.license).toList(),
);

({BillingSubscription subscription, BillingLicense license}) _pair(
  String suffix, {
  bool givesAccess = true,
  String? renewal = 'will_renew',
  String? assigned,
  SubscriptionTier tier = SubscriptionTier.plus,
  String anomaly = 'none',
}) {
  final discriminator = suffix == 'old'
      ? '1'
      : suffix == 'new'
      ? '2'
      : '3';
  final subscriptionId = '30000000-0000-4000-8000-00000000000$discriminator';
  return (
    subscription: BillingSubscription(
      id: subscriptionId,
      productId: 'craftsky_plus_$suffix',
      store: 'app_store',
      status: 'active',
      givesAccess: givesAccess,
      pendingPayment: false,
      autoRenewalStatus: renewal,
      anomaly: anomaly,
    ),
    license: BillingLicense(
      id: '40000000-0000-4000-8000-00000000000$discriminator',
      subscriptionId: subscriptionId,
      tier: tier,
      assignedDid: assigned == null ? null : Did.parse(assigned),
      assignable: true,
      anomaly: anomaly,
    ),
  );
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
  int reconciliationCalls = 0;
  int billingReads = 0;

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
