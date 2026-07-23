import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  Map<String, dynamic> post(String rkey, String did) => {
    'uri': 'at://$did/social.craftsky.feed.post/$rkey',
    'cid': 'bafy_$rkey',
    'rkey': rkey,
    'text': rkey,
    'tags': <String>[],
    'likeCount': 0,
    'repostCount': 0,
    'replyCount': 0,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'viewerHasSaved': false,
    'createdAt': '2026-05-04T18:23:45.000Z',
    'indexedAt': '2026-05-04T18:23:47.000Z',
    'author': {'did': did, 'handle': '$rkey.craftsky.social'},
  };

  test('decodes comment-section response shape', () {
    final section = PostCommentSectionMapper.fromMap({
      'post': post('root', 'did:plc:alice'),
      'sort': 'oldest',
      'comments': {
        'cursor': 'opaque-next',
        'items': [
          {
            'post': post('comment', 'did:plc:bob'),
            'placement': 'focused',
            'replies': {
              'loaded': true,
              'items': [
                {
                  'post': post('reply', 'did:plc:carol'),
                  'flattened': true,
                  'replyingTo': {
                    'uri': 'at://did:plc:bob/social.craftsky.feed.post/comment',
                    'did': 'did:plc:bob',
                    'handle': 'bob.craftsky.social',
                    'displayName': 'Bob',
                  },
                },
              ],
              'cursor': 'more-replies',
            },
          },
        ],
      },
      'focus': {
        'uri': 'at://did:plc:carol/social.craftsky.feed.post/reply',
        'status': 'included',
        'kind': 'reply',
        'commentUri': 'at://did:plc:bob/social.craftsky.feed.post/comment',
      },
    });

    expect(section.post.rkey, 'root');
    expect(section.sort, CommentSort.oldest);
    expect(section.comments.cursor, 'opaque-next');
    expect(section.comments.items.single.placement, CommentPlacement.focused);
    expect(section.comments.items.single.replies.loaded, isTrue);
    expect(section.comments.items.single.replies.cursor, 'more-replies');
    final reply = section.comments.items.single.replies.items.single;
    expect(reply.flattened, isTrue);
    expect(reply.replyingTo?.handle, 'bob.craftsky.social');
    expect(section.focus?.status, FocusStatus.included);
    expect(section.focus?.kind, FocusKind.reply);
    expect(
      section.toString(),
      [
        'PostCommentSection(',
        'post: at://did:plc:alice/social.craftsky.feed.post/root, ',
        'comments: 1, ',
        'loadedReplies: 1, ',
        'sort: oldest, ',
        'focus: included',
        ')',
      ].join(),
    );
  });

  test('page toString methods summarize list state', () {
    const comments = CommentPage(items: [], cursor: 'next-comments');
    const replies = ReplyPage(
      loaded: true,
      items: [],
      cursor: 'next-replies',
    );

    expect(comments.toString(), 'CommentPage(items: 0, hasMore: true)');
    expect(
      replies.toString(),
      'ReplyPage(loaded: true, items: 0, hasMore: true)',
    );
  });

  test('requires enum-backed comment placement', () {
    Map<String, dynamic> responseWithPlacement(String? placement) => {
      'post': post('root', 'did:plc:alice'),
      'sort': 'oldest',
      'comments': {
        'items': [
          {
            'post': post('comment', 'did:plc:bob'),
            'placement': ?placement,
            'replies': {'loaded': false, 'items': <Map<String, dynamic>>[]},
          },
        ],
      },
    };

    expect(
      () => PostCommentSectionMapper.fromMap(responseWithPlacement(null)),
      throwsA(anything),
    );
    expect(
      () =>
          PostCommentSectionMapper.fromMap(responseWithPlacement('elsewhere')),
      throwsA(anything),
    );
    expect(
      PostCommentSectionMapper.fromMap(
        responseWithPlacement('viewerAuthored'),
      ).comments.items.single.placement,
      CommentPlacement.viewerAuthored,
    );
  });

  test('requires replies object and decodes loaded states', () {
    Map<String, dynamic> responseWithComments(
      List<Map<String, dynamic>> items,
    ) => {
      'post': post('root', 'did:plc:alice'),
      'sort': 'oldest',
      'comments': {'items': items},
    };

    expect(
      () => PostCommentSectionMapper.fromMap(
        responseWithComments([
          {'post': post('missing', 'did:plc:bob'), 'placement': 'normal'},
        ]),
      ),
      throwsA(anything),
    );

    final section = PostCommentSectionMapper.fromMap(
      responseWithComments([
        {
          'post': post('unloaded', 'did:plc:bob'),
          'placement': 'normal',
          'replies': {'loaded': false, 'items': <Map<String, dynamic>>[]},
        },
        {
          'post': post('loaded-empty', 'did:plc:carol'),
          'placement': 'normal',
          'replies': {'loaded': true, 'items': <Map<String, dynamic>>[]},
        },
        {
          'post': post('loaded-more', 'did:plc:dave'),
          'placement': 'normal',
          'replies': {
            'loaded': true,
            'items': [
              {'post': post('reply', 'did:plc:erin'), 'flattened': false},
            ],
            'cursor': 'reply-cursor',
          },
        },
      ]),
    );

    expect(section.comments.items[0].replies.loaded, isFalse);
    expect(section.comments.items[0].replies.items, isEmpty);
    expect(section.comments.items[1].replies.loaded, isTrue);
    expect(section.comments.items[1].replies.items, isEmpty);
    expect(section.comments.items[2].replies.loaded, isTrue);
    expect(section.comments.items[2].replies.items.single.post.rkey, 'reply');
    expect(section.comments.items[2].replies.cursor, 'reply-cursor');
  });

  test('decodes flattened reply metadata structurally', () {
    final section = PostCommentSectionMapper.fromMap({
      'post': post('root', 'did:plc:alice'),
      'sort': 'oldest',
      'comments': {
        'items': [
          {
            'post': post('comment', 'did:plc:bob'),
            'placement': 'normal',
            'replies': {
              'loaded': true,
              'items': [
                {'post': post('direct', 'did:plc:carol'), 'flattened': false},
                {
                  'post': post('flattened', 'did:plc:dave'),
                  'flattened': true,
                  'replyingTo': {
                    'uri':
                        'at://did:plc:carol/social.craftsky.feed.post/direct',
                    'did': 'did:plc:carol',
                    'handle': 'carol.craftsky.social',
                  },
                },
              ],
            },
          },
        ],
      },
    });

    final direct = section.comments.items.single.replies.items.first;
    final flattened = section.comments.items.single.replies.items.last;
    expect(direct.flattened, isFalse);
    expect(direct.replyingTo, isNull);
    expect(flattened.flattened, isTrue);
    expect(flattened.replyingTo?.uri, contains('/direct'));
    expect(flattened.replyingTo?.did, 'did:plc:carol');
    expect(flattened.replyingTo?.handle, 'carol.craftsky.social');
  });
}
