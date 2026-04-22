import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('WhoAmI parses did + handle from JSON', () {
    const json = '{"did":"did:plc:alice","handle":"alice.bsky.social"}';
    final parsed = WhoAmIMapper.fromJson(json);
    expect(parsed.did, 'did:plc:alice');
    expect(parsed.handle, 'alice.bsky.social');
  });
}
