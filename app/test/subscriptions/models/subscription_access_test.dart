import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('SubscriptionAccess', () {
    test('UT-005 effective Business inherits Plus; dormant access is Free', () {
      final free = SubscriptionAccess.fromMap(
        _json(effective: 'free', givesAccess: false),
      );
      final plus = SubscriptionAccess.fromMap(
        _json(effective: 'plus', givesAccess: true, assigned: 'plus'),
      );
      final business = SubscriptionAccess.fromMap(
        _json(effective: 'business', givesAccess: true, assigned: 'business'),
      );
      final dormant = SubscriptionAccess.fromMap(
        _json(effective: 'free', givesAccess: false, assigned: 'business'),
      );
      expect(
        [
          free.allowsPlus,
          plus.allowsPlus,
          business.allowsPlus,
          dormant.allowsPlus,
        ],
        [false, true, true, false],
      );
      expect(
        [
          free.allowsBusiness,
          plus.allowsBusiness,
          business.allowsBusiness,
          dormant.allowsBusiness,
        ],
        [false, false, true, false],
      );
    });
    test('decodes canonical effective and assigned tier states', () {
      final cases =
          <
            ({
              Map<String, dynamic> json,
              SubscriptionTier effective,
              SubscriptionTier? assigned,
              bool givesAccess,
            })
          >[
            (
              json: {
                'did': 'did:plc:alice',
                'effectiveTier': 'free',
                'givesAccess': false,
              },
              effective: SubscriptionTier.free,
              assigned: null,
              givesAccess: false,
            ),
            (
              json: {
                'did': 'did:plc:alice',
                'effectiveTier': 'plus',
                'givesAccess': true,
                'assignedTier': 'plus',
              },
              effective: SubscriptionTier.plus,
              assigned: SubscriptionTier.plus,
              givesAccess: true,
            ),
            (
              json: {
                'did': 'did:plc:alice',
                'effectiveTier': 'business',
                'givesAccess': true,
                'assignedTier': 'business',
              },
              effective: SubscriptionTier.business,
              assigned: SubscriptionTier.business,
              givesAccess: true,
            ),
            (
              json: {
                'did': 'did:plc:alice',
                'effectiveTier': 'free',
                'givesAccess': false,
                'assignedTier': 'plus',
                'accessEndsAt': '2026-09-30T12:00:00Z',
              },
              effective: SubscriptionTier.free,
              assigned: SubscriptionTier.plus,
              givesAccess: false,
            ),
          ];

      for (final testCase in cases) {
        final access = SubscriptionAccess.fromMap(testCase.json);

        expect(access.did.value, 'did:plc:alice');
        expect(access.effectiveTier, testCase.effective);
        expect(access.assignedTier, testCase.assigned);
        expect(access.givesAccess, testCase.givesAccess);
      }

      expect(
        SubscriptionAccess.fromMap(cases.last.json).accessEndsAt,
        DateTime.utc(2026, 9, 30, 12),
      );
    });

    test('rejects an unknown effective tier', () {
      expect(
        () => SubscriptionAccess.fromMap({
          'did': 'did:plc:alice',
          'effectiveTier': 'enterprise',
          'givesAccess': true,
        }),
        throwsA(anything),
      );
    });

    test('UT-001 rejects contradictory access predicates', () {
      final invalid = [
        _json(effective: 'free', givesAccess: true, assigned: 'plus'),
        _json(effective: 'plus', givesAccess: false, assigned: 'plus'),
        _json(effective: 'business', givesAccess: true, assigned: 'plus'),
        _json(effective: 'plus', givesAccess: true),
        _json(effective: 'plus', givesAccess: true, assigned: 'business'),
      ];

      for (final json in invalid) {
        expect(() => SubscriptionAccess.fromMap(json), throwsFormatException);
      }
    });
  });
}

Map<String, dynamic> _json({
  required String effective,
  required bool givesAccess,
  String? assigned,
}) => {
  'did': 'did:plc:alice',
  'effectiveTier': effective,
  'givesAccess': givesAccess,
  'assignedTier': ?assigned,
};
