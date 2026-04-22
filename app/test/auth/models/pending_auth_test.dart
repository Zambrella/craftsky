import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('PendingAuth round-trips through JSON', () {
    final original = PendingAuth(
      handle: 'a.bsky.social',
      startedAt: DateTime.utc(2026, 4, 21, 12),
    );
    final roundTrip = PendingAuthMapper.fromJson(original.toJson());
    expect(roundTrip.handle, 'a.bsky.social');
    expect(roundTrip.startedAt, DateTime.utc(2026, 4, 21, 12));
  });
}
