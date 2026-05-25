import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('StoredSession round-trips through JSON', () {
    final original = StoredSession(
      token: 'tok',
      did: 'did:plc:a',
      handle: 'a.bsky.social',
    );
    final roundTrip = StoredSessionMapper.fromJson(original.toJson());
    expect(roundTrip.token, 'tok');
    expect(roundTrip.did, 'did:plc:a');
    expect(roundTrip.handle, 'a.bsky.social');
  });
}
