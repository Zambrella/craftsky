import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('standing ignores additive JSON fields', () {
    final standing = AccountStandingMapper.fromMap({
      'activeStrikeCount': 2,
      'strikeThreshold': 3,
      'thresholdSuspended': false,
      'severeSuspended': false,
      'suspended': false,
      'futureProjection': {'private': false},
    });

    expect(standing.activeStrikeCount, 2);
    expect(standing.strikeThreshold, 3);
    expect(standing.suspended, isFalse);
  });

  test('case references require UUIDv4 and canonicalize case', () {
    expect(
      ModerationCaseReference.parse(
        'mod-550E8400-E29B-41D4-A716-446655440000',
      ).value,
      'MOD-550e8400-e29b-41d4-a716-446655440000',
    );
    expect(
      () => ModerationCaseReference.parse(
        'MOD-550e8400-e29b-11d4-a716-446655440000',
      ),
      throwsFormatException,
    );
  });
}
