import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/shared/browsing/public_read_transport.dart';
import 'package:flutter_test/flutter_test.dart';

// IT-008 / FR-002, FR-006, RULE-001.
void main() {
  test('closed guest transport maps public reads and bounds pages', () {
    final request = PublicReadRequest.resolve('/v1/projects', 'GET', {
      'craftType': ['sewing'],
      'sort': 'popular',
      'limit': '25',
    });
    expect(request.path, '/v1/public/projects');
    expect(request.query, {
      'craftType': ['sewing'],
      'sort': 'popular',
      'limit': '20',
    });
    expect(
      PublicReadRequest.resolve(
        '/v1/profiles/@alice.test/comments',
        'GET',
        {},
      ).path,
      '/v1/public/profiles/@alice.test/comments',
    );
    for (final path in [
      '/v1/profiles/me',
      '/v1/profiles/recent',
      '/v1/search/suggestions',
      '/v1/search/hashtags/top',
      '/v1/search/recent',
      '/v1/notifications',
      'https://evil.test/v1/projects',
    ]) {
      expect(
        () => PublicReadRequest.resolve(path, 'GET', {}),
        throwsFormatException,
      );
    }
    expect(
      () => PublicReadRequest.resolve(
        '/v1/posts/did:plc:alice/key/likes',
        'POST',
        {},
      ),
      throwsFormatException,
    );
  });
  test('public thread and profile adapt only inert display defaults', () {
    final post = {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/key',
      'cid': 'cid',
      'rkey': 'key',
      'text': 'public',
      'createdAt': '2026-11-06T00:00:00Z',
      'tags': <String>[],
      'sponsored': false,
      'likeCount': 0,
      'replyCount': 0,
      'repostCount': 0,
      'author': {
        'did': 'did:plc:alice',
        'handle': 'alice.test',
        'business': false,
      },
    };
    final section =
        PublicReadRequest.resolve(
              '/v1/posts/did:plc:alice/key/comments',
              'GET',
              {'sort': 'follows'},
            ).adapt({
              'post': post,
              'ancestors': [
                {'state': 'unavailable'},
                {'state': 'visible', 'post': post},
              ],
              'replies': {
                'items': [post],
              },
            })!
            as Map;
    final parsed = PostCommentSectionMapper.fromMap(
      Map<String, dynamic>.from(section),
    );
    expect(parsed.ancestors, hasLength(2));
    expect(parsed.ancestors.first.post, isNull);
    expect(parsed.ancestors.last.post?.text, 'public');
    expect(section['sort'], 'oldest');
    expect((section['post'] as Map)['viewerHasLiked'], false);
    expect((section['comments'] as Map)['items'] as List, hasLength(1));
    final replies =
        PublicReadRequest.resolve(
          '/v1/posts/did:plc:alice/key/replies',
          'GET',
          {},
        ).adapt({
          'post': post,
          'replies': {
            'items': [post],
            'cursor': 'next-branch-page',
          },
        });
    final replyPage = ReplyPageMapper.fromMap(
      Map<String, dynamic>.from(replies! as Map),
    );
    expect(replyPage.loaded, isTrue);
    expect(replyPage.items.single.post.text, 'public');
    expect(replyPage.items.single.flattened, isFalse);
    expect(replyPage.cursor, 'next-branch-page');
    final profile =
        adaptPublicDisplay({
              'did': 'did:plc:alice',
              'handle': 'alice.test',
              'links': <Object>[],
              'business': false,
            })!
            as Map;
    expect(profile['business'], isNull);
    expect(profile['isCraftskyProfile'], true);
    expect(profile.containsKey('mutualFollowerCount'), false);
  });
}
