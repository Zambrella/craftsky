import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('BillingState', () {
    test('decodes camelCase owner state and correlates licenses by ID', () {
      final state = BillingState.fromMap(_ownerFixture());

      expect(state.requestedGeneration, 7);
      expect(state.reconciledGeneration, 6);
      expect(state.reconciliationStale, isTrue);
      expect(state.subscriptions, hasLength(2));
      expect(state.licenses, hasLength(2));
      expect(
        state.licensesBySubscriptionId[state.subscriptions.first.id]?.tier,
        SubscriptionTier.plus,
      );
      expect(
        state.licensesBySubscriptionId[state.subscriptions.last.id]?.tier,
        SubscriptionTier.business,
      );
      expect(state.subscriptions.first.store, 'future_store');
      expect(state.subscriptions.first.status, 'future_status');
      expect(state.subscriptions.first.autoRenewalStatus, 'future_renewal');
      expect(state.subscriptions.first.anomaly, 'future_anomaly');
      expect(
        state.reconciliationRequestedAt,
        DateTime.utc(2026, 9, 10, 10),
      );
      expect(state.toString(), 'BillingState([REDACTED])');
    });

    test('accepts omitted optional values and non-null empty arrays', () {
      final state = BillingState.fromMap({
        'billingAccountId': '10000000-0000-4000-8000-000000000001',
        'revenueCatAppUserId': '20000000-0000-4000-8000-000000000001',
        'requestedGeneration': 0,
        'reconciledGeneration': 0,
        'reconciliationStale': false,
        'subscriptions': <Object>[],
        'licenses': <Object>[],
      });

      expect(state.subscriptions, isEmpty);
      expect(state.licenses, isEmpty);
      expect(state.reconciledAt, isNull);
    });

    test('rejects malformed UUIDs and unresolved relationships', () {
      final invalidUuid = _ownerFixture()..['billingAccountId'] = 'not-a-uuid';
      expect(() => BillingState.fromMap(invalidUuid), throwsFormatException);

      final orphan = _ownerFixture();
      final licenses = orphan['licenses']! as List<Map<String, dynamic>>;
      licenses.first['subscriptionId'] = '30000000-0000-4000-8000-000000000099';
      expect(() => BillingState.fromMap(orphan), throwsFormatException);
    });

    test('rejects unknown tiers and wrong required field types', () {
      final unknownTier = _ownerFixture();
      final licenses = unknownTier['licenses']! as List<Map<String, dynamic>>;
      licenses.first['tier'] = 'enterprise';
      expect(() => BillingState.fromMap(unknownTier), throwsA(anything));

      final wrongType = _ownerFixture()..['requestedGeneration'] = '7';
      expect(() => BillingState.fromMap(wrongType), throwsA(anything));
    });
  });

  test('Assignment decodes its strict camelCase contract', () {
    final assignment = BillingAssignment.fromMap({
      'licenseId': '40000000-0000-4000-8000-000000000001',
      'targetDid': 'did:plc:bob',
      'assignedAt': '2026-09-10T12:00:00Z',
    });

    expect(assignment.targetDid.value, 'did:plc:bob');
    expect(assignment.assignedAt, DateTime.utc(2026, 9, 10, 12));
    expect(assignment.toString(), 'BillingAssignment([REDACTED])');
  });
}

Map<String, dynamic> _ownerFixture() => {
  'billingAccountId': '10000000-0000-4000-8000-000000000001',
  'revenueCatAppUserId': '20000000-0000-4000-8000-000000000001',
  'requestedGeneration': 7,
  'reconciledGeneration': 6,
  'reconciliationRequestedAt': '2026-09-10T10:00:00Z',
  'reconciledAt': '2026-09-10T09:00:00Z',
  'reconciliationStale': true,
  'subscriptions': <Map<String, dynamic>>[
    {
      'id': '30000000-0000-4000-8000-000000000001',
      'productId': 'plus-monthly',
      'store': 'future_store',
      'status': 'future_status',
      'givesAccess': true,
      'pendingPayment': false,
      'autoRenewalStatus': 'future_renewal',
      'currentPeriodStartsAt': '2026-09-01T00:00:00Z',
      'currentPeriodEndsAt': '2026-10-01T00:00:00Z',
      'endsAt': '2026-10-01T00:00:00Z',
      'anomaly': 'future_anomaly',
    },
    {
      'id': '30000000-0000-4000-8000-000000000002',
      'productId': 'business-monthly',
      'store': 'app_store',
      'status': 'active',
      'givesAccess': true,
      'pendingPayment': false,
      'anomaly': 'none',
    },
  ],
  'licenses': <Map<String, dynamic>>[
    {
      'id': '40000000-0000-4000-8000-000000000002',
      'subscriptionId': '30000000-0000-4000-8000-000000000002',
      'tier': 'business',
      'assignable': true,
      'anomaly': 'none',
    },
    {
      'id': '40000000-0000-4000-8000-000000000001',
      'subscriptionId': '30000000-0000-4000-8000-000000000001',
      'tier': 'plus',
      'assignedDid': 'did:plc:bob',
      'assignedAt': '2026-09-10T11:00:00Z',
      'assignable': false,
      'anomaly': 'future_anomaly',
    },
  ],
};
