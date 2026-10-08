import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_rules.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-004 absent tiers are independently purchasable when fresh', () {
    final state = _state();

    expect(
      purchaseEligibility(state, SubscriptionTier.business).eligible,
      isTrue,
    );
    expect(purchaseEligibility(state, SubscriptionTier.plus).eligible, isFalse);
  });

  test('UT-004 AppView none anomaly sentinel permits safe repurchase', () {
    final state = _state(
      givesAccess: false,
      autoRenewalStatus: 'will_not_renew',
      subscriptionAnomaly: 'none',
      licenseAnomaly: 'none',
    );

    expect(purchaseEligibility(state, SubscriptionTier.plus).eligible, isTrue);
  });

  test('UT-004 only exact terminal, reconciled state permits repurchase', () {
    const unsafeRenewals = <String?>[
      null,
      '',
      'will_renew',
      'will_change_product',
      'will_pause',
      'requires_price_increase_consent',
      'has_already_renewed',
      'future_value',
    ];
    for (final renewal in unsafeRenewals) {
      expect(
        purchaseEligibility(
          _state(givesAccess: false, autoRenewalStatus: renewal),
          SubscriptionTier.plus,
        ).eligible,
        isFalse,
        reason: '$renewal',
      );
    }

    expect(
      purchaseEligibility(
        _state(givesAccess: false, autoRenewalStatus: 'will_not_renew'),
        SubscriptionTier.plus,
      ).eligible,
      isTrue,
    );
  });

  test('AT-005 unsafe payment, reconciliation, and anomaly states block', () {
    final states = [
      _state(),
      _state(
        givesAccess: false,
        pendingPayment: true,
        autoRenewalStatus: 'will_not_renew',
      ),
      _state(
        givesAccess: false,
        autoRenewalStatus: 'will_not_renew',
        requestedGeneration: 2,
      ),
      _state(
        givesAccess: false,
        autoRenewalStatus: 'will_not_renew',
        reconciliationStale: true,
      ),
      _state(
        givesAccess: false,
        autoRenewalStatus: 'will_not_renew',
        subscriptionAnomaly: 'future_anomaly',
      ),
      _state(
        givesAccess: false,
        autoRenewalStatus: 'will_not_renew',
        licenseAnomaly: 'future_anomaly',
      ),
    ];

    for (final state in states) {
      expect(
        purchaseEligibility(state, SubscriptionTier.plus).eligible,
        isFalse,
      );
    }
  });

  test('AT-005 dormant assignment is disclosed without mutation', () {
    final state = _state(
      givesAccess: false,
      autoRenewalStatus: 'will_not_renew',
      assignedDid: Did.parse('did:plc:bob'),
    );

    final result = purchaseEligibility(state, SubscriptionTier.plus);

    expect(result.eligible, isTrue);
    expect(result.dormantAssignedDids, [Did.parse('did:plc:bob')]);
    expect(state.licenses.single.assignedDid, Did.parse('did:plc:bob'));
  });

  test('UT-004 unresolved owner relationships fail checkout closed', () {
    final valid = _state();
    final states = [
      _withRelations(valid, licenses: const []),
      _withRelations(valid, subscriptions: const []),
      _withRelations(
        valid,
        licenses: [...valid.licenses, valid.licenses.single],
      ),
    ];

    for (final state in states) {
      expect(
        purchaseEligibility(state, SubscriptionTier.business).reason,
        PurchaseEligibilityReason.anomaly,
      );
    }
  });
}

BillingState _withRelations(
  BillingState state, {
  List<BillingSubscription>? subscriptions,
  List<BillingLicense>? licenses,
}) => BillingState(
  billingAccountId: state.billingAccountId,
  revenueCatAppUserId: state.revenueCatAppUserId,
  requestedGeneration: state.requestedGeneration,
  reconciledGeneration: state.reconciledGeneration,
  reconciliationStale: state.reconciliationStale,
  subscriptions: subscriptions ?? state.subscriptions,
  licenses: licenses ?? state.licenses,
);

BillingState _state({
  SubscriptionTier tier = SubscriptionTier.plus,
  bool givesAccess = true,
  bool pendingPayment = false,
  String? autoRenewalStatus = 'will_renew',
  String subscriptionAnomaly = '',
  String licenseAnomaly = '',
  int requestedGeneration = 1,
  int reconciledGeneration = 1,
  bool reconciliationStale = false,
  Did? assignedDid,
}) => BillingState(
  billingAccountId: '10000000-0000-4000-8000-000000000001',
  revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
  requestedGeneration: requestedGeneration,
  reconciledGeneration: reconciledGeneration,
  reconciliationStale: reconciliationStale,
  subscriptions: [
    BillingSubscription(
      id: '30000000-0000-4000-8000-000000000001',
      productId: 'craftsky_plus',
      store: 'app_store',
      status: 'active',
      givesAccess: givesAccess,
      pendingPayment: pendingPayment,
      autoRenewalStatus: autoRenewalStatus,
      anomaly: subscriptionAnomaly,
    ),
  ],
  licenses: [
    BillingLicense(
      id: '40000000-0000-4000-8000-000000000001',
      subscriptionId: '30000000-0000-4000-8000-000000000001',
      tier: tier,
      assignedDid: assignedDid,
      assignable: true,
      anomaly: licenseAnomaly,
    ),
  ],
);
