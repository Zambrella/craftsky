import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('Profile', () {
    test('decodes follow-state and Craftsky count fields', () {
      final json = {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
        'displayName': 'Alice',
        'description': 'textile person',
        'crafts': ['sewing'],
        'viewerIsFollowing': true,
        'isCraftskyProfile': true,
        'followerCount': 12,
        'followingCount': 34,
      };

      final profile = ProfileMapper.fromMap(json);

      expect(profile.viewerIsFollowing, isTrue);
      expect(profile.isCraftskyProfile, isTrue);
      expect(profile.followerCount, 12);
      expect(profile.followingCount, 34);
    });

    test('allows non-Craftsky profiles with unknown counts', () {
      final json = {
        'did': 'did:plc:carol',
        'handle': 'carol.bsky.social',
        'displayName': 'Carol',
        'description': 'spinner',
        'crafts': <String>[],
        'viewerIsFollowing': false,
        'isCraftskyProfile': false,
      };

      final profile = ProfileMapper.fromMap(json);

      expect(profile.isCraftskyProfile, isFalse);
      expect(profile.viewerIsFollowing, isFalse);
      expect(profile.followerCount, isNull);
      expect(profile.followingCount, isNull);
    });
  });
}
