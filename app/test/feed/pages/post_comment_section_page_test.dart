import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

Post _post(String did, String rkey, String text, {int replyCount = 0}) => Post(
  uri: 'at://$did/social.craftsky.feed.post/$rkey',
  cid: 'bafy_$rkey',
  rkey: rkey,
  text: text,
  tags: const [],
  createdAt: DateTime.utc(2026, 5, 1, 12),
  indexedAt: DateTime.utc(2026, 5, 1, 12),
  author: PostAuthor(did: did, handle: '$rkey.craftsky.social'),
  likeCount: 0,
  repostCount: 0,
  replyCount: replyCount,
  viewerHasLiked: false,
  viewerHasReposted: false,
);

Future<void> _pumpCommentSection(
  WidgetTester tester, {
  required FakePostRepository repo,
  String? focus,
  Size size = const Size(390, 1200),
  RecordingMessenger? messenger,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(
              did: 'did:plc:alice',
              rkey: 'root',
              focus: focus,
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
    expect(l10n.postCommentsSortNewest, 'Newest');
    expect(l10n.postCommentsSortFollows, 'Follows');
    expect(l10n.postCommentsViewReplies, 'View replies');
    expect(l10n.postCommentsLoadMoreReplies, 'Load more replies');
    expect(l10n.postCommentsHideReplies, 'Hide replies');
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
        calls.add(focus);
        return PostCommentSection(
          post: root,
          sort: CommentSort.oldest,
          focus: const FocusContext(
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
            focus: const FocusContext(
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
                post: _post('did:plc:carol', 'reply-1', 'reply 1'),
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

    await tester.tap(find.text('View replies'));
    await tester.pumpAndSettle();
    expect(calls, [null]);
    expect(find.text('reply 1'), findsOneWidget);
    expect(find.text('Hide replies'), findsOneWidget);
    expect(find.text('Load more replies'), findsOneWidget);

    await tester.tap(find.text('Load more replies'));
    await tester.pumpAndSettle();
    expect(calls, [null, 'more-replies']);
    expect(find.text('reply 2'), findsOneWidget);

    await tester.tap(find.text('Hide replies'));
    await tester.pumpAndSettle();
    expect(find.text('reply 1'), findsNothing);
    expect(find.text('reply 2'), findsNothing);
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
      onCreate: ({required text, reply}) async {
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
      author: const PostAuthor(
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
      onCreate: ({required text, reply}) async {
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
            focus: const FocusContext(
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
                        replyingTo: const ReplyingToAuthor(
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

    final rootLeft = tester.getTopLeft(find.text('root post')).dx;
    final commentLeft = tester.getTopLeft(find.text('comment')).dx;
    final deepLeft = tester.getTopLeft(find.text('deep focused reply')).dx;
    expect(commentLeft, rootLeft);
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
              : const FocusContext(
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
    await tester.tap(find.text('Newest').last);
    await tester.pumpAndSettle();

    expect(
      tester.getTopLeft(find.text('viewer comment')).dy,
      lessThan(tester.getTopLeft(find.text('focused comment')).dy),
    );
  });
}
