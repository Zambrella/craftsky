import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_presentation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-010 maps distinct owner states without inferring access', () {
    expect(
      projectTierPresentation(_state()),
      SubscriptionTierPresentation.empty,
    );
    expect(
      projectTierPresentation(_state(status: 'active', givesAccess: true)),
      SubscriptionTierPresentation.active,
    );
    expect(
      projectTierPresentation(_state(status: 'cancelled', givesAccess: true)),
      SubscriptionTierPresentation.canceledAccessible,
    );
    expect(
      projectTierPresentation(_state(status: 'expired')),
      SubscriptionTierPresentation.dormant,
    );
    expect(
      projectTierPresentation(_state(status: 'active', stale: true)),
      SubscriptionTierPresentation.stale,
    );
    expect(
      projectTierPresentation(_state(status: 'active', pending: true)),
      SubscriptionTierPresentation.pending,
    );
    expect(
      projectTierPresentation(_state(status: 'active', anomaly: 'unexpected')),
      SubscriptionTierPresentation.anomaly,
    );
  });

  test('UT-010 unknown inactive provider state uses a safe fallback', () {
    final presentation = projectTierPresentation(
      _state(status: 'future_provider_value'),
    );

    expect(presentation, SubscriptionTierPresentation.statusUnavailable);
    expect(
      effectiveTierForPresentation(
        _state(status: 'future_provider_value', givesAccess: true),
        _access(SubscriptionTier.free),
      ),
      SubscriptionTier.free,
    );
  });

  for (final status in [
    'trialing',
    'in_grace_period',
    'in_billing_retry',
    'future_provider_value',
  ]) {
    test('$status preserves authoritative paid access presentation', () {
      final state = _state(status: status, givesAccess: true);
      expect(
        projectTierPresentation(state),
        SubscriptionTierPresentation.active,
      );
      expect(
        projectLicensePresentation(state, state.licenses.single),
        SubscriptionTierPresentation.active,
      );
      expect(
        effectiveTierForPresentation(state, _access(SubscriptionTier.free)),
        SubscriptionTier.free,
      );
    });
  }

  test('UT-010 AppView none anomaly sentinel renders healthy state', () {
    expect(
      projectTierPresentation(
        _state(
          status: 'active',
          givesAccess: true,
          anomaly: 'none',
        ),
      ),
      SubscriptionTierPresentation.active,
    );
  });
}

SubscriptionAccess _access(SubscriptionTier tier) => SubscriptionAccess(
  did: Did.parse('did:plc:alice'),
  effectiveTier: tier,
  givesAccess: tier != SubscriptionTier.free,
);

BillingState _state({
  String? status,
  bool givesAccess = false,
  bool stale = false,
  bool pending = false,
  String anomaly = '',
}) {
  final hasLicense = status != null;
  return BillingState(
    billingAccountId: '10000000-0000-4000-8000-000000000001',
    revenueCatAppUserId: '20000000-0000-4000-8000-000000000001',
    requestedGeneration: stale ? 2 : 1,
    reconciledGeneration: 1,
    reconciliationStale: stale,
    subscriptions: hasLicense
        ? [
            BillingSubscription(
              id: '30000000-0000-4000-8000-000000000001',
              productId: 'craftsky_plus',
              store: 'app_store',
              status: status,
              givesAccess: givesAccess,
              pendingPayment: pending,
              autoRenewalStatus: givesAccess ? 'will_renew' : 'will_not_renew',
              anomaly: anomaly,
            ),
          ]
        : const [],
    licenses: hasLicense
        ? [
            BillingLicense(
              id: '40000000-0000-4000-8000-000000000001',
              subscriptionId: '30000000-0000-4000-8000-000000000001',
              tier: SubscriptionTier.plus,
              assignable: true,
              anomaly: anomaly,
            ),
          ]
        : const [],
  );
}
