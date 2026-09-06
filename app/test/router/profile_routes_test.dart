import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-009 known profiles use a typed canonical DID route', () {
    expect(const ProfileRoute().location, '/profile');
    expect(
      UserProfileRoute(did: Did.parse('did:plc:alice')).location,
      '/profiles/did%3Aplc%3Aalice',
    );
  });

  test('UT-009 explicit handle input has a distinct alias route', () {
    expect(
      ProfileAliasRoute(handle: Handle.parse('old.example')).location,
      '/profiles/@old.example',
    );
  });
}
