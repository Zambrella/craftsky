import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('PendingAuth sign-in round-trips with its required handle', () {
    final original = PendingAuth.signIn(
      handle: 'a.bsky.social',
      startedAt: DateTime.utc(2026, 4, 21, 12),
    );
    final roundTrip = PendingAuthMapper.fromJson(original.toJson());
    expect(roundTrip.purpose, PendingAuthPurpose.signIn);
    expect(roundTrip.handle, 'a.bsky.social');
    expect(roundTrip.startedAt, DateTime.utc(2026, 4, 21, 12));
  });

  test('UT-009 registration round-trips without identity or session data', () {
    final original = PendingAuth.registration(
      startedAt: DateTime.utc(2026, 8, 30, 12),
    );
    final encoded = original.toMap();
    final roundTrip = PendingAuthMapper.fromMap(encoded);

    expect(roundTrip.purpose, PendingAuthPurpose.registration);
    expect(roundTrip.handle, isNull);
    expect(encoded, <String, Object?>{
      'purpose': 'registration',
      'handle': null,
      'startedAt': '2026-08-30T12:00:00.000Z',
    });
    expect(
      encoded.keys,
      isNot(contains(anyOf('token', 'code', 'receiptId', 'session'))),
    );
    expect(
      () => PendingAuth(
        purpose: PendingAuthPurpose.signIn,
        handle: null,
        startedAt: original.startedAt,
      ),
      throwsArgumentError,
    );
    expect(
      () => PendingAuth(
        purpose: PendingAuthPurpose.registration,
        handle: 'not-allowed.test',
        startedAt: original.startedAt,
      ),
      throwsArgumentError,
    );
  });
}
