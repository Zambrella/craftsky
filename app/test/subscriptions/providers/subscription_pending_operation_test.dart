import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_pending_operation_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('late callbacks cannot mutate or clear a replacement operation', () {
    final container = ProviderContainer.test();
    addTearDown(container.dispose);
    final controller = container.read(
      subscriptionPendingOperationProvider.notifier,
    );
    final did = Did.parse('did:plc:alice');
    expect(controller.beginPurchase(did, SubscriptionTier.plus), isTrue);
    final original = container.read(subscriptionPendingOperationProvider)!;
    controller
      ..recordPurchase(
        original.operationId,
        SubscriptionTier.plus,
        _state,
      )
      ..clearUnconfirmed(original.operationId);
    expect(
      container.read(subscriptionPendingOperationProvider)?.canReconcile,
      isTrue,
    );
    // A different route finishes reconciliation and begins another purchase.
    controller.clear(original.operationId);
    expect(controller.beginPurchase(did, SubscriptionTier.plus), isTrue);
    final replacement = container.read(subscriptionPendingOperationProvider)!;
    controller
      ..recordPurchase(original.operationId, SubscriptionTier.plus, _state)
      ..clearUnconfirmed(original.operationId)
      ..clear(original.operationId);
    expect(
      container.read(subscriptionPendingOperationProvider),
      same(replacement),
    );
    expect(replacement.canReconcile, isFalse);
    controller.clearUnconfirmed(replacement.operationId);
    expect(container.read(subscriptionPendingOperationProvider), isNull);
  });
}

const _state = BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: 1,
  reconciledGeneration: 1,
  reconciliationStale: false,
  subscriptions: [],
  licenses: [],
);
