import 'dart:async';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';

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
    expect(find.text('Alice commented on your post'), findsOneWidget);
    expect(find.text('viewer post'), findsNWidgets(3));
    expect(find.text('alice.craftsky.social followed you'), findsOneWidget);
  });

  testWidgets('UT-016 uses post, comment, and reply language in rows', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(800, 1800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      _TestApp(
        home: Scaffold(
          body: ListView(
            children: [
              NotificationRow(notification: _like('like-post')),
              NotificationRow(
                notification: _like(
                  'like-comment',
                  subjectPost: _commentPost(),
                ),
              ),
              NotificationRow(
                notification: _like(
                  'like-reply',
                  subjectPost: _replyPost(),
                ),
              ),
              NotificationRow(notification: _repost('repost-post')),
              NotificationRow(
                notification: _repost(
                  'repost-comment',
                  subjectPost: _commentPost(),
                ),
              ),
              NotificationRow(
                notification: _repost(
                  'repost-reply',
                  subjectPost: _replyPost(),
                ),
              ),
              NotificationRow(notification: _reply('response-post')),
              NotificationRow(
                notification: _reply(
                  'response-comment',
                  subjectPost: _commentPost(),
                ),
              ),
              NotificationRow(
                notification: _reply(
                  'response-reply',
                  subjectPost: _replyPost(),
                ),
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    for (final text in [
      'Alice liked your post',
      'Alice liked your comment',
      'Alice liked your reply',
      'Alice reposted your post',
      'Alice reposted your comment',
      'Alice reposted your reply',
      'Alice commented on your post',
      'Alice replied to your comment',
      'Alice replied to your reply',
    ]) {
      expect(find.text(text), findsOneWidget, reason: text);
    }
  });

  testWidgets(
    'UT-018 rows show actor avatars, action icons, and relative time',
    (tester) async {
      tester.view.physicalSize = const Size(800, 1400);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        _TestApp(
          home: Scaffold(
            body: ListView(
              children: [
                NotificationRow(notification: _follow('follow')),
                NotificationRow(notification: _like('like')),
                NotificationRow(notification: _repost('repost')),
                NotificationRow(notification: _reply('reply')),
                NotificationRow(notification: _mention('mention')),
                NotificationRow(notification: _quote('quote')),
                NotificationRow(
                  notification: _generic(
                    '00000000-0000-0000-0000-000000000001',
                    type: 'everythingElse',
                  ),
                ),
              ],
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(ProfileAvatar), findsNWidgets(7));
      expect(find.text('Follow'), findsOneWidget);
      expect(
        tester
            .widget<ProfileAvatar>(find.byType(ProfileAvatar).first)
            .avatarUrl,
        'https://cdn.example/avatar/alice.jpg',
      );
      for (final icon in [
        Icons.person_add_alt_outlined,
        Icons.favorite_outline,
        Icons.repeat,
        Icons.chat_bubble_outline,
        Icons.alternate_email,
        Icons.format_quote,
        Icons.notifications_none,
      ]) {
        expect(find.byIcon(icon), findsOneWidget, reason: '$icon');
      }

      final actorText = find.byWidgetPredicate(
        (widget) =>
            widget is Text &&
            widget.textSpan?.toPlainText().startsWith('Alice ') == true,
      );
      expect(actorText, findsWidgets);
      final actorSpan =
          tester.widget<Text>(actorText.first).textSpan! as TextSpan;
      final boldActor = actorSpan.children!.whereType<TextSpan>().firstWhere(
        (span) => span.text == 'Alice',
      );
      expect(boldActor.style?.fontWeight, FontWeight.bold);
      expect(
        tester.getTopLeft(find.byType(ProfileAvatar).first).dy,
        lessThan(tester.getTopLeft(actorText.first).dy),
      );

      final createdAt = DateTime.parse('2026-05-28T13:00:00Z');
      final elapsedDays = DateTime.now().difference(createdAt).inDays;
      expect(find.text('${elapsedDays}d'), findsNWidgets(7));
      expect(
        tester
            .widgetList<Tooltip>(find.byType(Tooltip))
            .every(
              (tooltip) => tooltip.message?.contains('2026') ?? false,
            ),
        isTrue,
      );
    },
  );

  testWidgets('UT-021 derives an actor avatar URL from an older CID response', (
    tester,
  ) async {
    await tester.pumpWidget(
      _TestApp(
        home: Scaffold(
          body: NotificationRow(
            notification: _follow('legacy-avatar', includeAvatarUrl: false),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      tester.widget<ProfileAvatar>(find.byType(ProfileAvatar)).avatarUrl,
      'https://cdn.bsky.app/img/avatar/plain/did:plc:alice/bafyavatar@jpeg',
    );
    expect(
      _follow(
        'legacy-devmedia-avatar',
        includeAvatarUrl: false,
        avatarCid: 'devmedia:alice-avatar',
      ).actor.displayAvatarUrl,
      isNull,
    );
  });

  testWidgets('UT-023 follow notification toggles Follow and Unfollow', (
    tester,
  ) async {
    final calls = <String>[];
    Profile result({required bool viewerIsFollowing}) => Profile(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
      crafts: const [],
      viewerIsFollowing: viewerIsFollowing,
    );
    final repository = FakeProfileRepository(
      onFollow: (handleOrDid) async {
        calls.add('follow:$handleOrDid');
        return result(viewerIsFollowing: true);
      },
      onUnfollow: (handleOrDid) async {
        calls.add('unfollow:$handleOrDid');
        return result(viewerIsFollowing: false);
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
        ],
        child: _TestApp(
          home: Scaffold(
            body: NotificationRow(
              notification: _follow(
                'follow-action',
                viewerIsFollowing: true,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Unfollow'), findsOneWidget);
    expect(find.text('Follow'), findsNothing);

    await tester.tap(find.text('Unfollow'));
    await tester.pumpAndSettle();
    expect(calls, ['unfollow:did:plc:alice']);
    expect(find.text('Follow'), findsOneWidget);

    await tester.tap(find.text('Follow'));
    await tester.pumpAndSettle();
    expect(calls, [
      'unfollow:did:plc:alice',
      'follow:did:plc:alice',
    ]);
    expect(find.text('Unfollow'), findsOneWidget);
  });

  testWidgets('UT-023 follow notification rolls back a failed mutation', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final repository = FakeProfileRepository(
      onFollow: (_) => Future<Profile>.error(Exception('follow failed')),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
        ],
        child: _TestApp(
          home: MessengerScope(
            messenger: messenger,
            child: Scaffold(
              body: NotificationRow(
                notification: _follow('failed-follow-action'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Follow'));
    await tester.pumpAndSettle();

    expect(find.text('Follow'), findsOneWidget);
    expect(messenger.calls, hasLength(1));
    expect(messenger.calls.single.$1, 'error');
    expect(messenger.calls.single.$2, 'Could not update follow state.');
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
    tester.view.physicalSize = const Size(800, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
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
          theme: AppTheme.lightThemeData,
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

    await tester.tap(find.text('Alice commented on your post').first);
    await tester.pumpAndSettle();
    expect(
      threadStates.last.uri.queryParameters['focus'],
      'at://did:plc:alice/social.craftsky.feed.post/reply1',
    );
    router.go('/');
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice commented on your post').last);
    await tester.pumpAndSettle();
    expect(threadStates.last.pathParameters['did'], 'did:plc:viewer');
    expect(threadStates.last.pathParameters['rkey'], 'root');
    expect(threadStates.last.uri.queryParameters['focus'], isNull);
  });

  testWidgets('BUG-002 liked comment opens its root thread with focus', (
    tester,
  ) async {
    GoRouterState? threadState;
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: NotificationRow(
              notification: _like(
                'like-comment',
                subjectPost: _commentPost(),
              ),
            ),
          ),
        ),
        GoRoute(
          path: '/posts/:did/:rkey',
          builder: (_, state) {
            threadState = state;
            return const Scaffold(body: Text('Thread route'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice liked your comment'));
    await tester.pumpAndSettle();

    expect(threadState?.pathParameters['did'], 'did:plc:root-author');
    expect(threadState?.pathParameters['rkey'], 'root');
    expect(
      threadState?.uri.queryParameters['focus'],
      'at://did:plc:viewer/social.craftsky.feed.post/comment',
    );
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
              theme: AppTheme.lightThemeData,
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

      final genericText = find.byWidgetPredicate(
        (widget) =>
            widget is Text &&
            widget.textSpan?.toPlainText().contains('New activity') == true,
      );
      final informationalRows = tester.widgetList<InkWell>(
        find.ancestor(
          of: genericText,
          matching: find.byType(InkWell),
        ),
      );
      expect(informationalRows, hasLength(2));
      expect(informationalRows.every((row) => row.onTap == null), isTrue);

      await tester.tap(genericText.first, warnIfMissed: false);
      await tester.tap(genericText.last, warnIfMissed: false);
      await tester.pump();
      expect(messenger.calls, isEmpty);

      final unavailableText = find.byWidgetPredicate(
        (widget) =>
            widget is Text &&
            widget.textSpan?.toPlainText().contains('Activity unavailable') ==
                true,
      );
      await tester.tap(unavailableText);
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
  Widget build(BuildContext context) => ProviderScope(
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: home,
    ),
  );
}

FollowNotification _follow(
  String rkey, {
  String? displayName = 'Alice',
  bool includeAvatarUrl = true,
  String avatarCid = 'bafyavatar',
  bool viewerIsFollowing = false,
}) =>
    CraftskyNotification.fromMap({
          'uri': 'at://did:plc:alice/app.bsky.graph.follow/$rkey',
          'cid': 'bafy$rkey',
          'rkey': rkey,
          'type': 'follow',
          'actor': {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'displayName': ?displayName,
            if (includeAvatarUrl)
              'avatar': 'https://cdn.example/avatar/alice.jpg',
            'avatarCid': avatarCid,
            'viewerIsFollowing': viewerIsFollowing,
          },
          'createdAt': '2026-05-28T13:00:00Z',
          'indexedAt': '2026-05-28T13:00:01Z',
        })
        as FollowNotification;

LikeNotification _like(
  String rkey, {
  Map<String, dynamic>? subjectPost,
}) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('like', rkey),
          'subjectPost': subjectPost ?? _post(),
        })
        as LikeNotification;

RepostNotification _repost(
  String rkey, {
  Map<String, dynamic>? subjectPost,
}) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('repost', rkey),
          'subjectPost': subjectPost ?? _post(),
        })
        as RepostNotification;

ReplyNotification _reply(
  String rkey, {
  bool includeFocus = true,
  Map<String, dynamic>? subjectPost,
}) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('reply', rkey),
          'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
          'subjectPost': subjectPost ?? _post(),
          if (includeFocus)
            'reply': {
              'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
              'cid': 'bafy$rkey',
              'rkey': rkey,
            },
        })
        as ReplyNotification;

MentionNotification _mention(String rkey) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('mention', rkey),
          'subjectPost': _post(),
        })
        as MentionNotification;

QuoteNotification _quote(String rkey) =>
    CraftskyNotification.fromMap({
          ..._baseNotification('quote', rkey),
          'subjectPost': _post(),
        })
        as QuoteNotification;

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
    'avatar': 'https://cdn.example/avatar/alice.jpg',
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

Map<String, dynamic> _commentPost() => {
  ..._post(),
  'uri': 'at://did:plc:viewer/social.craftsky.feed.post/comment',
  'cid': 'bafycomment',
  'rkey': 'comment',
  'text': 'viewer comment',
  'reply': {
    'root': {
      'uri': 'at://did:plc:root-author/social.craftsky.feed.post/root',
      'cid': 'bafyroot',
    },
    'parent': {
      'uri': 'at://did:plc:root-author/social.craftsky.feed.post/root',
      'cid': 'bafyroot',
    },
  },
};

Map<String, dynamic> _replyPost() => {
  ..._commentPost(),
  'uri': 'at://did:plc:viewer/social.craftsky.feed.post/reply',
  'cid': 'bafyreply',
  'rkey': 'reply',
  'text': 'viewer reply',
  'reply': {
    'root': {
      'uri': 'at://did:plc:root-author/social.craftsky.feed.post/root',
      'cid': 'bafyroot',
    },
    'parent': {
      'uri': 'at://did:plc:viewer/social.craftsky.feed.post/comment',
      'cid': 'bafycomment',
    },
  },
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
