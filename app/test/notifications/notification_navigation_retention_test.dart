import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  setUpAll(initializeMappers);

  for (final (target, reduced) in [
    ('avatar', false),
    ('name', false),
    ('card', false),
    ('avatar', true),
    ('name', true),
    ('card', true),
  ]) {
    testWidgets(
      'notification $target retains scroll and respects reduced motion '
      '($reduced)',
      (tester) async {
        final seen = Completer<void>();
        final router = await _pumpScene(tester, seen, reducedMotion: reduced);
        final scroll = find.descendant(
          of: find.byType(NotificationsPage),
          matching: find.byType(CustomScrollView),
        );
        await tester.drag(scroll, const Offset(0, -1000));
        await tester.pumpAndSettle();
        final position = tester
            .state<ScrollableState>(
              find
                  .descendant(of: scroll, matching: find.byType(Scrollable))
                  .first,
            )
            .position;
        final offset = position.pixels;
        expect(offset, greaterThan(500));
        final rows = find.byType(NotificationRow).evaluate().where((element) {
          final rect = tester.getRect(find.byWidget(element.widget));
          return rect.top > 80 && rect.bottom < 550;
        });
        final row = find.byWidget(rows.first.widget);
        if (target == 'avatar') {
          await tester.tap(
            find.descendant(of: row, matching: find.byType(ProfileAvatar)),
          );
        } else if (target == 'name') {
          final name = find.descendant(of: row, matching: find.text('Alice'));
          await tester.tap(name);
        } else {
          await tester.tapAt(
            tester.getRect(row).centerRight - const Offset(20, 0),
          );
        }
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 80));
        expect(
          find.byKey(const Key('profile-avatar-flight')),
          reduced ? findsNothing : findsOneWidget,
        );
        expect(
          find.byKey(const Key('profile-name-flight')),
          reduced ? findsNothing : findsOneWidget,
        );
        await tester.pumpAndSettle();
        seen.complete();
        await tester.pumpAndSettle();
        router.pop();
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 80));
        expect(
          find.byKey(const Key('profile-avatar-flight')),
          reduced ? findsNothing : findsOneWidget,
        );
        expect(
          find.byKey(const Key('profile-name-flight')),
          reduced ? findsNothing : findsOneWidget,
        );
        await tester.pumpAndSettle();
        expect(position.pixels, closeTo(offset, .1));
        expect(tester.takeException(), isNull);
      },
    );
  }

  for (final textScale in [1.0, 2.0]) {
    testWidgets(
      'returning inline name keeps its baseline at text scale $textScale',
      (
        tester,
      ) async {
        final seen = Completer<void>();
        final router = await _pumpScene(tester, seen, textScale: textScale);
        final row = find.byType(NotificationRow).first;
        final id = tester.widget<NotificationRow>(row).notification.id;
        final retainedRow = find.byWidgetPredicate(
          (widget) => widget is NotificationRow && widget.notification.id == id,
        );
        Finder title() => find.descendant(
          of: retainedRow,
          matching: find.byWidgetPredicate(
            (widget) => widget is Text && widget.textSpan != null,
          ),
        );
        final initialTitle = tester
            .getRect(title())
            .shift(-tester.getTopLeft(retainedRow));
        final initialHeight = tester.getSize(retainedRow).height;
        await tester.tap(
          find.descendant(of: row, matching: find.byType(ProfileAvatar)),
        );
        await tester.pumpAndSettle();
        seen.complete();
        await tester.pumpAndSettle();
        router.pop();
        await tester.pump();
        for (var frame = 0; frame < 10; frame++) {
          await tester.pump(const Duration(milliseconds: 30));
          expect(
            tester.getRect(title()).shift(-tester.getTopLeft(retainedRow)),
            initialTitle,
            reason: 'return frame $frame',
          );
          expect(
            tester.getSize(retainedRow).height,
            initialHeight,
            reason: 'return frame $frame must not reflow the row',
          );
        }
        await tester.pumpAndSettle();
        expect(
          tester.getRect(title()).shift(-tester.getTopLeft(retainedRow)),
          initialTitle,
        );
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('notification list stays at its offset after opening a card', (
    tester,
  ) async {
    final seen = Completer<void>();
    final router = await _pumpScene(tester, seen, postNotifications: true);
    final scroll = find.descendant(
      of: find.byType(NotificationsPage),
      matching: find.byType(CustomScrollView),
    );
    await tester.drag(scroll, const Offset(0, -1000));
    await tester.pumpAndSettle();
    final position = tester
        .state<ScrollableState>(
          find.descendant(of: scroll, matching: find.byType(Scrollable)).first,
        )
        .position;
    final offset = position.pixels;
    final row = find.byType(NotificationRow).evaluate().firstWhere((element) {
      final rect = tester.getRect(find.byWidget(element.widget));
      return rect.top > 80 && rect.bottom < 550;
    });
    await tester.tapAt(
      tester.getRect(find.byWidget(row.widget)).centerRight -
          const Offset(20, 0),
    );
    expect(router.state.uri.path, startsWith('/posts/'));
    await tester.pumpAndSettle();
    seen.complete();
    await tester.pumpAndSettle();
    router.pop();
    await tester.pumpAndSettle();
    expect(position.pixels, closeTo(offset, .1));
    expect(tester.takeException(), isNull);
  });
  testWidgets('paginated notification list retains its loaded rows on Back', (
    tester,
  ) async {
    final seen = Completer<void>();
    final router = await _pumpScene(tester, seen, paginated: true);
    final container = ProviderScope.containerOf(
      tester.element(find.byType(NotificationsPage)),
    );
    await container.read(notificationsProvider.notifier).loadMore();
    await tester.pumpAndSettle();
    final scroll = find.descendant(
      of: find.byType(NotificationsPage),
      matching: find.byType(CustomScrollView),
    );
    await tester.drag(scroll, const Offset(0, -4000));
    await tester.pumpAndSettle();
    final position = tester
        .state<ScrollableState>(
          find.descendant(of: scroll, matching: find.byType(Scrollable)).first,
        )
        .position;
    final offset = position.pixels;
    unawaited(router.push<void>('/detail'));
    await tester.pumpAndSettle();
    seen.complete();
    await tester.pumpAndSettle();
    router.pop();
    await tester.pumpAndSettle();
    expect(position.pixels, closeTo(offset, .1));
    expect(tester.takeException(), isNull);
  });
}

Future<GoRouter> _pumpScene(
  WidgetTester tester,
  Completer<void> seen, {
  bool paginated = false,
  bool postNotifications = false,
  bool reducedMotion = false,
  double textScale = 1,
}) async {
  final router = GoRouter(
    initialLocation: '/notifications',
    routes: [
      ShellRoute(
        builder: (_, _, child) => child,
        routes: [
          StatefulShellRoute.indexedStack(
            builder: (_, _, shell) => shell,
            branches: [
              StatefulShellBranch(
                routes: [
                  GoRoute(
                    path: '/notifications',
                    builder: (_, _) => const NotificationsPage(),
                  ),
                ],
              ),
            ],
          ),
          GoRoute(
            path: '/profiles/:did',
            builder: (context, state) => UserProfileRoute(
              did: Did.parse(state.pathParameters['did']!),
              $extra: state.extra as ProfileAvatarTransition?,
            ).build(context, state),
          ),
          GoRoute(
            path: '/posts/:did/:rkey',
            builder: (_, _) => const Scaffold(body: Text('Post detail')),
          ),
          GoRoute(
            path: '/detail',
            builder: (_, _) => const Scaffold(body: Text('Detail')),
          ),
        ],
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        notificationRepositoryProvider.overrideWithValue(
          _Notifications(
            paginated: paginated,
            postNotifications: postNotifications,
          ),
        ),
        notificationNewnessRepositoryProvider.overrideWithValue(
          _Newness(seen.future),
        ),
        profileRepositoryProvider.overrideWithValue(
          FakeProfileRepository(
            onFetch: (_) async => Profile(
              did: 'did:plc:alice',
              handle: 'alice.craftsky.social',
              displayName: 'Alice',
              crafts: const [],
            ),
          ),
        ),
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(
            onListByAuthor: (_, {cursor, limit}) async =>
                const PostPage(items: []),
            onListProjectsByAuthor: (_, {cursor, limit}) async =>
                const PostPage(items: []),
            onListCommentsByAuthor: (_, {cursor, limit}) async =>
                const PostPage(items: []),
          ),
        ),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            disableAnimations: reducedMotion,
            textScaler: TextScaler.linear(textScale),
          ),
          child: child!,
        ),
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    ),
  );
  await tester.pumpAndSettle();
  return router;
}

class _Notifications implements NotificationRepository {
  new({this.paginated = false, this.postNotifications = false});
  final bool postNotifications;
  final bool paginated;
  @override
  Future<NotificationPage> list({
    String? cursor,
    int? limit,
    NotificationCategory? category,
  }) async => NotificationPage(
    cursor: paginated && cursor == null ? 'page2' : null,
    items: [
      for (
        var i = paginated && cursor != null ? 20 : 0;
        i < (paginated ? (cursor == null ? 20 : 40) : 30);
        i++
      )
        CraftskyNotification.fromMap({
          'id': '00000000-0000-0000-0000-${i.toString().padLeft(12, '0')}',
          'uri': 'at://did:plc:alice/app.bsky.graph.follow/follow$i',
          'cid': 'bafyfollow$i',
          'rkey': 'follow$i',
          'type': postNotifications ? 'like' : 'follow',
          if (postNotifications)
            'subjectPost': {
              'uri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
              'cid': 'bafyroot',
              'rkey': 'root',
              'text': 'My original post',
              'tags': <String>[],
              'likeCount': 0,
              'repostCount': 0,
              'replyCount': 0,
              'viewerHasLiked': false,
              'viewerHasReposted': false,
              'viewerHasSaved': false,
              'sponsored': false,
              'createdAt': '2026-05-28T12:00:00Z',
              'indexedAt': '2026-05-28T12:00:01Z',
              'author': {
                'did': 'did:plc:viewer',
                'handle': 'viewer.craftsky.social',
              },
            },
          'actor': {
            'did': 'did:plc:alice',
            'handle': 'alice.craftsky.social',
            'displayName': 'Alice',
          },
          'createdAt': '2026-05-28T13:00:00Z',
          'indexedAt': '2026-05-28T13:00:01Z',
        }),
    ],
  );
}

class _Newness implements NotificationNewnessRepository {
  new(this.pending);
  final Future<void> pending;
  @override
  Future<int> count() async => 0;
  @override
  Future<void> markSeen() => pending;
}
