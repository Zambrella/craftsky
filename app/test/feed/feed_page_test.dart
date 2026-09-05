import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';
import 'fakes/fake_post_repository.dart';

final class _FeedPinRegistryStorage implements SessionRegistryStorage {
  _FeedPinRegistryStorage()
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

Map<String, dynamic> _postMap({
  required String rkey,
  String? text,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  int replyCount = 3,
  bool viewerHasReplied = false,
}) => {
  'uri': 'at://$did/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': text ?? 'timeline post $rkey',
  'tags': <String>[],
  'likeCount': 1,
  'repostCount': 2,
  'replyCount': replyCount,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'viewerHasReplied': viewerHasReplied,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did, 'handle': handle},
};

Post _post(
  String rkey, {
  String? text,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  int replyCount = 3,
  bool viewerHasReplied = false,
}) => PostMapper.fromMap(
  _postMap(
    rkey: rkey,
    text: text,
    did: did,
    handle: handle,
    replyCount: replyCount,
    viewerHasReplied: viewerHasReplied,
  ),
);

TimelineItem _timelinePost(Post post, {String? itemKey}) => TimelineItem(
  itemKey: itemKey ?? 'post:${post.uri}',
  post: post,
);

TimelinePage _timelinePage(List<Post> posts, {String? cursor}) => TimelinePage(
  items: [for (final post in posts) _timelinePost(post)],
  cursor: cursor,
);

TimelineItem _repostItem({
  required String itemKey,
  required Post post,
  required String reposterDid,
  required String reposterHandle,
  required String reposterName,
}) => TimelineItem(
  itemKey: itemKey,
  post: post,
  reason: RepostReason(
    type: RepostReasonType.repost,
    by: PostAuthor(
      did: reposterDid,
      handle: reposterHandle,
      displayName: reposterName,
    ),
    uri:
        'at://$reposterDid/social.craftsky.feed.repost/${itemKey.split('/').last}',
    cid: 'bafy_${itemKey.split('/').last}',
    createdAt: DateTime.parse('2026-05-04T18:24:00.000Z'),
    indexedAt: DateTime.parse('2026-05-04T18:24:01.000Z'),
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
  TextDirection textDirection = TextDirection.ltr,
  double topPadding = 0,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: List.from([
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postRepositoryProvider.overrideWithValue(repo),
        ...overrides,
      ]),
      child: MessengerScope(
        messenger: messenger ?? RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Directionality(
              textDirection: textDirection,
              child: MediaQuery(
                data: MediaQuery.of(
                  context,
                ).copyWith(padding: EdgeInsets.only(top: topPadding)),
                child: const FeedPage(),
              ),
            ),
          ),
        ),
      ),
    ),
  );
}

void main() {
  setUpAll(initializeMappers);

  testWidgets('compact Feed exposes one extended New post action', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(600, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([_post('a')]),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(CraftskyFloatingActionButton), findsOneWidget);
    expect(find.text('New post'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(PostCard),
        matching: find.text('New post'),
      ),
      findsNothing,
    );
  });

  testWidgets('wide Feed has no New post floating action', (tester) async {
    tester.view.physicalSize = const Size(1000, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([_post('a')]),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(CraftskyFloatingActionButton), findsNothing);
    expect(find.text('New post'), findsNothing);
  });

  for (final initialPosts in <List<Post>>[
    const [],
    [_post('before')],
  ]) {
    testWidgets(
      'Feed pull-to-refresh reloads '
      '${initialPosts.isEmpty ? 'empty' : 'nonempty'} data',
      (tester) async {
        var calls = 0;
        await _pump(
          tester,
          FakePostRepository(
            onListTimeline: ({cursor, limit}) async {
              calls++;
              return calls == 1
                  ? _timelinePage(initialPosts)
                  : _timelinePage([_post('refreshed')]);
            },
          ),
        );
        await tester.pumpAndSettle();

        await tester.drag(
          find.byType(CustomScrollView),
          const Offset(0, 400),
        );
        await tester.pumpAndSettle();

        expect(calls, 2);
        expect(find.text('timeline post refreshed'), findsOneWidget);
      },
    );
  }

  testWidgets('Feed refresh indicator starts below its sliver app bar', (
    tester,
  ) async {
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            const TimelinePage(items: []),
      ),
      topPadding: 24,
    );
    await tester.pumpAndSettle();

    expect(
      tester.widget<RefreshIndicator>(find.byType(RefreshIndicator)).edgeOffset,
      24 + kToolbarHeight,
    );
  });

  testWidgets('Feed back-to-top is absent while loading, empty, and at top', (
    tester,
  ) async {
    final gate = Completer<TimelinePage>();
    await _pump(
      tester,
      FakePostRepository(onListTimeline: ({cursor, limit}) => gate.future),
    );
    expect(find.byTooltip('Back to top'), findsNothing);

    gate.complete(const TimelinePage(items: []));
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsNothing);

    await tester.pumpWidget(const SizedBox.shrink());
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([
          for (var i = 0; i < 12; i++) _post('top-$i'),
        ]),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byTooltip('Back to top'), findsNothing);
  });

  testWidgets('Feed back-to-top appears after 200 px and returns to top', (
    tester,
  ) async {
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([
          for (var i = 0; i < 16; i++) _post('scroll-$i'),
        ]),
      ),
    );
    await tester.pumpAndSettle();
    final scrollable = tester.state<ScrollableState>(
      find.descendant(
        of: find.byType(CustomScrollView),
        matching: find.byType(Scrollable),
      ),
    );

    await tester.drag(find.byType(CustomScrollView), const Offset(0, -250));
    await tester.pumpAndSettle();

    expect(scrollable.position.pixels, greaterThanOrEqualTo(200));
    expect(find.byTooltip('Back to top'), findsOneWidget);

    await tester.tap(find.byTooltip('Back to top'));
    await tester.pumpAndSettle();

    expect(scrollable.position.pixels, scrollable.position.minScrollExtent);
    expect(find.byTooltip('Back to top'), findsNothing);
  });

  testWidgets('Feed actions do not overlap in right-to-left layouts', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(600, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([
          for (var i = 0; i < 16; i++) _post('rtl-$i'),
        ]),
      ),
      textDirection: TextDirection.rtl,
    );
    await tester.pumpAndSettle();
    await tester.drag(find.byType(CustomScrollView), const Offset(0, -250));
    await tester.pumpAndSettle();

    final composeRect = tester.getRect(
      find.byType(CraftskyFloatingActionButton),
    );
    final topRect = tester.getRect(find.byTooltip('Back to top'));
    expect(composeRect.overlaps(topRect), isFalse);
  });

  testWidgets('Feed long cards toggle between Show more and Show less', (
    tester,
  ) async {
    final longText = List.filled(301, 'x').join();
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            _timelinePage([_post('long', text: longText)]),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Show more'), findsOneWidget);
    await tester.ensureVisible(find.text('Show more'));
    await tester.tap(find.text('Show more'));
    await tester.pump(const Duration(milliseconds: 350));
    expect(find.text('Show less'), findsOneWidget);

    await tester.ensureVisible(find.text('Show less'));
    await tester.tap(find.text('Show less'));
    await tester.pump(const Duration(milliseconds: 350));
    expect(find.text('Show more'), findsOneWidget);
  });

  testWidgets('FeedPage renders timeline loading state', (tester) async {
    final gate = Completer<TimelinePage>();
    await _pump(
      tester,
      FakePostRepository(onListTimeline: ({cursor, limit}) => gate.future),
    );

    expect(find.text('Feed'), findsOneWidget);
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.text('timeline post a'), findsNothing);

    gate.complete(const TimelinePage(items: []));
  });

  testWidgets('AT-002 pins an own top-level timeline post', (tester) async {
    final targets = <String>[];
    final messenger = RecordingMessenger();
    final post = _post('timeline-pin');
    final repository = FakePostRepository(
      onListTimeline: ({cursor, limit}) async => _timelinePage([post]),
      onProfilePins: () async => const ProfilePinState(),
      onPin: (did, rkey) async {
        targets.add('$did/$rkey');
        return ProfilePinState(standardPostUri: post.uri.value);
      },
    );
    await _pump(
      tester,
      repository,
      overrides: [
        authSessionProvider.overrideWith(
          () => SignedInAuthSession(did: 'did:plc:alice'),
        ),
        secureSessionRegistryStorageProvider.overrideWithValue(
          _FeedPinRegistryStorage(),
        ),
      ],
      messenger: messenger,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(CraftskyIconsBold.more));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Pin post'));
    await tester.pumpAndSettle();

    expect(targets, ['did:plc:alice/timeline-pin']);
    expect(messenger.calls, [('info', 'Post pinned', null)]);
  });

  testWidgets('FeedPage renders loaded timeline post cards', (tester) async {
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            _timelinePage([_post('a'), _post('b')]),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('timeline post a'), findsOneWidget);
    expect(find.text('timeline post b'), findsOneWidget);
    expect(find.textContaining('@alice.craftsky.social'), findsWidgets);
  });

  testWidgets('FeedPage renders duplicate repost items with attribution', (
    tester,
  ) async {
    final original = _post(
      'shared',
      did: 'did:plc:carol',
      handle: 'carol.craftsky.social',
    );
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => TimelinePage(
          items: [
            _repostItem(
              itemKey: 'repost:at://did:plc:bob/social.craftsky.feed.repost/r1',
              post: original,
              reposterDid: 'did:plc:bob',
              reposterHandle: 'bob.craftsky.social',
              reposterName: 'Bob',
            ),
            _repostItem(
              itemKey:
                  'repost:at://did:plc:dana/social.craftsky.feed.repost/r2',
              post: original,
              reposterDid: 'did:plc:dana',
              reposterHandle: 'dana.craftsky.social',
              reposterName: 'Dana',
            ),
          ],
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Reposted by Bob'), findsOneWidget);
    expect(find.text('Reposted by Dana'), findsOneWidget);
    expect(find.text('timeline post shared'), findsNWidgets(2));
  });

  testWidgets('FeedPage renders empty state without suggestions', (
    tester,
  ) async {
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            const TimelinePage(items: []),
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
          return _timelinePage([_post('a')]);
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
            return _timelinePage(
              [for (var i = 0; i < 12; i++) _post('page1-$i')],
              cursor: 'c1',
            );
          }
          nextCursor = cursor;
          return _timelinePage([_post('page2')]);
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
            return _timelinePage(
              [for (var i = 0; i < 12; i++) _post('page1-$i')],
              cursor: 'c1',
            );
          }
          nextCursors.add(cursor);
          if (!allowNextPage) throw Exception('next page failed');
          return _timelinePage([_post('page2')]);
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
    await tester.ensureVisible(find.text('Retry'));
    await tester.pumpAndSettle();
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
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async =>
                  _timelinePage([_post('tapme')]),
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
    await tester.pump(const Duration(milliseconds: 350));
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
        onListTimeline: ({cursor, limit}) async => _timelinePage([post]),
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
    await tester.tap(find.byIcon(CraftskyIconsBold.like));
    await tester.pump();
    await tester.tap(find.byIcon(CraftskyIconsBold.repost));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Repost'));
    await tester.pumpAndSettle();

    expect(calls, [
      'like:did:plc:alice/actions',
      'repost:did:plc:alice/actions',
    ]);
    expect(find.byIcon(CraftskyIcons.liked), findsOneWidget);
  });

  testWidgets('FeedPage quote action opens composer with quote target', (
    tester,
  ) async {
    final target = _post('quote-target');
    final created = _post('quote-created');
    final repo = FakePostRepository(
      onListTimeline: ({cursor, limit}) async => _timelinePage([target]),
      onCreate: ({required text, reply, images}) async => created,
    );
    await _pump(tester, repo);

    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(CraftskyIconsBold.repost));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Quote'));
    await tester.pumpAndSettle();

    expect(find.text('timeline post quote-target'), findsWidgets);

    await tester.enterText(find.byType(TextField).first, 'quote commentary');
    await tester.pump();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(repo.lastCreateQuote?.uri, target.uri);
    expect(repo.lastCreateQuote?.cid, target.cid);
  });

  testWidgets('FeedPage shows an error when liking fails', (tester) async {
    final post = _post('like-fails');
    final messenger = RecordingMessenger();
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async => _timelinePage([post]),
        onLike: (did, rkey) async => throw Exception('pds write failed'),
      ),
      messenger: messenger,
    );

    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(CraftskyIconsBold.like));
    await tester.pump();
    await tester.pump();

    expect(
      messenger.calls,
      contains(('error', "Couldn't update like.", null)),
    );
    expect(find.byIcon(CraftskyIconsBold.like), findsOneWidget);
  });

  testWidgets('FeedPage reply opens focused thread and updates root row', (
    tester,
  ) async {
    GoRouterState? threadState;
    final root = _post('root');
    final created = _post('created');
    final container = ProviderContainer.test(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(
            onListTimeline: ({cursor, limit}) async => _timelinePage([root]),
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
    await tester.tap(find.byIcon(CraftskyIconsBold.comment));
    await tester.pumpAndSettle();
    expect(find.text('Regular post'), findsNothing);
    expect(find.text('Project post'), findsNothing);
    await tester.enterText(find.byType(TextField), 'new comment');
    await tester.pump();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Comment'));
    await tester.pumpAndSettle();

    expect(find.text('Thread route'), findsOneWidget);
    expect(threadState?.pathParameters['did'], root.author.did);
    expect(threadState?.pathParameters['rkey'], root.rkey);
    expect(threadState?.uri.queryParameters['focus'], created.uri);
    expect(threadState?.extra, isA<Post>());
    expect((threadState!.extra! as Post).uri, created.uri);

    final timeline = container.read(timelineProvider).value!;
    expect(timeline.items, hasLength(1));
    expect(timeline.items.single.post.uri, root.uri);
    expect(timeline.items.single.post.replyCount, 4);
    expect(timeline.items.single.post.viewerHasReplied, isTrue);
    expect(timeline.items.any((item) => item.post.uri == created.uri), isFalse);
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
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          authSessionProvider.overrideWith(
            () => SignedInAuthSession(did: 'did:plc:alice'),
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async =>
                  _timelinePage([own, other]),
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
    expect(find.byIcon(CraftskyIconsBold.more), findsNWidgets(2));

    await tester.tap(find.byIcon(CraftskyIconsBold.more).last);
    await tester.pumpAndSettle();
    expect(find.text('Delete post'), findsNothing);
    await tester.tapAt(const Offset(10, 10));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(CraftskyIconsBold.more).first);
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
        onListTimeline: ({cursor, limit}) async => _timelinePage([
          _post('other', did: 'did:plc:bob', handle: 'bob.craftsky.social'),
        ]),
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
    await tester.tap(find.byIcon(CraftskyIconsBold.more));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Report post'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Spam'));
    await tester.pump();
    await tester.tap(find.widgetWithText(TextButton, 'Submit'));
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
    Object? capturedProject;
    await _pump(
      tester,
      FakePostRepository(
        onListTimeline: ({cursor, limit}) async =>
            _timelinePage([_post('old')]),
        onCreateWithFacets:
            ({required text, reply, project, images, facets}) async {
              capturedReply = reply;
              capturedProject = project;
              return _post('new');
            },
      ),
    );

    await tester.pumpAndSettle();
    await tester.tap(find.text('New post'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Regular post'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'top-level');
    await tester.pump();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Post'));
    await tester.pumpAndSettle();

    expect(capturedReply, isNull);
    expect(capturedProject, isNull);
    expect(find.text('timeline post new'), findsOneWidget);
    expect(find.text('timeline post old'), findsOneWidget);
  });
}
