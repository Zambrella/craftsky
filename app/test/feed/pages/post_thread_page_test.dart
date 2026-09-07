import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:craftsky_app/feed/pages/post_thread_page.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart'
    hide PostCommentSection;
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_interaction_summary.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

final class _ThreadPinRegistryStorage implements SessionRegistryStorage {
  _ThreadPinRegistryStorage()
    : value = SessionRegistry.empty().upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
      );

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

Post _rootPost(
  String text, {
  int likeCount = 0,
  int repostCount = 0,
  int quoteCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
}) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
  cid: 'bafyroot',
  rkey: 'root',
  text: text,
  tags: const [],
  createdAt: DateTime.utc(2026, 7, 16),
  indexedAt: DateTime.utc(2026, 7, 16),
  author: PostAuthor(
    did: 'did:plc:alice',
    handle: 'alice.craftsky.social',
  ),
  likeCount: likeCount,
  repostCount: repostCount,
  quoteCount: quoteCount,
  replyCount: 0,
  viewerHasLiked: viewerHasLiked,
  viewerHasReposted: viewerHasReposted,
  viewerHasSaved: false,
);

PostCommentSection _section(
  String text, {
  int likeCount = 0,
  int repostCount = 0,
  int quoteCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
  List<CommentItem> comments = const [],
}) => PostCommentSection(
  post: _rootPost(
    text,
    likeCount: likeCount,
    repostCount: repostCount,
    quoteCount: quoteCount,
    viewerHasLiked: viewerHasLiked,
    viewerHasReposted: viewerHasReposted,
  ),
  sort: CommentSort.oldest,
  comments: CommentPage(items: comments),
);

Post _responsePost({
  required String did,
  required String rkey,
  required PostRef parent,
  int likeCount = 2,
}) {
  final root = PostRef(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
    cid: 'bafyroot',
  );
  return _rootPost('response $rkey', likeCount: likeCount).copyWith(
    uri: 'at://$did/social.craftsky.feed.post/$rkey',
    cid: 'bafy$rkey',
    rkey: rkey,
    author: PostAuthor(did: did, handle: '$rkey.craftsky.social'),
    reply: PostReply(root: root, parent: parent),
  );
}

InteractionWriteResponse _interaction(Post post) => InteractionWriteResponse(
  uri: 'at://did:plc:viewer/social.craftsky.feed.like/interaction',
  cid: 'bafyinteraction',
  rkey: 'interaction',
  subject: PostRef(uri: post.uri, cid: post.cid),
  createdAt: DateTime.utc(2026, 9, 6),
);

Post _post({required String rkey, required String text, PostReply? reply}) =>
    Post(
      uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
      cid: 'bafy$rkey',
      rkey: rkey,
      text: text,
      tags: const [],
      createdAt: DateTime.utc(2026, 7, 16),
      indexedAt: DateTime.utc(2026, 7, 16),
      author: PostAuthor(
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
      ),
      likeCount: 0,
      repostCount: 0,
      replyCount: 0,
      viewerHasLiked: false,
      viewerHasReposted: false,
      viewerHasSaved: false,
      reply: reply,
    );

Future<GoRouter> _pumpThreadRoute(
  WidgetTester tester, {
  required FakePostRepository repository,
  required RecordingMessenger messenger,
}) async {
  final router = GoRouter(
    initialLocation: '/feed',
    routes: [
      GoRoute(
        path: '/feed',
        builder: (_, _) => const Scaffold(body: Text('Feed destination')),
      ),
      GoRoute(
        path: '/posts/:did/:rkey',
        builder: (_, state) => FormFactorWidget(
          child: PostThreadPage(
            did: Did.parse(state.pathParameters['did']!),
            rkey: RecordKey.parse(state.pathParameters['rkey']!),
          ),
        ),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        postRepositoryProvider.overrideWithValue(repository),
        authSessionProvider.overrideWith(
          () => SignedInAuthSession(did: 'did:plc:alice'),
        ),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    ),
  );
  unawaited(router.push<void>('/posts/did:plc:alice/root'));
  await tester.pumpAndSettle();
  return router;
}

void main() {
  testWidgets(
    'AT-001 renders only nonzero root summary links immediately after the card',
    (tester) async {
      final response =
          _rootPost(
            'liked response',
            likeCount: 9,
            repostCount: 8,
            quoteCount: 7,
          ).copyWith(
            uri: 'at://did:plc:bob/social.craftsky.feed.post/comment',
            cid: 'bafycomment',
            rkey: 'comment',
            author: PostAuthor(
              did: 'did:plc:bob',
              handle: 'bob.craftsky.social',
            ),
            reply: PostReply(
              root: PostRef(
                uri: AtUri.parse(
                  'at://did:plc:alice/social.craftsky.feed.post/root',
                ),
                cid: Cid.parse('bafyroot'),
              ),
              parent: PostRef(
                uri: AtUri.parse(
                  'at://did:plc:alice/social.craftsky.feed.post/root',
                ),
                cid: Cid.parse('bafyroot'),
              ),
            ),
          );
      final section = _section(
        'thread root',
        likeCount: 3,
        quoteCount: 1,
        comments: [
          CommentItem(
            post: response,
            placement: CommentPlacement.normal,
            replies: const ReplyPage(loaded: false, items: []),
          ),
        ],
      );

      await _pumpThread(
        tester,
        FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
              section,
        ),
      );

      final summary = find.byType(PostInteractionSummary);
      final rootCard = find.byWidgetPredicate(
        (widget) => widget is PostCard && widget.post.uri == section.post.uri,
      );
      final responseCard = find.byWidgetPredicate(
        (widget) => widget is PostCard && widget.post.uri == response.uri,
      );
      final rootSliver = find.ancestor(
        of: rootCard,
        matching: find.byType(SliverToBoxAdapter),
      );

      expect(summary, findsOneWidget);
      expect(
        tester.widget<PostInteractionSummary>(summary).post.uri,
        section.post.uri,
      );
      expect(
        find.descendant(of: rootSliver, matching: summary),
        findsOneWidget,
      );
      expect(
        tester.getTopLeft(rootCard).dy,
        lessThan(tester.getTopLeft(summary).dy),
      );
      expect(
        tester.getTopLeft(summary).dy,
        lessThan(tester.getTopLeft(responseCard).dy),
      );
      expect(find.text('3 Likes'), findsOneWidget);
      expect(find.text('1 Quote'), findsOneWidget);
      expect(find.textContaining('Repost'), findsNothing);
      expect(
        tester
            .widgetList<Text>(
              find.descendant(of: summary, matching: find.byType(Text)),
            )
            .map((text) => text.data),
        ['3 Likes', '1 Quote'],
      );
    },
  );

  testWidgets('AT-001 omits the root summary when every count is zero', (
    tester,
  ) async {
    await _pumpThread(
      tester,
      FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section('zero interactions'),
      ),
    );

    expect(find.byType(PostInteractionSummary), findsNothing);
    expect(
      find.textContaining(RegExp(r'^0 (Likes?|Reposts?|Quotes?)$')),
      findsNothing,
    );
  });

  for (final responseType in ['comment', 'nested reply']) {
    testWidgets(
      'AT-005 $responseType View likes opens the response-specific route',
      (tester) async {
        final rootRef = PostRef(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
          cid: 'bafyroot',
        );
        final comment = _responsePost(
          did: 'did:plc:bob',
          rkey: 'c1',
          parent: rootRef,
        );
        final nested = _responsePost(
          did: 'did:plc:carol',
          rkey: 'r1',
          parent: PostRef(uri: comment.uri, cid: comment.cid),
        );
        final target = responseType == 'comment' ? comment : nested;
        final section = _section(
          'thread root',
          likeCount: 1,
          comments: [
            CommentItem(
              post: comment,
              placement: CommentPlacement.normal,
              replies: ReplyPage(
                loaded: true,
                items: responseType == 'nested reply'
                    ? [ReplyItem(post: nested, flattened: false)]
                    : const [],
              ),
            ),
          ],
        );
        Uri? opened;
        final router = GoRouter(
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) => FormFactorWidget(
                child: PostThreadPage(
                  did: Did.parse('did:plc:alice'),
                  rkey: RecordKey.parse('root'),
                ),
              ),
            ),
            GoRoute(
              path: '/posts/:did/:rkey/likes',
              builder: (_, state) {
                opened = state.uri;
                return const Scaffold(body: Text('Response likes'));
              },
            ),
          ],
        );
        addTearDown(router.dispose);
        await _pumpThreadRouter(
          tester,
          FakePostRepository(
            onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
                section,
          ),
          router,
        );

        final targetCard = find.byWidgetPredicate(
          (widget) => widget is PostCard && widget.post.uri == target.uri,
        );
        await tester.ensureVisible(targetCard);
        await tester.pumpAndSettle();
        await tester.tap(
          find.descendant(
            of: targetCard,
            matching: find.byIcon(CraftskyIconsBold.more),
          ),
        );
        await tester.pumpAndSettle();

        expect(find.text('View likes'), findsOneWidget);
        expect(find.text('View reposts'), findsNothing);
        expect(find.text('View quotes'), findsNothing);
        expect(find.byType(PostInteractionSummary), findsOneWidget);
        expect(
          tester
              .widget<PostInteractionSummary>(
                find.byType(PostInteractionSummary),
              )
              .post
              .uri,
          section.post.uri,
        );

        await tester.tap(find.text('View likes'));
        await tester.pumpAndSettle();

        expect(
          Uri.decodeComponent(opened!.path),
          '/posts/${target.author.did}/${target.rkey}/likes',
        );
        expect(opened!.path, isNot(contains('/did:plc:alice/root/likes')));
      },
    );
  }

  testWidgets(
    'AT-005 zero-like response and root omit response interaction actions',
    (tester) async {
      final rootRef = PostRef(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
        cid: 'bafyroot',
      );
      final comment = _responsePost(
        did: 'did:plc:bob',
        rkey: 'c0',
        parent: rootRef,
        likeCount: 0,
      );
      await _pumpThread(
        tester,
        FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
              _section(
                'thread root',
                comments: [
                  CommentItem(
                    post: comment,
                    placement: CommentPlacement.normal,
                    replies: const ReplyPage(loaded: false, items: []),
                  ),
                ],
              ),
        ),
      );

      for (final card in find.byType(PostCard).evaluate()) {
        final menu = find.descendant(
          of: find.byWidget(card.widget),
          matching: find.byIcon(CraftskyIconsBold.more),
        );
        await tester.tap(menu);
        await tester.pumpAndSettle();
        expect(find.text('View likes'), findsNothing);
        expect(find.text('View reposts'), findsNothing);
        expect(find.text('View quotes'), findsNothing);
        await tester.tapAt(Offset.zero);
        await tester.pumpAndSettle();
      }
      expect(find.byType(PostInteractionSummary), findsNothing);
    },
  );

  testWidgets(
    'REG-002 summary is root-only and View likes is positive-response-only',
    (tester) async {
      final rootRef = PostRef(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
        cid: 'bafyroot',
      );
      final comment = _responsePost(
        did: 'did:plc:bob',
        rkey: 'comment',
        parent: rootRef,
      );
      final nested = _responsePost(
        did: 'did:plc:carol',
        rkey: 'nested',
        parent: PostRef(uri: comment.uri, cid: comment.cid),
      );
      final unliked = _responsePost(
        did: 'did:plc:dana',
        rkey: 'unliked',
        parent: rootRef,
        likeCount: 0,
      );
      final section = _section(
        'thread root',
        likeCount: 4,
        repostCount: 3,
        quoteCount: 2,
        comments: [
          CommentItem(
            post: comment,
            placement: CommentPlacement.normal,
            replies: ReplyPage(
              loaded: true,
              items: [ReplyItem(post: nested, flattened: false)],
            ),
          ),
          CommentItem(
            post: unliked,
            placement: CommentPlacement.normal,
            replies: const ReplyPage(loaded: false, items: []),
          ),
        ],
      );

      await _pumpThread(
        tester,
        FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
              section,
        ),
      );

      expect(find.byType(PostInteractionSummary), findsOneWidget);
      expect(
        tester
            .widget<PostInteractionSummary>(
              find.byType(PostInteractionSummary),
            )
            .post
            .uri,
        section.post.uri,
      );

      for (final expectation in [
        (post: section.post, viewLikes: false),
        (post: comment, viewLikes: true),
        (post: nested, viewLikes: true),
        (post: unliked, viewLikes: false),
      ]) {
        final card = find.byWidgetPredicate(
          (widget) =>
              widget is PostCard && widget.post.uri == expectation.post.uri,
        );
        await tester.ensureVisible(card);
        await tester.pumpAndSettle();
        final menu = tester.widget<CraftskyContextMenuButton>(
          find.descendant(
            of: card,
            matching: find.byType(CraftskyContextMenuButton),
          ),
        );
        final labels = menu.groups
            .expand((group) => group.items)
            .map((item) => item.text);
        expect(labels.contains('View likes'), expectation.viewLikes);
        expect(labels, isNot(contains('View reposts')));
        expect(labels, isNot(contains('View quotes')));
        expect(
          find.descendant(
            of: card,
            matching: find.byType(PostInteractionSummary),
          ),
          findsNothing,
        );
      }
    },
  );

  for (final destination in [
    (
      label: '3 Likes',
      child: 'likes',
      expected: '/posts/did:plc:alice/root/likes',
    ),
    (
      label: '2 Reposts',
      child: 'reposts',
      expected: '/posts/did:plc:alice/root/reposts',
    ),
    (
      label: '1 Quote',
      child: 'quotes',
      expected: '/posts/did:plc:alice/root/quotes',
    ),
  ]) {
    testWidgets(
      'AT-002 ${destination.label} opens its post-specific route '
      'without mutation',
      (tester) async {
        var mutations = 0;
        Uri? opened;
        final section = _section(
          'thread root',
          likeCount: 3,
          repostCount: 2,
          quoteCount: 1,
        );
        final repository = FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
              section,
          onLike: (did, rkey) async {
            mutations++;
            return _interaction(section.post);
          },
          onUnlike: (did, rkey) async => mutations++,
          onRepost: (did, rkey) async {
            mutations++;
            return _interaction(section.post);
          },
          onUnrepost: (did, rkey) async => mutations++,
        );
        final router = GoRouter(
          routes: [
            GoRoute(
              path: '/',
              builder: (_, _) => FormFactorWidget(
                child: PostThreadPage(
                  did: Did.parse('did:plc:alice'),
                  rkey: RecordKey.parse('root'),
                ),
              ),
            ),
            GoRoute(
              path: '/posts/:did/:rkey/${destination.child}',
              builder: (_, state) {
                opened = state.uri;
                return const Scaffold(body: Text('Interaction destination'));
              },
            ),
          ],
        );
        addTearDown(router.dispose);

        await _pumpThreadRouter(tester, repository, router);
        await tester.tap(find.text(destination.label));
        await tester.pumpAndSettle();

        expect(Uri.decodeComponent(opened!.path), destination.expected);
        expect(mutations, 0);
      },
    );
  }

  for (final mutation in [
    (
      name: 'like',
      likes: 0,
      reposts: 0,
      liked: false,
      reposted: false,
      label: '1 Like',
    ),
    (
      name: 'unlike',
      likes: 1,
      reposts: 0,
      liked: true,
      reposted: false,
      label: '1 Like',
    ),
    (
      name: 'repost',
      likes: 0,
      reposts: 0,
      liked: false,
      reposted: false,
      label: '1 Repost',
    ),
    (
      name: 'unrepost',
      likes: 0,
      reposts: 1,
      liked: false,
      reposted: true,
      label: '1 Repost',
    ),
  ]) {
    testWidgets(
      'AT-008 successful ${mutation.name} replacement rebuilds the '
      'model-backed summary',
      (tester) async {
        final section = _section(
          'mutable root',
          likeCount: mutation.likes,
          repostCount: mutation.reposts,
          viewerHasLiked: mutation.liked,
          viewerHasReposted: mutation.reposted,
        );
        final repository = FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
              section,
          onLike: (did, rkey) async => _interaction(section.post),
          onUnlike: (did, rkey) async {},
          onRepost: (did, rkey) async => _interaction(section.post),
          onUnrepost: (did, rkey) async {},
        );

        await _pumpThread(tester, repository);
        final card = tester.widget<PostCard>(find.byType(PostCard).first);
        if (mutation.name == 'like' || mutation.name == 'unlike') {
          card.onLike!();
        } else {
          card.onRepost!();
        }
        await tester.pumpAndSettle();

        if (mutation.name == 'like' || mutation.name == 'repost') {
          expect(find.text(mutation.label), findsOneWidget);
          expect(
            tester
                .widget<PostInteractionSummary>(
                  find.byType(PostInteractionSummary),
                )
                .post,
            tester.widget<PostCard>(find.byType(PostCard).first).post,
          );
        } else {
          expect(find.text(mutation.label), findsNothing);
          expect(find.byType(PostInteractionSummary), findsNothing);
          final updated = tester
              .widget<PostCard>(find.byType(PostCard).first)
              .post;
          expect(updated.viewerHasLiked, isFalse);
          expect(updated.viewerHasReposted, isFalse);
          expect(updated.likeCount, 0);
          expect(updated.repostCount, 0);
        }
      },
    );
  }

  testWidgets('successful root deletion returns to the previous route', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final repository = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          _section('thread root'),
      onDelete: (did, rkey) async {},
    );
    await _pumpThreadRoute(
      tester,
      repository: repository,
      messenger: messenger,
    );

    await tester.tap(find.byIcon(CraftskyIconsBold.more));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();

    expect(find.text('Feed destination'), findsOneWidget);
    expect(messenger.calls.last.$2, 'Post deleted.');
  });

  testWidgets('failed root deletion keeps the thread open', (tester) async {
    final messenger = RecordingMessenger();
    final repository = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          _section('thread root'),
      onDelete: (did, rkey) async => throw Exception('delete failed'),
    );
    await _pumpThreadRoute(
      tester,
      repository: repository,
      messenger: messenger,
    );

    await tester.tap(find.byIcon(CraftskyIconsBold.more));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();

    expect(find.text('thread root'), findsOneWidget);
    expect(find.text('Feed destination'), findsNothing);
    expect(
      messenger.calls.last.$2,
      "Couldn't delete that comment or reply.",
    );
  });

  testWidgets('successful comment deletion keeps the thread open', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final initialSection = _section('thread root');
    final rootRef = PostRef(
      uri: initialSection.post.uri,
      cid: initialSection.post.cid,
    );
    final section = initialSection.copyWith(
      comments: CommentPage(
        items: [
          CommentItem(
            post: _post(
              rkey: 'comment',
              text: 'owned comment',
              reply: PostReply(root: rootRef, parent: rootRef),
            ),
            placement: CommentPlacement.viewerAuthored,
            replies: const ReplyPage(loaded: false, items: []),
          ),
        ],
      ),
    );
    final repository = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          section,
      onDelete: (did, rkey) async {},
    );
    await _pumpThreadRoute(
      tester,
      repository: repository,
      messenger: messenger,
    );

    await tester.tap(find.byIcon(CraftskyIconsBold.more).last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete comment'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();

    expect(find.text('thread root'), findsOneWidget);
    expect(find.text('Feed destination'), findsNothing);
    expect(messenger.calls.last.$2, 'Comment deleted.');
  });

  testWidgets('AT-002 REG-006 pins the owner-authored thread root', (
    tester,
  ) async {
    final targets = <String>[];
    final messenger = RecordingMessenger();
    final section = _section('thread root');
    final repository = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          section,
      onProfilePins: () async => const ProfilePinState(),
      onPin: (did, rkey) async {
        targets.add('$did/$rkey');
        return ProfilePinState(standardPostUri: section.post.uri.value);
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          postRepositoryProvider.overrideWithValue(repository),
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _ThreadPinRegistryStorage(),
          ),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse('did:plc:alice'),
                rkey: RecordKey.parse('root'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(CraftskyIconsBold.more));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Pin post'));
    await tester.pumpAndSettle();

    expect(targets, ['did:plc:alice/root']);
    expect(messenger.calls, [('info', 'Post pinned', null)]);
  });

  testWidgets(
    'REG-006 response liker group preserves report and delete actions',
    (tester) async {
      final rootRef = PostRef(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
        cid: 'bafyroot',
      );
      final comment = _responsePost(
        did: 'did:plc:bob',
        rkey: 'comment',
        parent: rootRef,
      );
      final nested = _responsePost(
        did: 'did:plc:alice',
        rkey: 'nested',
        parent: PostRef(uri: comment.uri, cid: comment.cid),
      );
      final deleted = <String>[];
      final repository = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            _section(
              'thread root',
              comments: [
                CommentItem(
                  post: comment,
                  placement: CommentPlacement.normal,
                  replies: ReplyPage(
                    loaded: true,
                    items: [ReplyItem(post: nested, flattened: false)],
                  ),
                ),
              ],
            ),
        onProfilePins: () async => const ProfilePinState(),
        onDelete: (did, rkey) async => deleted.add('$did/$rkey'),
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            postRepositoryProvider.overrideWithValue(repository),
            authSessionProvider.overrideWith(
              () => SignedInAuthSession(did: 'did:plc:alice'),
            ),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _ThreadPinRegistryStorage(),
            ),
          ],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: FormFactorWidget(
                child: PostThreadPage(
                  did: Did.parse('did:plc:alice'),
                  rkey: RecordKey.parse('root'),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final commentCard = find.byWidgetPredicate(
        (widget) => widget is PostCard && widget.post.uri == comment.uri,
      );
      final nestedCard = find.byWidgetPredicate(
        (widget) => widget is PostCard && widget.post.uri == nested.uri,
      );
      final commentMenu = tester.widget<CraftskyContextMenuButton>(
        find.descendant(
          of: commentCard,
          matching: find.byType(CraftskyContextMenuButton),
        ),
      );
      final nestedMenu = tester.widget<CraftskyContextMenuButton>(
        find.descendant(
          of: nestedCard,
          matching: find.byType(CraftskyContextMenuButton),
        ),
      );

      expect(commentMenu.groups.first.items.single.text, 'View likes');
      expect(
        commentMenu.groups
            .skip(1)
            .expand((group) => group.items)
            .map(
              (item) => item.text,
            ),
        contains('Report comment'),
      );
      expect(nestedMenu.groups.first.items.single.text, 'View likes');
      expect(
        nestedMenu.groups
            .skip(1)
            .expand((group) => group.items)
            .map(
              (item) => item.text,
            ),
        contains('Delete reply'),
      );
      expect(tester.widget<PostCard>(commentCard).onReply, isNotNull);
      expect(tester.widget<PostCard>(nestedCard).onReply, isNotNull);

      commentMenu.groups
          .expand((group) => group.items)
          .singleWhere((item) => item.text == 'Report comment')
          .onPressed!();
      await tester.pumpAndSettle();
      expect(find.text('Report comment'), findsOneWidget);
      Navigator.of(tester.element(find.text('Report comment'))).pop();
      await tester.pumpAndSettle();

      nestedMenu.groups
          .expand((group) => group.items)
          .singleWhere((item) => item.text == 'Delete reply')
          .onPressed!();
      await tester.pumpAndSettle();
      expect(find.text('Delete reply?'), findsOneWidget);
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(deleted, ['did:plc:alice/nested']);
    },
  );

  testWidgets('notification destination 404 shows permanent recovery actions', (
    tester,
  ) async {
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
          throw const ApiBadRequest(
            'post_not_found',
            details: ApiFailureDetails(statusCode: 404),
          ),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [postRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(
              did: Did.parse('did:plc:alice'),
              rkey: RecordKey.parse('root'),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Post'), findsOneWidget);
    expect(find.text('This is no longer available'), findsOneWidget);
    expect(
      find.text('This post or profile may have been deleted or hidden.'),
      findsOneWidget,
    );
    expect(find.widgetWithText(TextButton, 'Back'), findsOneWidget);
    expect(
      find.widgetWithText(TextButton, 'View notifications'),
      findsOneWidget,
    );
    expect(find.text('Retry'), findsNothing);
  });

  testWidgets('permanent refresh error hides previously loaded post content', (
    tester,
  ) async {
    final did = Did.parse('did:plc:alice');
    final rkey = RecordKey.parse('root');
    var calls = 0;
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        if (calls++ == 0) {
          return _section('previously loaded private content');
        }
        throw const ApiBadRequest(
          'post_not_found',
          details: ApiFailureDetails(statusCode: 404),
        );
      },
    );
    final container = ProviderContainer(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      retry: (_, _) => null,
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(did: did, rkey: rkey),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('previously loaded private content'), findsOneWidget);

    container.invalidate(postCommentSectionProvider(did, rkey));
    await tester.pumpAndSettle();

    expect(calls, 2);
    final state = container.read(
      postCommentSectionProvider(did, rkey),
    );
    expect(state.hasError, isTrue);
    expect(state.error, isA<ApiBadRequest>());
    expect(find.text('This is no longer available'), findsOneWidget);
    expect(find.text('previously loaded private content'), findsNothing);
  });

  testWidgets('transient refresh error keeps destination Retry available', (
    tester,
  ) async {
    final did = Did.parse('did:plc:alice');
    final rkey = RecordKey.parse('root');
    var calls = 0;
    final repo = FakePostRepository(
      onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
        if (calls++ == 0) return _section('authenticated cached post');
        throw const ApiNetworkError('offline');
      },
    );
    final container = ProviderContainer(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      retry: (_, _) => null,
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: FormFactorWidget(
            child: PostThreadPage(did: did, rkey: rkey),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('authenticated cached post'), findsOneWidget);

    container.invalidate(postCommentSectionProvider(did, rkey));
    await tester.pumpAndSettle();

    expect(calls, 2);
    expect(find.text('authenticated cached post'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
  });

  testWidgets(
    'permanent recovery actions use back stack and notifications route',
    (
      tester,
    ) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            throw const ApiBadRequest(
              'post_not_found',
              details: ApiFailureDetails(statusCode: 404),
            ),
      );
      final router = GoRouter(
        initialLocation: '/feed',
        routes: [
          GoRoute(
            path: '/feed',
            builder: (_, _) => const Scaffold(body: Text('Feed destination')),
          ),
          GoRoute(
            path: '/notifications',
            builder: (_, _) =>
                const Scaffold(body: Text('Notifications destination')),
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (_, state) => FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse(state.pathParameters['did']!),
                rkey: RecordKey.parse(state.pathParameters['rkey']!),
              ),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          child: MaterialApp.router(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            routerConfig: router,
          ),
        ),
      );
      unawaited(router.push<void>('/posts/did:plc:alice/root'));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(TextButton, 'Back'));
      await tester.pumpAndSettle();
      expect(find.text('Feed destination'), findsOneWidget);

      unawaited(router.push<void>('/posts/did:plc:alice/root'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(TextButton, 'View notifications'));
      await tester.pumpAndSettle();
      expect(find.text('Notifications destination'), findsOneWidget);
    },
  );

  testWidgets(
    'transient destination failure retries in place and then renders',
    (
      tester,
    ) async {
      var calls = 0;
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
          if (calls++ == 0) throw const ApiNetworkError('offline');
          return _section('loaded after destination retry');
        },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          retry: (_, _) => null,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse('did:plc:alice'),
                rkey: RecordKey.parse('root'),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text("That didn't load"), findsOneWidget);
      expect(find.text('Check your connection and try again.'), findsOneWidget);
      expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);
      expect(calls, 1);

      await tester.tap(find.widgetWithText(TextButton, 'Retry'));
      await tester.pumpAndSettle();

      expect(calls, 2);
      expect(find.text('loaded after destination retry'), findsOneWidget);
      expect(find.text('This is no longer available'), findsNothing);
    },
  );

  testWidgets(
    'authentication loss exposes no notification-specific error state',
    (
      tester,
    ) async {
      final repo = FakePostRepository(
        onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async =>
            throw const ApiUnauthorized(
              details: ApiFailureDetails(statusCode: 401),
            ),
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          retry: (_, _) => null,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: FormFactorWidget(
              child: PostThreadPage(
                did: Did.parse('did:plc:alice'),
                rkey: RecordKey.parse('root'),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Post'), findsOneWidget);
      expect(find.text('This is no longer available'), findsNothing);
      expect(find.text("That didn't load"), findsNothing);
      expect(find.widgetWithText(TextButton, 'Retry'), findsNothing);
    },
  );
}

Future<void> _pumpThread(
  WidgetTester tester,
  FakePostRepository repository,
) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repository)],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: FormFactorWidget(
          child: PostThreadPage(
            did: Did.parse('did:plc:alice'),
            rkey: RecordKey.parse('root'),
          ),
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _pumpThreadRouter(
  WidgetTester tester,
  FakePostRepository repository,
  GoRouter router,
) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repository)],
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        routerConfig: router,
      ),
    ),
  );
  await tester.pumpAndSettle();
}
