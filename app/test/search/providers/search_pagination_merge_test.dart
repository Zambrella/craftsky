import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/providers/search_pagination.dart';
import 'package:flutter_test/flutter_test.dart';

Map<String, dynamic> _postMap(String rkey) => {
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
};

Post _post(String rkey) => PostMapper.fromMap(_postMap(rkey));

ProfileSearchResult _profile({
  required String did,
  required String handle,
}) => ProfileSearchResult(
  did: did,
  handle: handle,
  isCraftskyProfile: true,
  viewerIsFollowing: false,
);

void main() {
  setUpAll(initializeMappers);

  test(
    'UT-008 appendUniquePosts keeps existing duplicate and appends new rows',
    () {
      final merged = appendUniquePosts([_post('a')], [_post('a'), _post('b')]);

      expect(merged.map((post) => post.rkey.toString()), ['a', 'b']);
    },
  );

  test('UT-011 same DID with a new handle keeps one current result', () {
    final current = _profile(
      did: 'did:plc:alice',
      handle: 'alice.old.example',
    );

    final merged = appendUniqueProfiles(
      [
        current,
      ],
      [
        _profile(did: 'did:plc:alice', handle: 'alice.new.example'),
      ],
    );

    expect(merged, [same(current)]);
  });

  test('UT-011 same handle with different DIDs stays two identities', () {
    final merged = appendUniqueProfiles(
      [
        _profile(did: 'did:plc:alice', handle: 'shared.example'),
      ],
      [
        _profile(did: 'did:plc:bob', handle: 'shared.example'),
      ],
    );

    expect(merged.map((profile) => profile.did.toString()), [
      'did:plc:alice',
      'did:plc:bob',
    ]);
  });

  test('UT-011 adjacent-page duplicate DID does not duplicate', () {
    final merged = appendUniqueProfiles(
      [
        _profile(did: 'did:plc:alice', handle: 'alice.example'),
      ],
      [
        _profile(did: 'did:plc:alice', handle: 'alice.example'),
        _profile(did: 'did:plc:bob', handle: 'bob.example'),
      ],
    );

    expect(merged.map((profile) => profile.did.toString()), [
      'did:plc:alice',
      'did:plc:bob',
    ]);
  });
}
