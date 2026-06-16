import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

Post _post(
  String did,
  String rkey,
  String text, {
  int replyCount = 0,
  DateTime? createdAt,
}) => Post(
  uri: 'at://$did/social.craftsky.feed.post/$rkey',
  cid: 'bafy_$rkey',
  rkey: rkey,
  text: text,
  tags: const [],
  createdAt: createdAt ?? DateTime.utc(2026, 5, 1, 12),
  indexedAt: createdAt ?? DateTime.utc(2026, 5, 1, 12),
  author: PostAuthor(did: did, handle: '$rkey.craftsky.social'),
  likeCount: 0,
  repostCount: 0,
  replyCount: replyCount,
  viewerHasLiked: false,
  viewerHasReposted: false,
);

InteractionWriteResponse _likeResponse(Post post) => InteractionWriteResponse(
  uri: 'at://did:plc:viewer/social.craftsky.feed.like/like-${post.rkey}',
  cid: 'bafy_like_${post.rkey}',
  rkey: 'like-${post.rkey}',
  subject: PostRef(uri: post.uri, cid: post.cid),
  createdAt: DateTime.utc(2026, 5, 1, 12, 1),
);

InteractionWriteResponse _repostResponse(Post post) => InteractionWriteResponse(
  uri: 'at://did:plc:viewer/social.craftsky.feed.repost/repost-${post.rkey}',
  cid: 'bafy_repost_${post.rkey}',
  rkey: 'repost-${post.rkey}',
  subject: PostRef(uri: post.uri, cid: post.cid),
  createdAt: DateTime.utc(2026, 5, 1, 12, 1),
);

Future<void> _pumpCommentSection(
  WidgetTester tester, {
  required FakePostRepository repo,
  String? focus,
  Post? initialCreatedPost,
  Size size = const Size(390, 1200),
  RecordingMessenger? messenger,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        postRepositoryProvider.overrideWithValue(repo),
        accountSuggestionRepositoryProvider.overrideWithValue(
          const MockAccountSuggestionRepository(
            accounts: [
              AccountSuggestion(
                did: 'did:plc:carol',
                handle: 'carol.craftsky.social',
                displayName: null,
                avatar: null,
                isCraftskyProfile: true,
              ),
            ],
          ),
        ),
      ],
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(
              did: Did.parse('did:plc:alice'),
              rkey: RecordKey.parse('root'),
              focus: focus == null ? null : AtUri.parse(focus),
              initialCreatedPost: initialCreatedPost,
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('comment section labels are exposed through localizations', (
    tester,
  ) async {
    late AppLocalizations l10n;
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) {
            l10n = AppLocalizations.of(context);
            return const SizedBox.shrink();
          },
        ),
      ),
    );

    expect(l10n.postCommentsSortOldest, 'Oldest');
    expect(l10n.postCommentsSortOldestDescription, 'Conversation order');
    expect(l10n.postCommentsSortNewest, 'Newest');
    expect(l10n.postCommentsSortNewestDescription, 'Most recent on top');
    expect(l10n.postCommentsSortFollows, 'Follows');
    expect(l10n.postCommentsSortFollowsDescription, 'People you follow first');
    expect(l10n.postCommentsViewReplies, 'View replies');
    expect(l10n.postCommentsViewReplyCount(1), 'Show 1 reply');
    expect(l10n.postCommentsViewReplyCount(3), 'Show 3 replies');
    expect(l10n.postCommentsLoadMoreReplies, 'Load more replies');
    expect(l10n.postCommentsHideReplies, 'Hide replies');
    expect(l10n.commentDeleteAction, 'Delete comment');
    expect(l10n.replyDeleteAction, 'Delete reply');
    expect(l10n.postCommentsFocusNotFound, isNotEmpty);
    expect(l10n.postCommentsFocusMismatchedRoot, isNotEmpty);
  });

  testWidgets('focused link renders root context and focused reply', (
    tester,
  ) async {
    const focusUri = 'at://did:plc:carol/social.craftsky.feed.post/reply';
    final calls = <String?>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post('did:plc:bob', 'comment', 'focused branch comment');
    final reply = _post('did:plc:carol', 'reply', 'focused reply');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        calls.add(focus?.value);
        return PostCommentSection(
          post: root,
          sort: CommentSort.oldest,
          focus: FocusContext(
            uri: focusUri,
            status: FocusStatus.included,
            kind: FocusKind.reply,
            commentUri: 'at://did:plc:bob/social.craftsky.feed.post/comment',
          ),
          comments: CommentPage(
            items: [
              CommentItem(
                post: comment,
                placement: CommentPlacement.focused,
                replies: ReplyPage(
                  loaded: true,
                  items: [ReplyItem(post: reply, flattened: false)],
                ),
              ),
            ],
          ),
        );
      },
    );

    await _pumpCommentSection(tester, repo: repo, focus: focusUri);
    await tester.pumpAndSettle();

    expect(calls, [focusUri]);
    expect(find.text('root post'), findsOneWidget);
    expect(find.text('focused branch comment'), findsOneWidget);
    expect(find.text('focused reply'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('focused-comment-target')),
      findsOneWidget,
    );
  });

  testWidgets(
    'root project post uses detail card while comments stay compact',
    (
      tester,
    ) async {
      final root = _post('did:plc:alice', 'root', 'root post').copyWith(
        project: const Project(
          common: ProjectCommon(
            craftType: ProjectOptionCatalogs.sewingCraftToken,
            title: 'Root jacket',
            duration: '3 weekends',
            materials: ['linen'],
          ),
          details: SewingProjectDetails(fitNotes: 'Root fit notes'),
        ),
      );
      final comment = _post('did:plc:bob', 'comment', 'comment').copyWith(
        project: const Project(
          common: ProjectCommon(
            craftType: ProjectOptionCatalogs.knittingCraftToken,
            title: 'Comment sweater',
            duration: '1 month',
          ),
          details: KnittingProjectDetails(finishedSize: '42 in bust'),
        ),
      );
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            PostCommentSection(
              post: root,
              sort: CommentSort.oldest,
              comments: CommentPage(
                items: [
                  CommentItem(
                    post: comment,
                    placement: CommentPlacement.normal,
                    replies: const ReplyPage(loaded: false, items: []),
                  ),
                ],
              ),
            ),
      );

      await _pumpCommentSection(tester, repo: repo);
      await tester.pumpAndSettle();

      expect(find.byKey(const ValueKey('project-detail-card')), findsOneWidget);
      expect(find.text('Root jacket'), findsOneWidget);
      expect(find.text('DURATION'), findsOneWidget);
      expect(find.text('3 weekends'), findsOneWidget);
      expect(find.text('MATERIALS'), findsOneWidget);
      expect(find.text('linen'), findsOneWidget);
      expect(find.text('FIT NOTES'), findsOneWidget);
      expect(find.text('Root fit notes'), findsOneWidget);
      expect(find.text('Comment sweater'), findsOneWidget);
      expect(find.text('1 month'), findsNothing);
    },
  );

  testWidgets('focused reply branch is promoted before normal comments', (
    tester,
  ) async {
    const focusUri = 'at://did:plc:carol/social.craftsky.feed.post/reply';
    final root = _post('did:plc:alice', 'root', 'root post');
    final focusedComment = _post(
      'did:plc:bob',
      'focused-comment',
      'promoted comment',
    );
    final focusedReply = _post('did:plc:carol', 'reply', 'focused reply');
    final normalComment = _post(
      'did:plc:dave',
      'normal-comment',
      'normal comment',
    );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            focus: FocusContext(
              uri: focusUri,
              status: FocusStatus.included,
              kind: FocusKind.reply,
              commentUri:
                  'at://did:plc:bob/social.craftsky.feed.post/focused-comment',
            ),
            comments: CommentPage(
              items: [
                CommentItem(
                  post: focusedComment,
                  placement: CommentPlacement.focused,
                  replies: ReplyPage(
                    loaded: true,
                    items: [ReplyItem(post: focusedReply, flattened: false)],
                    cursor: 'more-focused-replies',
                  ),
                ),
                CommentItem(
                  post: normalComment,
                  placement: CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
    );

    await _pumpCommentSection(tester, repo: repo, focus: focusUri);
    await tester.pumpAndSettle();

    expect(
      tester.getTopLeft(find.text('promoted comment')).dy,
      lessThan(tester.getTopLeft(find.text('normal comment')).dy),
    );
    expect(find.text('focused reply'), findsOneWidget);
  });

  testWidgets('focused reply scrolls into view in a loaded branch', (
    tester,
  ) async {
    const focusUri = 'at://did:plc:target/social.craftsky.feed.post/focused';
    final root = _post('did:plc:alice', 'root', 'root post');
    final focusedComment = _post(
      'did:plc:bob',
      'focused-comment',
      'promoted comment',
    );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            focus: FocusContext(
              uri: focusUri,
              status: FocusStatus.included,
              kind: FocusKind.reply,
              commentUri:
                  'at://did:plc:bob/social.craftsky.feed.post/focused-comment',
            ),
            comments: CommentPage(
              items: [
                CommentItem(
                  post: focusedComment,
                  placement: CommentPlacement.focused,
                  replies: ReplyPage(
                    loaded: true,
                    items: [
                      for (var i = 0; i < 8; i++)
                        ReplyItem(
                          post: _post(
                            'did:plc:reply$i',
                            'reply-$i',
                            'reply $i',
                          ),
                          flattened: false,
                        ),
                      ReplyItem(
                        post: _post(
                          'did:plc:target',
                          'focused',
                          'focused reply',
                        ),
                        flattened: false,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
    );

    await _pumpCommentSection(
      tester,
      repo: repo,
      focus: focusUri,
      size: const Size(390, 420),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));

    final focusedRect = tester.getRect(find.text('focused reply'));
    expect(focusedRect.top, greaterThanOrEqualTo(0));
    expect(focusedRect.bottom, lessThanOrEqualTo(420));
  });

  testWidgets('root post initially shows comments only without replies', (
    tester,
  ) async {
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post('did:plc:bob', 'comment', 'top-level comment');
    final hiddenReply = _post('did:plc:carol', 'reply', 'hidden reply');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: ReplyPage(
                    loaded: false,
                    items: [ReplyItem(post: hiddenReply, flattened: false)],
                  ),
                ),
              ],
            ),
          ),
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();

    expect(find.text('root post'), findsOneWidget);
    expect(find.text('top-level comment'), findsOneWidget);
    expect(find.text('hidden reply'), findsNothing);
  });

  testWidgets('scrolling near the end loads the next comment page', (
    tester,
  ) async {
    final calls = <String?>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        calls.add(cursor);
        if (cursor == null) {
          return PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            comments: CommentPage(
              cursor: 'page-2',
              items: [
                for (var i = 0; i < 10; i++)
                  CommentItem(
                    post: _post('did:plc:bob', 'comment-$i', 'comment $i'),
                    placement: CommentPlacement.normal,
                    replies: const ReplyPage(loaded: false, items: []),
                  ),
              ],
            ),
          );
        }
        return PostCommentSection(
          post: root,
          sort: CommentSort.oldest,
          comments: CommentPage(
            items: [
              CommentItem(
                post: _post('did:plc:carol', 'comment-10', 'comment 10'),
                placement: CommentPlacement.normal,
                replies: const ReplyPage(loaded: false, items: []),
              ),
            ],
          ),
        );
      },
    );

    await _pumpCommentSection(
      tester,
      repo: repo,
      size: const Size(390, 220),
    );
    await tester.pumpAndSettle();

    expect(find.text('comment 10'), findsNothing);
    await tester.scrollUntilVisible(
      find.text('comment 9'),
      250,
      scrollable: find.byType(Scrollable),
    );
    await tester.pumpAndSettle();

    expect(calls, [null, 'page-2']);
    await tester.scrollUntilVisible(
      find.text('comment 10'),
      250,
      scrollable: find.byType(Scrollable),
    );
    expect(find.text('comment 10'), findsOneWidget);
  });

  testWidgets('expands, loads more, and hides child replies', (
    tester,
  ) async {
    final calls = <String?>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post(
      'did:plc:bob',
      'comment',
      'comment with replies',
      replyCount: 12,
    );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
      onListCommentBranchReplies: (did, rkey, {cursor, limit}) async {
        calls.add(cursor);
        if (cursor == null) {
          return ReplyPage(
            loaded: true,
            items: [
              ReplyItem(
                post: _post(
                  'did:plc:carol',
                  'reply-1',
                  'reply 1',
                  replyCount: 4,
                ),
                flattened: false,
              ),
            ],
            cursor: 'more-replies',
          );
        }
        return ReplyPage(
          loaded: true,
          items: [
            ReplyItem(
              post: _post('did:plc:dave', 'reply-2', 'reply 2'),
              flattened: false,
            ),
          ],
        );
      },
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();

    expect(find.text('Show 12 replies'), findsOneWidget);
    await tester.tap(find.text('Show 12 replies'));
    await tester.pumpAndSettle();
    expect(calls, [null]);
    expect(find.text('reply 1'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is DecoratedBox &&
            widget.decoration is BoxDecoration &&
            (widget.decoration as BoxDecoration).color == BrandColors.paper2,
      ),
      findsOneWidget,
    );
    expect(find.text('Hide replies'), findsOneWidget);
    expect(find.text('Load more replies'), findsOneWidget);
    expect(find.text('4'), findsNothing);

    await tester.tap(find.text('Load more replies'));
    await tester.pumpAndSettle();
    expect(calls, [null, 'more-replies']);
    expect(find.text('reply 2'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('reply 1')).dy,
      lessThan(tester.getTopLeft(find.text('reply 2')).dy),
    );

    final hideReplies = find.widgetWithText(TextButton, 'Hide replies');
    await tester.ensureVisible(hideReplies);
    await tester.pumpAndSettle();
    await tester.tap(hideReplies);
    await tester.pumpAndSettle();
    expect(find.text('reply 1'), findsNothing);
    expect(find.text('reply 2'), findsNothing);
  });

  testWidgets('wires like actions for root post, comments, and replies', (
    tester,
  ) async {
    final calls = <String>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post('did:plc:bob', 'comment', 'comment');
    final reply = _post('did:plc:carol', 'reply', 'reply');
    final postsByRkey = {
      root.rkey: root,
      comment.rkey: comment,
      reply.rkey: reply,
    };
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: ReplyPage(
                    loaded: true,
                    items: [ReplyItem(post: reply, flattened: false)],
                  ),
                ),
              ],
            ),
          ),
      onLike: (did, rkey) async {
        calls.add('$did/$rkey');
        return _likeResponse(postsByRkey[rkey]!);
      },
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.favorite_border).at(0));
    await tester.pump();
    await tester.tap(find.byIcon(Icons.favorite_border).at(0));
    await tester.pump();
    await tester.tap(find.byIcon(Icons.favorite_border).at(0));
    await tester.pump();

    expect(calls, [
      'did:plc:alice/root',
      'did:plc:bob/comment',
      'did:plc:carol/reply',
    ]);
    expect(find.byIcon(Icons.favorite), findsNWidgets(3));
    expect(find.byIcon(Icons.favorite_border), findsNothing);
    expect(find.text('1'), findsNWidgets(3));
  });

  testWidgets('wires repost action for the root post', (tester) async {
    final calls = <String>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            comments: const CommentPage(items: []),
          ),
      onRepost: (did, rkey) async {
        calls.add('$did/$rkey');
        return _repostResponse(root);
      },
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.repeat));
    await tester.pump();

    expect(calls, ['did:plc:alice/root']);
    expect(find.text('1'), findsOneWidget);
  });

  testWidgets('selecting comment sort rerenders backend ordered comments', (
    tester,
  ) async {
    final sorts = <CommentSort?>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        sorts.add(sort);
        final newest = sort == CommentSort.newest;
        final items = newest
            ? [
                _post('did:plc:viewer', 'viewer-new', 'viewer new'),
                _post('did:plc:other', 'normal-new', 'normal new'),
              ]
            : [
                _post('did:plc:viewer', 'viewer-old', 'viewer old'),
                _post('did:plc:other', 'normal-old', 'normal old'),
              ];
        return PostCommentSection(
          post: root,
          sort: sort ?? CommentSort.oldest,
          comments: CommentPage(
            items: [
              for (final post in items)
                CommentItem(
                  post: post,
                  placement: post.author.did == 'did:plc:viewer'
                      ? CommentPlacement.viewerAuthored
                      : CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
            ],
          ),
        );
      },
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();
    expect(find.text('viewer old'), findsOneWidget);

    await tester.tap(find.text('Oldest'));
    await tester.pumpAndSettle();
    expect(find.text('Conversation order'), findsOneWidget);
    expect(find.byIcon(Icons.check_box), findsOneWidget);
    await tester.tap(find.text('Newest').last);
    await tester.pumpAndSettle();

    expect(sorts, [CommentSort.oldest, CommentSort.newest]);
    expect(find.text('viewer new'), findsOneWidget);
    expect(find.text('normal new'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('viewer new')).dy,
      lessThan(tester.getTopLeft(find.text('normal new')).dy),
    );
  });

  testWidgets('new top-level comment appears in viewer group after create', (
    tester,
  ) async {
    final root = _post('did:plc:alice', 'root', 'root post');
    final normal = _post('did:plc:other', 'normal', 'normal comment');
    final created = _post('did:plc:viewer', 'created', 'created comment');
    var createCalls = 0;
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: normal,
                  placement: CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
      onCreate: ({required text, reply, images}) async {
        createCalls += 1;
        return created;
      },
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const ValueKey('threadStickyReplyPrompt')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'created comment');
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Reply'));
    await tester.pumpAndSettle();

    expect(createCalls, 1);
    expect(find.text('created comment'), findsOneWidget);
    expect(find.text('1'), findsOneWidget);
    expect(find.byType(TextField), findsNothing);
    await tester.scrollUntilVisible(
      find.text('normal comment'),
      250,
      scrollable: find.byType(Scrollable),
    );
    expect(
      tester.getTopLeft(find.text('created comment')).dy,
      lessThan(tester.getTopLeft(find.text('normal comment')).dy),
    );
    expect(find.text('Oldest'), findsOneWidget);
  });

  testWidgets('new top-level comment appears above a focused branch', (
    tester,
  ) async {
    const focusUri = 'at://did:plc:focus/social.craftsky.feed.post/focused';
    final root = _post('did:plc:alice', 'root', 'root post');
    final focused = _post('did:plc:focus', 'focused', 'focused comment');
    final normal = _post('did:plc:other', 'normal', 'normal comment');
    final created = _post('did:plc:viewer', 'created', 'created comment');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            focus: FocusContext(
              uri: focusUri,
              status: FocusStatus.included,
              kind: FocusKind.comment,
            ),
            comments: CommentPage(
              items: [
                CommentItem(
                  post: focused,
                  placement: CommentPlacement.focused,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
                CommentItem(
                  post: normal,
                  placement: CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
      onCreate: ({required text, reply, images}) async => created,
    );

    await _pumpCommentSection(tester, repo: repo, focus: focusUri);
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const ValueKey('threadStickyReplyPrompt')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'created comment');
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Reply'));
    await tester.pumpAndSettle();

    expect(find.text('created comment'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('created comment')).dy,
      lessThan(tester.getTopLeft(find.text('focused comment')).dy),
    );
  });

  testWidgets('new top-level comment scrolls into view after create', (
    tester,
  ) async {
    final root = _post('did:plc:alice', 'root', 'root post');
    final created = _post(
      'did:plc:viewer',
      'created',
      'created comment',
      createdAt: DateTime.utc(2026, 5, 1, 12, 20),
    );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            comments: CommentPage(
              items: [
                for (var i = 0; i < 12; i++)
                  CommentItem(
                    post: _post(
                      'did:plc:other$i',
                      'comment-$i',
                      'comment $i',
                      createdAt: DateTime.utc(2026, 5, 1, 12, i),
                    ),
                    placement: CommentPlacement.normal,
                    replies: const ReplyPage(loaded: false, items: []),
                  ),
              ],
            ),
          ),
      onCreate: ({required text, reply, images}) async => created,
    );

    await _pumpCommentSection(
      tester,
      repo: repo,
      size: const Size(390, 420),
    );
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text('comment 11'),
      250,
      scrollable: find.byType(Scrollable),
    );

    await tester.tap(find.byKey(const ValueKey('threadStickyReplyPrompt')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'created comment');
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Reply'));
    await tester.pumpAndSettle();

    final createdRect = tester.getRect(find.text('created comment'));
    expect(createdRect.top, greaterThanOrEqualTo(0));
    expect(createdRect.bottom, lessThanOrEqualTo(420));
  });

  testWidgets('initial created comment renders before it is indexed', (
    tester,
  ) async {
    final root = _post('did:plc:alice', 'root', 'root post');
    final created =
        _post(
          'did:plc:viewer',
          'created',
          'created comment',
        ).copyWith(
          reply: PostReply(
            root: PostRef(uri: root.uri, cid: root.cid),
            parent: PostRef(uri: root.uri, cid: root.cid),
          ),
        );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            focus: FocusContext(
              uri: created.uri,
              status: FocusStatus.notFound,
              kind: FocusKind.comment,
            ),
            comments: const CommentPage(items: []),
          ),
    );

    await _pumpCommentSection(
      tester,
      repo: repo,
      focus: created.uri,
      initialCreatedPost: created,
    );
    await tester.pumpAndSettle();

    expect(find.text('created comment'), findsOneWidget);
    expect(find.text('1'), findsOneWidget);
  });

  testWidgets('replying to a collapsed comment loads the visible branch', (
    tester,
  ) async {
    final replyLoadCursors = <String?>[];
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment =
        _post(
          'did:plc:bob',
          'comment',
          'collapsed comment',
          replyCount: 15,
        ).copyWith(
          reply: PostReply(
            root: PostRef(uri: root.uri, cid: root.cid),
            parent: PostRef(uri: root.uri, cid: root.cid),
          ),
        );
    final existingReplies = [
      for (var i = 0; i < 15; i++)
        _post(
          'did:plc:reply$i',
          'existing-$i',
          'existing reply $i',
          createdAt: DateTime.utc(2026, 5, 1, 12, i),
        ),
    ];
    final createdReply = _post(
      'did:plc:viewer',
      'created-reply',
      'created reply',
      createdAt: DateTime.utc(2026, 5, 1, 12, 16),
    );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
      onListCommentBranchReplies: (did, rkey, {cursor, limit}) async {
        replyLoadCursors.add(cursor);
        if (cursor == null) {
          return ReplyPage(
            loaded: true,
            items: [
              for (final reply in existingReplies.take(10))
                ReplyItem(post: reply, flattened: false),
            ],
            cursor: 'more-replies',
          );
        }
        return ReplyPage(
          loaded: true,
          items: [
            for (final reply in existingReplies.skip(10))
              ReplyItem(post: reply, flattened: false),
          ],
        );
      },
      onCreate: ({required text, reply, images}) async => createdReply,
    );

    await _pumpCommentSection(
      tester,
      repo: repo,
      size: const Size(390, 420),
    );
    await tester.pumpAndSettle();
    expect(find.text('Show 15 replies'), findsOneWidget);

    final replyButton = find.widgetWithText(TextButton, 'Reply').first;
    await tester.ensureVisible(replyButton);
    await tester.pumpAndSettle();
    await tester.tap(replyButton);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'created reply');
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Reply'));
    await tester.pumpAndSettle();

    expect(replyLoadCursors, [null, 'more-replies']);
    expect(find.text('existing reply 0'), findsOneWidget);
    expect(find.text('existing reply 14'), findsOneWidget);
    expect(find.text('created reply'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('existing reply 14')).dy,
      lessThan(tester.getTopLeft(find.text('created reply')).dy),
    );
    expect(find.text('Show 15 replies'), findsNothing);
    expect(find.text('Load more replies'), findsNothing);
    final createdRect = tester.getRect(find.text('created reply'));
    expect(createdRect.top, greaterThanOrEqualTo(0));
    expect(createdRect.bottom, lessThanOrEqualTo(420));

    final hideReplies = find.widgetWithText(TextButton, 'Hide replies');
    await tester.scrollUntilVisible(
      hideReplies,
      -250,
      scrollable: find.byType(Scrollable),
    );
    await tester.pumpAndSettle();
    await tester.tap(hideReplies);
    await tester.pumpAndSettle();
    expect(find.text('Show 16 replies'), findsOneWidget);
  });

  testWidgets('replying to a comment increments root comment count', (
    tester,
  ) async {
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post('did:plc:bob', 'comment', 'comment').copyWith(
      reply: PostReply(
        root: PostRef(uri: root.uri, cid: root.cid),
        parent: PostRef(uri: root.uri, cid: root.cid),
      ),
    );
    final createdReply = _post(
      'did:plc:viewer',
      'created-reply',
      'created reply',
    );
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: const ReplyPage(loaded: false, items: []),
                ),
              ],
            ),
          ),
      onCreate: ({required text, reply, images}) async => createdReply,
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();
    expect(find.text('1'), findsNothing);

    await tester.tap(find.widgetWithText(TextButton, 'Reply').first);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'created reply');
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Reply'));
    await tester.pumpAndSettle();

    expect(find.text('1'), findsOneWidget);
  });

  testWidgets('replying to a reply inserts created reply into comment branch', (
    tester,
  ) async {
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post('did:plc:bob', 'comment', 'comment');
    final reply = Post(
      uri: 'at://did:plc:carol/social.craftsky.feed.post/reply',
      cid: 'bafy_reply',
      rkey: 'reply',
      text: 'visible reply',
      tags: const [],
      createdAt: DateTime.utc(2026, 5, 1, 12),
      indexedAt: DateTime.utc(2026, 5, 1, 12),
      author: PostAuthor(
        did: 'did:plc:carol',
        handle: 'carol.craftsky.social',
      ),
      likeCount: 0,
      repostCount: 0,
      replyCount: 0,
      viewerHasLiked: false,
      viewerHasReposted: false,
      reply: PostReply(
        root: PostRef(uri: root.uri, cid: root.cid),
        parent: PostRef(uri: comment.uri, cid: comment.cid),
      ),
    );
    final created = _post(
      'did:plc:viewer',
      'created-reply',
      'created nested reply',
    );
    PostReply? capturedReply;
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: sort ?? CommentSort.oldest,
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: ReplyPage(
                    loaded: true,
                    items: [ReplyItem(post: reply, flattened: false)],
                  ),
                ),
              ],
            ),
          ),
      onCreate: ({required text, reply, images}) async {
        capturedReply = reply;
        return created;
      },
    );

    await _pumpCommentSection(tester, repo: repo);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Reply').last);
    await tester.pumpAndSettle();
    expect(find.byType(TextField), findsOneWidget);
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      startsWith('@carol.craftsky.social'),
    );
    await tester.enterText(
      find.byType(TextField),
      '@carol.craftsky.social created nested reply',
    );
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Reply'));
    await tester.pumpAndSettle();

    expect(capturedReply?.root.uri, root.uri);
    expect(capturedReply?.parent.uri, reply.uri);
    expect(find.text('created nested reply'), findsOneWidget);
  });

  testWidgets('focused deep reply renders without a third visual level', (
    tester,
  ) async {
    const focusUri = 'at://did:plc:dave/social.craftsky.feed.post/deep';
    final root = _post('did:plc:alice', 'root', 'root post');
    final comment = _post('did:plc:bob', 'comment', 'comment');
    final deep = _post('did:plc:dave', 'deep', 'deep focused reply');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          PostCommentSection(
            post: root,
            sort: CommentSort.oldest,
            focus: FocusContext(
              uri: focusUri,
              status: FocusStatus.included,
              kind: FocusKind.reply,
              commentUri: 'at://did:plc:bob/social.craftsky.feed.post/comment',
            ),
            comments: CommentPage(
              items: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.focused,
                  replies: ReplyPage(
                    loaded: true,
                    items: [
                      ReplyItem(
                        post: deep,
                        flattened: true,
                        replyingTo: ReplyingToAuthor(
                          uri:
                              'at://did:plc:carol/social.craftsky.feed.post/reply',
                          did: 'did:plc:carol',
                          handle: 'carol.craftsky.social',
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
    );

    await _pumpCommentSection(tester, repo: repo, focus: focusUri);
    await tester.pumpAndSettle();

    final commentLeft = tester.getTopLeft(find.text('comment')).dx;
    final deepLeft = tester.getTopLeft(find.text('deep focused reply')).dx;
    expect(deepLeft, greaterThan(commentLeft));
    expect(deepLeft - commentLeft, lessThan(80));
  });

  testWidgets('focus promotion clears on sort change', (tester) async {
    const focusUri = 'at://did:plc:focus/social.craftsky.feed.post/focused';
    final root = _post('did:plc:alice', 'root', 'root post');
    final focused = _post('did:plc:focus', 'focused', 'focused comment');
    final viewer = _post('did:plc:viewer', 'viewer', 'viewer comment');
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        final sorted = sort == CommentSort.newest;
        return PostCommentSection(
          post: root,
          sort: sort ?? CommentSort.oldest,
          focus: sorted
              ? null
              : FocusContext(
                  uri: focusUri,
                  status: FocusStatus.included,
                  kind: FocusKind.comment,
                ),
          comments: CommentPage(
            items: sorted
                ? [
                    CommentItem(
                      post: viewer,
                      placement: CommentPlacement.viewerAuthored,
                      replies: const ReplyPage(loaded: false, items: []),
                    ),
                    CommentItem(
                      post: focused,
                      placement: CommentPlacement.normal,
                      replies: const ReplyPage(loaded: false, items: []),
                    ),
                  ]
                : [
                    CommentItem(
                      post: focused,
                      placement: CommentPlacement.focused,
                      replies: const ReplyPage(loaded: false, items: []),
                    ),
                    CommentItem(
                      post: viewer,
                      placement: CommentPlacement.viewerAuthored,
                      replies: const ReplyPage(loaded: false, items: []),
                    ),
                  ],
          ),
        );
      },
    );

    await _pumpCommentSection(tester, repo: repo, focus: focusUri);
    await tester.pumpAndSettle();
    expect(
      tester.getTopLeft(find.text('focused comment')).dy,
      lessThan(tester.getTopLeft(find.text('viewer comment')).dy),
    );

    await tester.tap(find.text('Oldest'));
    await tester.pumpAndSettle();
    expect(find.text('Most recent on top'), findsOneWidget);
    await tester.tap(find.text('Newest').last);
    await tester.pumpAndSettle();

    expect(
      tester.getTopLeft(find.text('viewer comment')).dy,
      lessThan(tester.getTopLeft(find.text('focused comment')).dy),
    );
  });
}
