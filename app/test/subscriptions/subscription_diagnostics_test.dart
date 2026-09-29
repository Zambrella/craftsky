import 'dart:io';

import 'package:craftsky_app/auth/models/billing_owner_binding.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-009 billing diagnostics redact identifier and payment canaries', () {
    const uuid = '90000000-0000-4000-8000-000000000009';
    const token = 'secret-session-token';
    const transaction = 'transaction-provider-999';
    final values = <Object>[
      BillingOwnerBinding(
        did: 'did:plc:sensitive',
        revenueCatAppUserId: uuid,
      ),
      const BillingState(
        billingAccountId: uuid,
        revenueCatAppUserId: uuid,
        requestedGeneration: 1,
        reconciledGeneration: 1,
        reconciliationStale: false,
        subscriptions: [],
        licenses: [],
      ),
      SubscriptionAccess(
        did: Did.parse('did:plc:sensitive'),
        effectiveTier: SubscriptionTier.plus,
        givesAccess: true,
      ),
      const BillingOwnerGuardException('owner_session_changed'),
      const RevenueCatUnavailableException(),
    ];

    final diagnostics = values.join(' ');
    for (final canary in [uuid, token, transaction, 'did:plc:sensitive']) {
      expect(diagnostics, isNot(contains(canary)));
    }
  });

  test('REG-006 native adapter installs no raw provider log forwarding', () {
    final source = File(
      'lib/subscriptions/services/revenuecat_service_native.dart',
    ).readAsStringSync();

    expect(source, isNot(contains('setLogHandler')));
    expect(source, isNot(contains('PlatformException')));
    expect(source, isNot(contains('CustomerInfo')));
    expect(source, isNot(contains('logOut')));
  });
}
