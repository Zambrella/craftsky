import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/reconciliation_coordinator.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-005 first observed post-baseline generation becomes target', () {
    final tracker = ReconciliationTracker(
      baseline: _state(requested: 7, reconciled: 7),
      intent: ReconciliationIntent.purchase,
    );

    expect(
      tracker.evaluate(_state(requested: 9, reconciled: 8)).decision,
      ReconciliationDecision.pending,
    );
    expect(tracker.targetGeneration, 9);
    expect(
      tracker.evaluate(_state(requested: 9, reconciled: 9)).decision,
      ReconciliationDecision.pending,
    );
    expect(
      tracker
          .evaluate(_state(requested: 9, reconciled: 9, includeLicense: true))
          .decision,
      ReconciliationDecision.completed,
    );
  });

  test('UT-005 unchanged restore completes but unchanged purchase waits', () {
    final baseline = _state(requested: 7, reconciled: 7);
    final reconciled = _state(requested: 8, reconciled: 8);

    expect(
      ReconciliationTracker(
        baseline: baseline,
        intent: ReconciliationIntent.restore,
      ).evaluate(reconciled).decision,
      ReconciliationDecision.completed,
    );
    expect(
      ReconciliationTracker(
        baseline: baseline,
        intent: ReconciliationIntent.purchase,
      ).evaluate(reconciled).decision,
      ReconciliationDecision.pending,
    );
  });

  test('UT-005 same subscription recovery and anomaly are recognized', () {
    final baseline = _state(
      requested: 7,
      reconciled: 7,
      includeLicense: true,
      givesAccess: false,
    );

    expect(
      ReconciliationTracker(
            baseline: baseline,
            intent: ReconciliationIntent.purchase,
          )
          .evaluate(
            _state(
              requested: 8,
              reconciled: 8,
              includeLicense: true,
            ),
          )
          .decision,
      ReconciliationDecision.completed,
    );
    expect(
      ReconciliationTracker(
            baseline: baseline,
            intent: ReconciliationIntent.purchase,
          )
          .evaluate(
            _state(
              requested: 8,
              reconciled: 8,
              includeLicense: true,
              anomaly: 'future_anomaly',
            ),
          )
          .decision,
      ReconciliationDecision.failed,
    );
  });

  test('UT-005 purchase comparison can predate generation baseline', () {
    final beforeProvider = _state(requested: 7, reconciled: 7);
    final afterProvider = _state(
      requested: 7,
      reconciled: 7,
      includeLicense: true,
    );
    final tracker = ReconciliationTracker(
      baseline: afterProvider,
      comparisonBaseline: beforeProvider,
      intent: ReconciliationIntent.purchase,
    );

    expect(
      tracker
          .evaluate(
            _state(
              requested: 8,
              reconciled: 8,
              includeLicense: true,
            ),
          )
          .decision,
      ReconciliationDecision.completed,
    );
  });

  test('UT-005 AppView none anomaly sentinel permits reconciliation', () {
    final tracker = ReconciliationTracker(
      baseline: _state(requested: 7, reconciled: 7),
      intent: ReconciliationIntent.restore,
    );

    final result = tracker.evaluate(
      _state(
        requested: 8,
        reconciled: 8,
        anomaly: 'none',
        includeLicense: true,
      ),
    );

    expect(result.decision, ReconciliationDecision.completed);
  });

  test('UT-005 unresolved relationship fails reconciled state', () {
    final tracker = ReconciliationTracker(
      baseline: _state(requested: 7, reconciled: 7),
      intent: ReconciliationIntent.restore,
    );

    final result = tracker.evaluate(
      _state(
        requested: 8,
        reconciled: 8,
        includeSubscription: true,
      ),
    );

    expect(result.decision, ReconciliationDecision.failed);
  });
}

BillingState _state({
  required int requested,
  required int reconciled,
  bool includeSubscription = false,
  bool includeLicense = false,
  bool givesAccess = true,
  String anomaly = '',
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: requested,
  reconciledGeneration: reconciled,
  reconciliationStale: requested > reconciled,
  subscriptions: includeSubscription || includeLicense
      ? [
          BillingSubscription(
            id: '30000000-0000-4000-8000-000000000001',
            productId: 'craftsky_plus',
            store: 'app_store',
            status: 'active',
            givesAccess: givesAccess,
            pendingPayment: false,
            autoRenewalStatus: 'will_renew',
            anomaly: anomaly,
          ),
        ]
      : const [],
  licenses: includeLicense
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
