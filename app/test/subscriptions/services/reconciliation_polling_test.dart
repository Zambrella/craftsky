import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'AT-007 polls through concurrent generations to recognized purchase',
    () async {
      final reads = <BillingState>[
        _state(9, 8),
        _state(9, 9),
        _state(9, 9, withLicense: true),
      ];
      var requests = 0;
      var delays = 0;
      final coordinator = ReconciliationCoordinator(
        requestReconciliation: () async => requests++,
        readBillingState: () async => reads.removeAt(0),
        wait: (_) async => delays++,
        maxAttempts: 5,
      );

      final result = await coordinator.reconcile(
        baseline: _state(7, 7),
        intent: ReconciliationIntent.purchase,
      );

      expect(result.outcome, ReconciliationOutcome.completed);
      expect(result.targetGeneration, 9);
      expect(requests, 1);
      expect(delays, 2);
    },
  );

  test(
    'UT-011 timeout and transient reads remain pending without retry',
    () async {
      var reads = 0;
      final coordinator = ReconciliationCoordinator(
        requestReconciliation: () async {},
        readBillingState: () async {
          reads++;
          if (reads == 1) throw StateError('transient');
          return _state(7, 7);
        },
        wait: (_) async {},
        maxAttempts: 3,
      );

      final result = await coordinator.reconcile(
        baseline: _state(7, 7),
        intent: ReconciliationIntent.purchase,
      );

      expect(result.outcome, ReconciliationOutcome.timedOut);
      expect(reads, 3);
    },
  );

  test('UT-011 cancellation stops polling before another read', () async {
    var current = true;
    var reads = 0;
    final coordinator = ReconciliationCoordinator(
      requestReconciliation: () async {},
      readBillingState: () async {
        reads++;
        return _state(8, 7);
      },
      wait: (_) async => current = false,
      isCurrent: () => current,
      maxAttempts: 5,
    );

    final result = await coordinator.reconcile(
      baseline: _state(7, 7),
      intent: ReconciliationIntent.purchase,
    );

    expect(result.outcome, ReconciliationOutcome.cancelled);
    expect(reads, 1);
  });
}

BillingState _state(
  int requested,
  int reconciled, {
  bool withLicense = false,
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: requested,
  reconciledGeneration: reconciled,
  reconciliationStale: requested > reconciled,
  subscriptions: withLicense
      ? const [
          BillingSubscription(
            id: '30000000-0000-4000-8000-000000000001',
            productId: 'craftsky_plus',
            store: 'app_store',
            status: 'active',
            givesAccess: true,
            pendingPayment: false,
            anomaly: 'none',
          ),
        ]
      : const [],
  licenses: withLicense
      ? const [
          BillingLicense(
            id: '40000000-0000-4000-8000-000000000001',
            subscriptionId: '30000000-0000-4000-8000-000000000001',
            tier: SubscriptionTier.plus,
            assignable: true,
            anomaly: 'none',
          ),
        ]
      : const [],
);
