import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/recording_messenger.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('NotificationsPage renders its title', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            const _FakeNotificationRepository(NotificationPage(items: [])),
          ),
        ],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = AppLocalizations.of(
      tester.element(find.byType(NotificationsPage)),
    );
    expect(find.text(l10n.notificationsTitle), findsWidgets);
  });

  testWidgets('renders empty state and mixed rows', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            _FakeNotificationRepository(
              NotificationPage(
                items: [
                  _follow('follow1'),
                  _like('like1'),
                  _repost('repost1'),
                  _reply('reply1'),
                  _follow('fallback', displayName: null),
                ],
              ),
            ),
          ),
        ],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Alice followed you'), findsOneWidget);
    expect(find.text('Alice liked your post'), findsOneWidget);
    expect(find.text('Alice reposted your post'), findsOneWidget);
    expect(find.text('Alice replied to your post'), findsOneWidget);
    expect(find.text('viewer post'), findsNWidgets(3));
    expect(find.text('alice.craftsky.social followed you'), findsOneWidget);
  });

  testWidgets('preserves rows during load-more progress and retry', (
    tester,
  ) async {
    final nextPage = Completer<NotificationPage>();
    final repo = _QueueNotificationRepository([
      Future.value(NotificationPage(items: [_follow('follow1')], cursor: 'c1')),
      nextPage.future,
      Future.value(NotificationPage(items: [_follow('follow2')])),
    ]);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Alice followed you'), findsOneWidget);

    await tester.tap(find.text('Load more'));
    await tester.pump();
    expect(find.text('Alice followed you'), findsOneWidget);
    expect(find.byType(StitchProgressIndicator), findsOneWidget);

    nextPage.completeError(Exception('boom'));
    await tester.pumpAndSettle();
    expect(find.text('Alice followed you'), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);

    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(repo.calls.map((call) => call.cursor), [null, 'c1', 'c1']);
    expect(find.text('Alice followed you'), findsNWidgets(2));
  });

  testWidgets('row taps navigate to profile, subject thread, and focus', (
    tester,
  ) async {
    GoRouterState? profileState;
    final threadStates = <GoRouterState>[];
    final router = GoRouter(
      initialLocation: '/',
      routes: [
        GoRoute(
          path: '/',
          builder: (context, state) => const NotificationsPage(),
        ),
        GoRoute(
          path: '/profile/:handle',
          builder: (context, state) {
            profileState = state;
            return const Scaffold(body: Text('Profile route'));
          },
        ),
        GoRoute(
          path: '/posts/:did/:rkey',
          builder: (context, state) {
            threadStates.add(state);
            return const Scaffold(body: Text('Thread route'));
          },
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            _FakeNotificationRepository(
              NotificationPage(
                items: [
                  _follow('follow1'),
                  _like('like1'),
                  _repost('repost1'),
                  _reply('reply1'),
                  _reply('reply2', includeFocus: false),
                ],
              ),
            ),
          ),
        ],
        child: MaterialApp.router(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice followed you'));
    await tester.pumpAndSettle();
    expect(profileState?.pathParameters['handle'], 'alice.craftsky.social');
    router.go('/');
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice liked your post'));
    await tester.pumpAndSettle();
    expect(threadStates.last.pathParameters['did'], 'did:plc:viewer');
    expect(threadStates.last.pathParameters['rkey'], 'root');
    expect(threadStates.last.uri.queryParameters['focus'], isNull);
    router.go('/');
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice reposted your post'));
    await tester.pumpAndSettle();
    expect(threadStates.last.pathParameters['did'], 'did:plc:viewer');
    expect(threadStates.last.pathParameters['rkey'], 'root');
    router.go('/');
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice replied to your post').first);
    await tester.pumpAndSettle();
    expect(
      threadStates.last.uri.queryParameters['focus'],
      'at://did:plc:alice/social.craftsky.feed.post/reply1',
    );
    router.go('/');
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice replied to your post').last);
    await tester.pumpAndSettle();
    expect(threadStates.last.pathParameters['did'], 'did:plc:viewer');
    expect(threadStates.last.pathParameters['rkey'], 'root');
    expect(threadStates.last.uri.queryParameters['focus'], isNull);
  });

  testWidgets(
    'AT-009 generic and unknown rows are inert while tombstones warn',
    (tester) async {
      final messenger = RecordingMessenger();
      final generic = _generic(
        '00000000-0000-0000-0000-000000000001',
        type: 'everythingElse',
      );
      final unknown = _generic(
        '00000000-0000-0000-0000-000000000003',
        type: 'futureCategory',
      );
      final unavailable = _unavailable(
        '00000000-0000-0000-0000-000000000002',
      );

      await tester.pumpWidget(
        ProviderScope(
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Scaffold(
                body: Column(
                  children: [
                    NotificationRow(notification: generic),
                    NotificationRow(notification: unknown),
                    NotificationRow(notification: unavailable),
                  ],
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      final informationalTiles = tester.widgetList<ListTile>(
        find.ancestor(
          of: find.text('New activity'),
          matching: find.byType(ListTile),
        ),
      );
      expect(informationalTiles, hasLength(2));
      expect(informationalTiles.every((tile) => tile.onTap == null), isTrue);

      await tester.tap(find.text('New activity').first, warnIfMissed: false);
      await tester.tap(find.text('New activity').last, warnIfMissed: false);
      await tester.pump();
      expect(messenger.calls, isEmpty);

      await tester.tap(find.text('Activity unavailable'));
      await tester.pump();

      expect(messenger.calls, hasLength(1));
      expect(messenger.calls.single.$1, 'warning');
      expect(messenger.calls.single.$2, 'Activity unavailable');
    },
  );
}

class _TestApp extends StatelessWidget {
  const _TestApp({required this.home});

  final Widget home;

  @override
  Widget build(BuildContext context) => MaterialApp(
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: home,
  );
}

FollowNotification _follow(String rkey, {String? displayName = 'Alice'}) =>
    CraftskyNotification.fromMap({
          'uri': 'at://did:plc:alice/app.bsky.graph.follow/$rkey',
          'cid': 'bafy$rkey',
          'rkey': rkey,
          'type': 'follow',
          'actor': {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'displayName': ?displayName,
          },
          'createdAt': '2026-05-28T13:00:00Z',
          'indexedAt': '2026-05-28T13:00:01Z',
        })
        as FollowNotification;

LikeNotification _like(String rkey) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('like', rkey),
          'subjectPost': _post(),
        })
        as LikeNotification;

RepostNotification _repost(String rkey) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('repost', rkey),
          'subjectPost': _post(),
        })
        as RepostNotification;

ReplyNotification _reply(String rkey, {bool includeFocus = true}) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('reply', rkey),
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
          'subjectPost': _post(),
          if (includeFocus)
            'reply': {
              'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
              'cid': 'bafy$rkey',
              'rkey': rkey,
            },
        })
        as ReplyNotification;

GenericNotification _generic(String id, {required String type}) =>
    CraftskyNotification.fromMap({
          ..._baseNotification(type, 'generic'),
          'id': id,
        })
        as GenericNotification;

UnavailableNotification _unavailable(String id) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('like', 'unavailable'),
          'id': id,
          'actor': {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'available': false,
          },
        })
        as UnavailableNotification;

Map<String, dynamic> _baseNotification(String type, String rkey) => {
  'uri': 'at://did:plc:alice/social.craftsky.feed.$type/$rkey',
  'cid': 'bafy$rkey',
  'rkey': rkey,
  'type': type,
  'actor': {
    'did': 'did:plc:alice',
    'handle': 'alice.craftsky.social',
    'displayName': 'Alice',
  },
  'createdAt': '2026-05-28T13:00:00Z',
  'indexedAt': '2026-05-28T13:00:01Z',
};

Map<String, dynamic> _post() => {
  'uri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
  'cid': 'bafyroot',
  'rkey': 'root',
  'text': 'viewer post',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasReplied': false,
  'createdAt': '2026-05-28T12:00:00Z',
  'indexedAt': '2026-05-28T12:00:01Z',
  'author': {'did': 'did:plc:viewer', 'handle': 'viewer.craftsky.social'},
};

class _FakeNotificationRepository implements NotificationRepository {
  const _FakeNotificationRepository(this.page);

  final NotificationPage page;

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) async => page;
}

class _QueueNotificationRepository implements NotificationRepository {
  _QueueNotificationRepository(this.responses);

  final List<Future<NotificationPage>> responses;
  final calls = <_Call>[];

  @override
  Future<NotificationPage> list({String? cursor, int? limit}) {
    calls.add(_Call(cursor: cursor, limit: limit));
    return responses.removeAt(0);
  }
}

final class _Call {
  const _Call({this.cursor, this.limit});

  final String? cursor;
  final int? limit;
}
