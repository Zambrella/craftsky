import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_tabs/profile_posts_tab.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

Post _post(String rkey) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
    cid: 'bafy_$rkey',
    rkey: rkey,
    text: 'post $rkey',
    tags: const [],
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    createdAt: DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
    ),
  );
}

Future<void> _pump(
  WidgetTester tester, {
  required FakePostRepository repo,
  required bool isOwnProfile,
  RecordingMessenger? messenger,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: CustomScrollView(
              slivers: [
                ProfilePostsTab(
                  handle: 'alice.craftsky.social',
                  isOwnProfile: isOwnProfile,
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  group('ProfilePostsTab', () {
    testWidgets('renders posts from userPostsProvider', (tester) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async =>
            PostPage(items: [_post('a'), _post('b')]),
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();

      expect(find.text('post a'), findsOneWidget);
      expect(find.text('post b'), findsOneWidget);
      expect(find.text('New post'), findsNothing);
    });

    testWidgets('shows composer entry point on own profile', (tester) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
      );

      await _pump(tester, repo: repo, isOwnProfile: true);
      await tester.pumpAndSettle();

      expect(find.text('New post'), findsOneWidget);
      expect(find.text('No posts yet.'), findsOneWidget);
    });

    testWidgets('own-profile New post opens chooser and project branch', (
      tester,
    ) async {
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => const PostPage(items: []),
      );

      await _pump(tester, repo: repo, isOwnProfile: true);
      await tester.pumpAndSettle();

      await tester.tap(find.text('New post'));
      await tester.pumpAndSettle();

      expect(find.text('Regular post'), findsOneWidget);
      expect(find.text('Project post'), findsOneWidget);

      await tester.tap(find.text('Project post'));
      await tester.pumpAndSettle();

      expect(find.text('Project post'), findsOneWidget);
      expect(find.byKey(const Key('craftType-select-button')), findsOneWidget);
    });

    testWidgets('scrolling near the end appends the next page', (tester) async {
      final calls = <({String? cursor, int? limit})>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async {
          calls.add((cursor: cursor, limit: limit));
          if (calls.length == 1) {
            return PostPage(
              items: [for (var i = 0; i < 10; i++) _post('a$i')],
              cursor: 'c1',
            );
          }
          expect(cursor, 'c1');
          return PostPage(items: [_post('b')]);
        },
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('post a9'),
        500,
        scrollable: find.byType(Scrollable),
      );
      await tester.pumpAndSettle();

      expect(calls, [
        (cursor: null, limit: 10),
        (cursor: 'c1', limit: 10),
      ]);
      expect(find.text('post a9'), findsOneWidget);
      expect(find.text('post b'), findsOneWidget);
      expect(find.text('Load more posts'), findsNothing);
    });

    testWidgets('wires reply composer, like, and repost actions', (
      tester,
    ) async {
      final calls = <String>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async =>
            PostPage(items: [_post('a')]),
        onLike: (did, rkey) async {
          calls.add('like:$did/$rkey');
          final post = _post(rkey);
          return InteractionWriteResponse(
            uri: 'at://did:plc:viewer/social.craftsky.feed.like/like1',
            cid: 'bafy_like',
            rkey: 'like1',
            subject: PostRef(uri: post.uri, cid: post.cid),
            createdAt: DateTime.now(),
          );
        },
        onRepost: (did, rkey) async {
          calls.add('repost:$did/$rkey');
          final post = _post(rkey);
          return InteractionWriteResponse(
            uri: 'at://did:plc:viewer/social.craftsky.feed.repost/repost1',
            cid: 'bafy_repost',
            rkey: 'repost1',
            subject: PostRef(uri: post.uri, cid: post.cid),
            createdAt: DateTime.now(),
          );
        },
      );

      await _pump(tester, repo: repo, isOwnProfile: false);
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(Icons.chat_bubble_outline));
      await tester.pumpAndSettle();
      expect(find.text('Reply'), findsWidgets);
      await tester.tap(find.byIcon(Icons.close));
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(Icons.favorite_border));
      await tester.pump();
      await tester.tap(find.byIcon(Icons.repeat));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Repost'));
      await tester.pumpAndSettle();

      expect(calls, [
        'like:did:plc:alice/a',
        'repost:did:plc:alice/a',
      ]);
    });

    testWidgets('reply create opens thread focused on the new comment', (
      tester,
    ) async {
      GoRouterState? threadState;
      final root = _post('root');
      final created = _post('created');
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async => PostPage(items: [root]),
        onCreate: ({required text, reply, images}) async => created,
      );
      final router = GoRouter(
        initialLocation: '/',
        routes: [
          GoRoute(
            path: '/',
            builder: (context, state) => const Scaffold(
              body: CustomScrollView(
                slivers: [
                  ProfilePostsTab(
                    handle: 'alice.craftsky.social',
                    isOwnProfile: false,
                  ),
                ],
              ),
            ),
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (context, state) {
              threadState = state;
              return const Scaffold(body: Text('Thread route'));
            },
          ),
        ],
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [postRepositoryProvider.overrideWithValue(repo)],
          child: MessengerScope(
            messenger: RecordingMessenger(),
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              routerConfig: router,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.chat_bubble_outline));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'new comment');
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, 'Reply'));
      await tester.pumpAndSettle();

      expect(find.text('Thread route'), findsOneWidget);
      expect(threadState?.pathParameters['did'], root.author.did);
      expect(threadState?.pathParameters['rkey'], root.rkey);
      expect(threadState?.uri.queryParameters['focus'], created.uri);
      expect(threadState?.extra, isA<Post>());
      expect((threadState!.extra! as Post).uri, created.uri);
      expect((threadState!.extra! as Post).reply?.root.uri, root.uri);
    });

    testWidgets('delete confirmation removes a post', (tester) async {
      final messenger = RecordingMessenger();
      final deleted = <String>[];
      final repo = FakePostRepository(
        onListByAuthor: (_, {cursor, limit}) async =>
            PostPage(items: [_post('a'), _post('b')]),
        onDelete: (_, rkey) async => deleted.add(rkey),
      );

      await _pump(tester, repo: repo, isOwnProfile: true, messenger: messenger);
      await tester.pumpAndSettle();
      await tester.tap(find.byIcon(Icons.more_horiz).first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete post').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(deleted, ['a']);
      expect(find.text('post a'), findsNothing);
      expect(find.text('post b'), findsOneWidget);
      expect(messenger.calls.last.$2, 'Post deleted.');
    });
  });
}
