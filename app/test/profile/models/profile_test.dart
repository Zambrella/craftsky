import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('Profile', () {
    test('decodes follow-state and CraftSky count fields', () {
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
        'mutualFollowerCount': 5,
        'postCount': 8,
        'postsLast7Days': 2,
        'projectCount': 0,
        'moderation': {'warningKind': 'profile'},
      };

      final profile = ProfileMapper.fromMap(json);

      expect(profile.viewerIsFollowing, isTrue);
      expect(profile.isCraftskyProfile, isTrue);
      expect(profile.followerCount, 12);
      expect(profile.followingCount, 34);
      expect(profile.mutualFollowerCount, 5);
      expect(profile.postCount, 8);
      expect(profile.postsLast7Days, 2);
      expect(profile.projectCount, 0);
      expect(profile.moderation?.warningKind, 'profile');
    });

    test('allows non-CraftSky profiles with unknown counts', () {
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

  group('ProfileAccountPage', () {
    test('decodes display-ready account summaries and opaque cursor', () {
      final page = ProfileAccountPageMapper.fromMap({
        'items': [
          {
            'did': 'did:plc:bob',
            'handle': 'bob.craftsky.social',
            'displayName': 'Bob',
            'description': 'quilts',
            'avatar': 'https://cdn.example/avatar.jpg',
            'isCraftskyProfile': true,
          },
        ],
        'cursor': 'opaque-next',
        'totalCount': 12,
      });

      expect(page.totalCount, 12);
      expect(page.cursor, 'opaque-next');
      expect(page.items, hasLength(1));
      final account = page.items.single;
      expect(account.did.toString(), 'did:plc:bob');
      expect(account.handle.toString(), 'bob.craftsky.social');
      expect(account.displayName, 'Bob');
      expect(account.description, 'quilts');
      expect(account.avatar, 'https://cdn.example/avatar.jpg');
      expect(account.isCraftskyProfile, isTrue);
    });

    test('decodes missing cursor as null', () {
      final page = ProfileAccountPageMapper.fromMap({
        'items': <Map<String, dynamic>>[],
        'totalCount': 0,
      });

      expect(page.items, isEmpty);
      expect(page.cursor, isNull);
      expect(page.totalCount, 0);
    });
  });
}
