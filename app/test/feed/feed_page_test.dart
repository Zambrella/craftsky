import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'fakes/fake_post_repository.dart';
import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';

Map<String, dynamic> _postMap({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  int replyCount = 3,
  bool viewerHasReplied = false,
}) => {
  'uri': 'at://$did/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'timeline post $rkey',
  'tags': <String>[],
  'likeCount': 1,
  'repostCount': 2,
  'replyCount': replyCount,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasReplied': viewerHasReplied,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did, 'handle': handle},
};

Post _post(
  String rkey, {
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  int replyCount = 3,
  bool viewerHasReplied = false,
}) => PostMapper.fromMap(
  _postMap(
    rkey: rkey,
    did: did,
    handle: handle,
    replyCount: replyCount,
    viewerHasReplied: viewerHasReplied,
  ),
);

InteractionWriteResponse _interaction(Post post) => InteractionWriteResponse(
  uri: 'at://did:plc:viewer/social.craftsky.feed.like/like1',
  cid: 'bafy_like',
  rkey: 'like1',
  subject: PostRef(uri: post.uri, cid: post.cid),
  createdAt: DateTime.parse('2026-05-04T18:25:00.000Z'),
);

Future<void> _pump(
  WidgetTester tester,
  FakePostRepository repo, {
  List<dynamic> overrides = const [],
  RecordingMessenger? messenger,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from([
        postRepositoryProvider.overrideWithValue(repo),
        ...overrides,
      ]),
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const FeedPage(),
        ),
      ),
    ),
  );
}

void main() {
  setUpAll(initializeMappers);

  testWidgets('FeedPage renders timeline loading state', (tester) async {
    final gate = Completer<PostPage>();
    await _pump(
      tester,
      FakePostRepository(onListTimeline: ({cursor, limit}) => gate.future),
    );

    expect(find.text('Feed'), findsOneWidget);
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.text('timeline post a'), findsNothing);

    gate.complete(const PostPage(items: []));
  });

  testWidgets('FeedPage renders loaded timeline post cards', (tester) async {
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            PostPage(items: [_post('a'), _post('b')]),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('timeline post a'), findsOneWidget);
    expect(find.text('timeline post b'), findsOneWidget);
    expect(find.textContaining('@alice.craftsky.social'), findsWidgets);
  });

  testWidgets('FeedPage renders empty state without suggestions', (
    tester,
  ) async {
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => const PostPage(items: []),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Your feed is quiet.'), findsOneWidget);
    expect(find.textContaining('recommend'), findsNothing);
    expect(find.textContaining('Discover'), findsNothing);
  });

  testWidgets('FeedPage initial error can retry first page', (tester) async {
    var calls = 0;
    var allowSuccess = false;
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          if (!allowSuccess) throw Exception('boom');
          return PostPage(items: [_post('a')]);
        },
      ),
    );

    await tester.pump();
    await tester.pump();
    expect(find.text("Feed didn't load."), findsOneWidget);

    allowSuccess = true;
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(calls, greaterThanOrEqualTo(2));
    expect(find.text('timeline post a'), findsOneWidget);
  });

  testWidgets('FeedPage scroll near end appends next timeline page', (
    tester,
  ) async {
    var calls = 0;
    String? nextCursor;
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          if (calls == 1) {
            return PostPage(
              items: [for (var i = 0; i < 12; i++) _post('page1-$i')],
              cursor: 'c1',
            );
          }
          nextCursor = cursor;
          return PostPage(items: [_post('page2')]);
        },
      ),
    );

    await tester.pumpAndSettle();
    expect(find.text('timeline post page1-0'), findsOneWidget);

    await tester.drag(find.byType(CustomScrollView), const Offset(0, -1200));
    await tester.pumpAndSettle();

    expect(nextCursor, 'c1');
    await tester.scrollUntilVisible(
      find.text('timeline post page2'),
      400,
      scrollable: find.byType(Scrollable),
    );
    expect(find.text('timeline post page2'), findsOneWidget);
  });

  testWidgets('FeedPage load-more error preserves posts and retries cursor', (
    tester,
  ) async {
    var calls = 0;
    final nextCursors = <String?>[];
    var allowNextPage = false;
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          calls++;
          if (calls == 1) {
            return PostPage(
              items: [for (var i = 0; i < 12; i++) _post('page1-$i')],
              cursor: 'c1',
            );
          }
          nextCursors.add(cursor);
          if (!allowNextPage) throw Exception('next page failed');
          return PostPage(items: [_post('page2')]);
        },
      ),
    );

    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -1200));
    await tester.pumpAndSettle();

    expect(find.textContaining('timeline post page1-'), findsWidgets);
    await tester.scrollUntilVisible(
      find.text('Retry'),
      400,
      scrollable: find.byType(Scrollable),
    );
    expect(find.text('Retry'), findsOneWidget);
    expect(nextCursors, ['c1']);

    allowNextPage = true;
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(nextCursors, ['c1', 'c1']);
    await tester.scrollUntilVisible(
      find.text('timeline post page2'),
      400,
      scrollable: find.byType(Scrollable),
    );
    expect(find.text('timeline post page2'), findsOneWidget);
  });

  testWidgets('FeedPage row tap opens thread route', (tester) async {
    GoRouterState? threadState;
    final router = GoRouter(
      initialLocation: '/',
      routes: [
        GoRoute(path: '/', builder: (context, state) => const FeedPage()),
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
        overrides: [
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async =>
                  PostPage(items: [_post('tapme')]),
            ),
          ),
        ],
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );

    await tester.pumpAndSettle();
    await tester.tap(find.text('timeline post tapme'));
    await tester.pumpAndSettle();

    expect(find.text('Thread route'), findsOneWidget);
    expect(threadState?.pathParameters['did'], 'did:plc:alice');
    expect(threadState?.pathParameters['rkey'], 'tapme');
  });

  testWidgets('FeedPage like and repost actions update the row', (
    tester,
  ) async {
    final post = _post('actions');
    final calls = <String>[];
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => PostPage(items: [post]),
        onLike: (did, rkey) async {
          calls.add('like:$did/$rkey');
          return _interaction(post);
        },
        onRepost: (did, rkey) async {
          calls.add('repost:$did/$rkey');
          return _interaction(post);
        },
      ),
    );

    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.favorite_border));
    await tester.pump();
    await tester.tap(find.byIcon(Icons.repeat));
    await tester.pump();

    expect(calls, [
      'like:did:plc:alice/actions',
      'repost:did:plc:alice/actions',
    ]);
    expect(find.byIcon(Icons.favorite), findsOneWidget);
  });

  testWidgets('FeedPage reply opens focused thread and updates root row', (
    tester,
  ) async {
    GoRouterState? threadState;
    final root = _post('root', replyCount: 3);
    final created = _post('created');
    final container = ProviderContainer.test(
      overrides: [
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(
            onListTimeline: ({cursor, limit}) async => PostPage(items: [root]),
            onCreate: ({required text, reply, images}) async => created,
          ),
        ),
      ],
    );
    addTearDown(container.dispose);
    final timelineSub = container.listen(timelineProvider, (_, _) {});
    addTearDown(timelineSub.close);
    final router = GoRouter(
      initialLocation: '/',
      routes: [
        GoRoute(path: '/', builder: (context, state) => const FeedPage()),
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
      UncontrolledProviderScope(
        container: container,
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

    final timeline = container.read(timelineProvider).value!;
    expect(timeline.items, hasLength(1));
    expect(timeline.items.single.uri, root.uri);
    expect(timeline.items.single.replyCount, 4);
    expect(timeline.items.single.viewerHasReplied, isTrue);
    expect(timeline.items.any((post) => post.uri == created.uri), isFalse);
  });

  testWidgets('FeedPage only exposes delete for own rows and removes row', (
    tester,
  ) async {
    final deleted = <String>[];
    final own = _post('own');
    final other = _post(
      'other',
      did: 'did:plc:bob',
      handle: 'bob.craftsky.social',
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async =>
                  PostPage(items: [own, other]),
              onDelete: (did, rkey) async => deleted.add('$did/$rkey'),
            ),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const FeedPage(),
          ),
        ),
      ),
    );

    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.more_horiz), findsNWidgets(2));

    await tester.tap(find.byIcon(Icons.more_horiz).last);
    await tester.pumpAndSettle();
    expect(find.text('Delete post'), findsNothing);
    await tester.tapAt(const Offset(10, 10));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.more_horiz).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();

    expect(deleted, ['did:plc:alice/own']);
    expect(find.text('timeline post own'), findsNothing);
    expect(find.text('timeline post other'), findsOneWidget);
  });

  testWidgets('FeedPage reports another user post through the report sheet', (
    tester,
  ) async {
    ReportSubmission? submitted;
    String? submittedTarget;
    final messenger = RecordingMessenger();

    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => PostPage(
          items: [
            _post('other', did: 'did:plc:bob', handle: 'bob.craftsky.social'),
          ],
        ),
        onReport: (did, rkey, submission) async {
          submittedTarget = '$did/$rkey';
          submitted = submission;
          return const ReportResult(reportId: 'report-1', status: 'accepted');
        },
      ),
      overrides: [
        authSessionProvider.overrideWith(
          () => SignedInAuthSession(did: 'did:plc:viewer'),
        ),
      ],
      messenger: messenger,
    );

    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.more_horiz));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Report post'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Spam'));
    await tester.pump();
    await tester.ensureVisible(find.widgetWithText(FilledButton, 'Submit'));
    await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
    await tester.pumpAndSettle();

    expect(submittedTarget, 'did:plc:bob/other');
    expect(submitted?.reasonType, 'spam');
    expect(find.text('Report post'), findsNothing);
    expect(
      messenger.calls,
      contains(('info', 'Thanks — your report was submitted.', null)),
    );
  });

  testWidgets('FeedPage compose creates top-level post and prepends it', (
    tester,
  ) async {
    PostReply? capturedReply;
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            PostPage(items: [_post('old')]),
        onCreate: ({required text, reply, images}) async {
          capturedReply = reply;
          return _post('new');
        },
      ),
    );

    await tester.pumpAndSettle();
    await tester.tap(find.text('New post'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'top-level');
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(capturedReply, isNull);
    expect(find.text('timeline post new'), findsOneWidget);
    expect(find.text('timeline post old'), findsOneWidget);
  });
}
