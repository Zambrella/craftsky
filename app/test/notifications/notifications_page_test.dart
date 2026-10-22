import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/moderation/models/account_moderation.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/services/notification_destination_inference.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/notifications/widgets/notification_row.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/shared/widgets/sort_menu_button.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/deterministic_pump.dart';

void main() {
  setUpAll(initializeMappers);

  for (final systemNotification in [false, true]) {
    testWidgets(
      '${systemNotification ? 'system' : 'in-app'} reply notification '
      'opens root thread with the reply expanded and highlighted',
      (tester) async {
        final notification = _reply('new-reply', subjectPost: _commentPost());
        final comment = notification.subjectPost.copyWith(replyCount: 1);
        final rootUri = comment.reply!.root.uri;
        final replyUri = notification.reply!.uri;
        final root = PostMapper.fromMap({
          ..._post(),
          'uri': rootUri.toString(),
          'author': {
            'did': 'did:plc:root-author',
            'handle': 'root.craftsky.social',
          },
        });
        final reply = PostMapper.fromMap({
          ..._post(),
          'uri': replyUri.toString(),
          'rkey': 'new-reply',
          'text': 'The notified reply',
          'reply': {
            'root': {'uri': rootUri.toString(), 'cid': root.cid.toString()},
            'parent': {
              'uri': comment.uri.toString(),
              'cid': comment.cid.toString(),
            },
          },
        });
        final repository = FakePostRepository(
          onCommentSection: (did, rkey, {cursor, sort, focus, limit}) async {
            expect(did.toString(), 'did:plc:root-author');
            expect(rkey.toString(), 'root');
            expect(focus, replyUri);
            return PostCommentSection(
              post: root,
              sort: sort!,
              focus: FocusContext(
                uri: replyUri.toString(),
                status: FocusStatus.included,
                kind: FocusKind.reply,
                commentUri: comment.uri.toString(),
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
        late GoRouter router;
        router = GoRouter(
          routes: [
            GoRoute(
              path: '/',
              builder: (context, _) => Scaffold(
                body: systemNotification
                    ? TextButton(
                        onPressed: () => navigateToNotificationOutcome(
                          context,
                          router,
                          NotificationDestinationInference.forFacts(
                            NotificationOpenAttempt.fromProviderData({
                              'payloadVersion': '1',
                              'type': 'reply',
                              'accountSubscriptionId': 'binding',
                              'subjectUri': comment.uri.toString(),
                              'rootUri': rootUri.toString(),
                              'sourceUri': replyUri.toString(),
                            }).facts,
                          ),
                        ),
                        child: const Text('Open system notification'),
                      )
                    : NotificationRow(notification: notification),
              ),
            ),
            GoRoute(
              path: '/posts/:did/:rkey',
              builder: (context, state) => FormFactorWidget(
                child: PostThreadRoute(
                  did: state.pathParameters['did']!,
                  rkey: state.pathParameters['rkey']!,
                  focus: state.uri.queryParameters['focus'],
                  $extra: state.extra as Post?,
                ).build(context, state),
              ),
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [postRepositoryProvider.overrideWithValue(repository)],
            child: MaterialApp.router(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              routerConfig: router,
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.tap(
          find.text(
            systemNotification
                ? 'Open system notification'
                : 'Alice replied to your comment',
          ),
        );
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 400));
        await tester.pump(const Duration(milliseconds: 400));
        expect(find.text('The notified reply'), findsOneWidget);
        final focused = tester.widget<PostCard>(
          find.byKey(const ValueKey('focused-comment-target')),
        );
        expect(focused.post.uri, replyUri);
        expect(focused.isHighlighted, isTrue);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(seconds: 3));
      },
    );
  }

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

  testWidgets(
    'selector applies categories, preserves selection on refresh, '
    'and clears from empty state',
    (tester) async {
      final repo = _QueueNotificationRepository([
        Future.value(NotificationPage(items: [_follow('all')])),
        Future.value(const NotificationPage(items: [])),
        Future.value(const NotificationPage(items: [])),
        Future.value(NotificationPage(items: [_follow('all-again')])),
      ]);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
          child: const _TestApp(home: NotificationsPage()),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('All categories'), findsOneWidget);
      await tester.tap(find.text('All categories'));
      await tester.pumpAndSettle();
      expect(find.byType(BottomSheet), findsOneWidget);
      await tester.tap(find.text('Mentions'));
      await tester.pumpAndSettle();
      expect(find.byType(BottomSheet), findsNothing);
      expect(
        find.text('No notifications in this category yet.'),
        findsOneWidget,
      );
      expect(find.text('Alice followed you'), findsNothing);
      await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
      await tester.pumpAndSettle();
      expect(repo.calls.last.category, NotificationCategory.mention);
      await tester.tap(find.text('Show all notifications'));
      await tester.pumpAndSettle();
      expect(repo.calls.map((call) => call.category), [
        null,
        NotificationCategory.mention,
        NotificationCategory.mention,
        null,
      ]);
      expect(find.text('Alice followed you'), findsOneWidget);
    },
  );

  testWidgets('dismissing the category sheet preserves All', (tester) async {
    final repo = _QueueNotificationRepository([
      Future.value(const NotificationPage(items: [])),
    ]);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('All categories'));
    await tester.pumpAndSettle();
    Navigator.of(tester.element(find.byType(BottomSheet))).pop();
    await tester.pumpAndSettle();
    expect(repo.calls, hasLength(1));
    expect(find.text('All categories'), findsOneWidget);
  });

  testWidgets('changing category hides old rows while the new page loads', (
    tester,
  ) async {
    final pending = Completer<NotificationPage>();
    final repo = _QueueNotificationRepository([
      Future.value(NotificationPage(items: [_follow('all')])),
      pending.future,
    ]);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('All categories'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Mentions'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('Alice followed you'), findsNothing);
    expect(find.byType(CraftskySkeletonSliverList), findsOneWidget);
    pending.complete(const NotificationPage(items: []));
    await tester.pumpAndSettle();
    expect(find.text('No notifications in this category yet.'), findsOneWidget);
  });

  testWidgets(
    'AppBar uses the shared filter menu and scrolls all categories on a phone',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
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
      final appBar = tester.widget<SliverAppBar>(find.byType(SliverAppBar));
      expect(appBar.bottom, isNull);
      expect(find.byTooltip('Notification settings'), findsNothing);
      expect(
        find.byType(SortMenuButton<NotificationCategory?>),
        findsOneWidget,
      );
      await tester.tap(find.text('All categories'));
      await tester.pumpAndSettle();
      expect(find.byType(BottomSheet), findsOneWidget);
      expect(find.byIcon(CraftskyIcons.unselectedOption), findsNothing);
      expect(find.byIcon(CraftskyIcons.selectedOption), findsNothing);
      expect(find.byIcon(CraftskyIcons.notifications), findsWidgets);
      expect(find.byIcon(CraftskyIcons.like), findsOneWidget);
      expect(find.byIcon(CraftskyIcons.mention), findsOneWidget);
      expect(find.byIcon(CraftskyIcons.check), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.ensureVisible(find.text('Moderation'));
      await tester.tap(find.text('Moderation'));
      await tester.pumpAndSettle();
      expect(find.byType(BottomSheet), findsNothing);
      expect(find.text('Moderation'), findsNWidgets(2));
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('initial loading uses activity row skeletons', (tester) async {
    final page = Completer<NotificationPage>();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            _QueueNotificationRepository([page.future]),
          ),
        ],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pump();

    expect(find.byType(CraftskySkeletonSliverList), findsOneWidget);
    expect(find.byType(ActivityRowSkeleton), findsNWidgets(6));
    expect(find.byType(StitchProgressIndicator), findsNothing);
  });

  testWidgets('refresh indicator starts below the sliver app bar', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          notificationRepositoryProvider.overrideWithValue(
            const _FakeNotificationRepository(NotificationPage(items: [])),
          ),
        ],
        child: const _TestApp(topPadding: 24, home: NotificationsPage()),
      ),
    );
    await pumpUntilFound(
      tester,
      find.byType(YarnRefreshScrollView),
      description: 'the notifications refresh control',
    );

    expect(
      tester
          .widget<YarnRefreshScrollView>(find.byType(YarnRefreshScrollView))
          .headerSliverCount,
      1,
    );
  });

  testWidgets('pull to refresh reloads an empty notifications page', (
    tester,
  ) async {
    final repo = _QueueNotificationRepository([
      Future.value(const NotificationPage(items: [])),
      Future.value(NotificationPage(items: [_follow('refreshed')])),
    ]);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [notificationRepositoryProvider.overrideWithValue(repo)],
        child: const _TestApp(home: NotificationsPage()),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('No notifications yet.'), findsOneWidget);
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pumpAndSettle();

    expect(repo.calls.map((call) => call.cursor), [null, null]);
    expect(find.text('Alice followed you'), findsOneWidget);
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
    expect(find.text('viewer post'), findsNWidgets(2));
    expect(find.text('new reply text'), findsOneWidget);
    expect(find.text('alice.craftsky.social followed you'), findsOneWidget);
    final customisedAvatars = tester
        .widgetList<ProfileAvatar>(find.byType(ProfileAvatar))
        .where((avatar) => avatar.customisation.colour == 'rose')
        .toList();
    expect(customisedAvatars, isNotEmpty);
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
                notification: _like('like-reply', subjectPost: _replyPost()),
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
                NotificationRow(notification: _instagramMatch()),
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

      expect(find.byType(ProfileAvatar), findsNWidgets(8));
      expect(find.text('Follow'), findsNWidgets(2));
      expect(
        find.text('You found Alice through your Instagram following'),
        findsOneWidget,
      );
      expect(
        tester
            .widget<ProfileAvatar>(find.byType(ProfileAvatar).first)
            .avatarUrl,
        'https://cdn.example/avatar/alice.jpg',
      );
      for (final icon in [
        CraftskyIcons.follow,
        CraftskyIcons.like,
        CraftskyIcons.repost,
        CraftskyIcons.comment,
        CraftskyIcons.mention,
        CraftskyIcons.quote,
        CraftskyIcons.findPeople,
        CraftskyIcons.notification,
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
      expect(boldActor.style?.fontWeight, FontWeight.w600);
      expect(
        tester.getTopLeft(find.byType(ProfileAvatar).first).dy,
        lessThan(tester.getTopLeft(actorText.first).dy),
      );

      final createdAt = DateTime.parse('2026-05-28T13:00:00Z');
      final elapsedDays = DateTime.now().difference(createdAt).inDays;
      expect(find.text('${elapsedDays}d'), findsNWidgets(8));
      expect(
        tester
            .widgetList<Tooltip>(find.byType(Tooltip))
            .every((tooltip) => tooltip.message?.contains('2026') ?? false),
        isTrue,
      );
    },
  );

  for (final scale in [1.0, 2.0]) {
    testWidgets('notification uses the unified identity at text scale $scale', (
      tester,
    ) async {
      await tester.pumpWidget(
        _TestApp(
          home: Scaffold(
            body: Center(
              child: MediaQuery(
                data: MediaQueryData(textScaler: TextScaler.linear(scale)),
                child: SizedBox(
                  width: 360,
                  child: NotificationRow(
                    notification: _follow(
                      'unified-identity',
                      crafts: const ['knitting', 'crochet', 'sewing'],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CompactProfileIdentity), findsOneWidget);
      final identity = tester.widget<CompactProfileIdentity>(
        find.byType(CompactProfileIdentity),
      );
      expect(identity.crafts, ['knitting', 'crochet', 'sewing']);
      final name = tester.getRect(find.text('Alice'));
      final handle = tester.getRect(find.text('@alice.craftsky.social'));
      final avatar = tester.getRect(find.byType(ProfileAvatar));
      expect(avatar.right, lessThan(name.left));
      expect(name.left, handle.left);
      expect(handle.top, greaterThanOrEqualTo(name.bottom));
      expect(
        tester.getTopLeft(find.text('Alice followed you')).dy,
        greaterThanOrEqualTo(handle.bottom),
      );
      expect(
        tester.widget<Text>(find.text('@alice.craftsky.social')).style?.color,
        AppTheme.lightThemeData.colorScheme.outline,
      );
      if (scale == 1) expect(find.byType(CraftIcon), findsNWidgets(3));
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('notification identity handle opens the actor profile', (
    tester,
  ) async {
    GoRouterState? openedProfile;
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: NotificationRow(notification: _like('identity-tap')),
          ),
        ),
        GoRoute(
          path: '/profiles/:did',
          builder: (_, state) {
            openedProfile = state;
            return const Scaffold(body: Text('Actor profile'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('@alice.craftsky.social'));
    await tester.pumpAndSettle();
    expect(openedProfile?.pathParameters['did'], 'did:plc:alice');
  });

  testWidgets(
    'unavailable notifications do not show a profile handle or crafts',
    (
      tester,
    ) async {
      await tester.pumpWidget(
        _TestApp(
          home: Scaffold(
            body: NotificationRow(
              notification: _unavailable('unavailable-identity'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CompactProfileIdentity), findsNothing);
      expect(find.byType(CraftIcon), findsNothing);
      expect(find.text('@alice.craftsky.social'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('AT-011 post-bearing rows render subjects with PostSummary', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(800, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      _TestApp(
        home: Scaffold(
          body: ListView(
            children: [
              NotificationRow(notification: _like('like-summary')),
              NotificationRow(notification: _repost('repost-summary')),
              NotificationRow(notification: _reply('reply-summary')),
              NotificationRow(notification: _mention('mention-summary')),
              NotificationRow(notification: _quote('quote-summary')),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(PostSummary), findsNWidgets(5));
    expect(find.text('viewer post'), findsNWidgets(4));
    expect(find.text('new reply text'), findsOneWidget);
    expect(find.byIcon(CraftskyIcons.saved), findsNothing);
    expect(find.byIcon(CraftskyIconsBold.unsaved), findsNothing);
  });

  testWidgets(
    'text previews show project titles and quote context without media',
    (tester) async {
      tester.view.physicalSize = const Size(1000, 1600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final quote = CraftskyNotification.fromMap({
        ..._baseNotification('quote', 'quote-preview'),
        'subjectPost': _post(),
        'preview': {
          'text': 'Look at this stitch',
          'projectTitle': 'My sweater',
          'role': 'post',
        },
        'quotedPreview': {
          'text': 'Original comment',
          'projectTitle': 'Winter scarf',
          'role': 'comment',
        },
      });
      final mention = CraftskyNotification.fromMap({
        ..._baseNotification('mention', 'mention-preview'),
        'subjectPost': {
          ..._post(),
          'reply': {
            'root': {
              'uri': 'at://did:plc:viewer/social.craftsky.feed.post/root',
              'cid': 'rootcid',
            },
            'parent': {
              'uri': 'at://did:plc:viewer/social.craftsky.feed.post/parent',
              'cid': 'parentcid',
            },
          },
        },
        'preview': {'text': 'What do you think?', 'role': 'reply'},
      });
      await tester.pumpWidget(
        _TestApp(
          home: Scaffold(
            body: Column(
              children: [
                NotificationRow(notification: quote),
                NotificationRow(notification: mention),
                NotificationRow(notification: _reply('reply-preview')),
              ],
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Alice quoted your comment'), findsOneWidget);
      expect(find.text('Alice mentioned you in a reply'), findsOneWidget);
      for (final text in [
        'Look at this stitch',
        'My sweater',
        'Original comment',
        'Winter scarf',
        'What do you think?',
        'new reply text',
      ]) {
        expect(find.text(text), findsOneWidget);
      }
      expect(find.text('viewer post'), findsNothing);
      for (final summary in tester.widgetList<PostSummary>(
        find.byType(PostSummary),
      )) {
        expect(summary.data.image, isNull);
        expect(summary.data.external, isNull);
      }
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
      onFetch: (_) async => result(viewerIsFollowing: true),
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
        overrides: [profileRepositoryProvider.overrideWithValue(repository)],
        child: _TestApp(
          home: Scaffold(
            body: NotificationRow(
              notification: _follow('follow-action', viewerIsFollowing: true),
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
    expect(calls, ['unfollow:did:plc:alice', 'follow:did:plc:alice']);
    expect(find.text('Unfollow'), findsOneWidget);
    await tester.pump(const Duration(seconds: 31));
  });

  testWidgets('notification follows profile mutations in both directions', (
    tester,
  ) async {
    var profile = Profile(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      crafts: const [],
    );
    final repository = FakeProfileRepository(
      onFetch: (_) async => profile,
      onFollow: (_) async =>
          profile = profile.copyWith(viewerIsFollowing: true),
      onUnfollow: (_) async =>
          profile = profile.copyWith(viewerIsFollowing: false),
    );
    final container = ProviderContainer.test(
      overrides: [profileRepositoryProvider.overrideWithValue(repository)],
    );
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: _TestApp(
          home: Scaffold(
            body: NotificationRow(notification: _follow('profile-follow')),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Follow'), findsOneWidget);

    // Use the profile page mutation without tapping the notification row.
    for (final following in [true, false]) {
      final current = await container.read(
        userProfileProvider(profile.did).future,
      );
      await container
          .read(toggleFollowProfileProvider.notifier)
          .toggle(
            cacheKey: profile.did,
            profile: current,
          );
      await tester.pumpAndSettle();
      expect(find.text(following ? 'Unfollow' : 'Follow'), findsOneWidget);
    }
    await tester.pump(const Duration(seconds: 31));
  });

  for (final delayedLookup in [false, true]) {
    testWidgets(
      'follow notification accepts a delayed follow response '
      '(delayed lookup: $delayedLookup)',
      (tester) async {
        final messenger = RecordingMessenger();
        final pending = Completer<Profile>();
        final lookup = Completer<Profile>();
        final profile = Profile(
          did: 'did:plc:alice',
          handle: 'alice.craftsky.social',
          crafts: const [],
        );
        final repository = FakeProfileRepository(
          onFetch: (_) async => delayedLookup ? await lookup.future : profile,
          onFollow: (_) => pending.future,
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
                    notification: _follow('delayed-follow'),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();

        await tester.tap(find.text('Follow'));
        await tester.pump();
        expect(find.byType(StitchProgressIndicator), findsOneWidget);
        final button = tester.widget<ButtonStyleButton>(
          find.ancestor(
            of: find.text('Follow'),
            matching: find.byWidgetPredicate(
              (widget) => widget is ButtonStyleButton,
            ),
          ),
        );
        expect(button.onPressed, isNull);
        if (delayedLookup) {
          expect(repository.followOperationKeys, isEmpty);
          lookup.complete(profile);
          await pumpUntil(
            tester,
            () => repository.followOperationKeys.isNotEmpty,
            description: 'follow request after lookup',
          );
        }
        expect(repository.followOperationKeys, hasLength(1));

        pending.complete(profile.copyWith(viewerIsFollowing: true));
        await tester.pumpAndSettle();

        expect(messenger.calls, isEmpty);
        expect(find.text('Unfollow'), findsOneWidget);
        expect(find.byType(StitchProgressIndicator), findsNothing);
        await tester.pump(const Duration(seconds: 31));
      },
    );
  }

  testWidgets('UT-023 follow notification rolls back a failed mutation', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final repository = FakeProfileRepository(
      onFetch: (_) async => Profile(
        did: 'did:plc:alice',
        handle: 'alice.craftsky.social',
        crafts: const [],
      ),
      onFollow: (_) => Future<Profile>.error(Exception('follow failed')),
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [profileRepositoryProvider.overrideWithValue(repository)],
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

  testWidgets('AT-006 loaded notification disappears when actor mute starts', (
    tester,
  ) async {
    final account = AccountKey('did:plc:viewer');
    final completer = Completer<ProfileRelationship>();
    final relationshipRepository = FakeProfileRepository(
      onMute: (_) => completer.future,
    );
    final notification = CraftskyNotification.fromMap({
      ..._baseNotification('follow', 'pending-mute'),
      'actor': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
        'displayName': 'Alice',
        'muted': false,
        'blocking': false,
        'blockedBy': false,
      },
    });
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          accountRelationshipRepositoryProvider(
            account,
          ).overrideWith((ref) async => relationshipRepository),
        ],
        child: _TestApp(
          home: Scaffold(
            body: NotificationRow(
              notification: notification,
              owner: AccountSessionLease(
                account: account,
                sessionGeneration: 1,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final container = ProviderScope.containerOf(
      tester.element(find.byType(NotificationRow)),
    );
    unawaited(
      container
          .read(
            profileRelationshipProvider(
              account,
              Did.parse('did:plc:alice'),
            ).notifier,
          )
          .mutate(ProfileRelationshipAction.mute),
    );
    await tester.pump();

    expect(find.text('Alice followed you'), findsNothing);
    completer.complete(const ProfileRelationship(muted: true));
    await tester.pumpAndSettle();
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
          path: '/profiles/:did',
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
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Alice followed you'));
    await tester.pumpAndSettle();
    expect(profileState?.pathParameters['did'], 'did:plc:alice');
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
              notification: _like('like-comment', subjectPost: _commentPost()),
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
          localizationsDelegates: appLocalizationsDelegates,
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
      final unavailable = _unavailable('00000000-0000-0000-0000-000000000002');

      await tester.pumpWidget(
        ProviderScope(
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
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
        find.ancestor(of: genericText, matching: find.byType(InkWell)),
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

  testWidgets('moderation row explains the update and opens full history', (
    tester,
  ) async {
    var openedModeration = false;
    final notification = ModerationNotification(
      SystemNotificationCommon(
        id: '00000000-0000-4000-8000-000000000325',
        createdAt: DateTime.utc(2026, 9, 11, 12),
        indexedAt: DateTime.utc(2026, 9, 11, 12, 0, 1),
      ),
      caseReference: ModerationCaseReference.parse(
        'MOD-550e8400-e29b-41d4-a716-446655440000',
      ),
    );
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) =>
              Scaffold(body: NotificationRow(notification: notification)),
        ),
        GoRoute(
          path: '/profile/settings/moderation',
          builder: (_, _) {
            openedModeration = true;
            return const Scaffold(body: Text('Moderation case'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text('Review an update to your account standing'),
      findsOneWidget,
    );
    expect(find.byIcon(CraftskyIcons.privacy), findsOneWidget);

    await tester.tap(find.text('Review an update to your account standing'));
    await tester.pumpAndSettle();

    expect(openedModeration, isTrue);
  });

  testWidgets('moderation history needs one back action to leave', (
    tester,
  ) async {
    final notification = ModerationNotification(
      SystemNotificationCommon(
        id: '00000000-0000-4000-8000-000000000326',
        createdAt: DateTime.utc(2026, 9, 11, 12),
        indexedAt: DateTime.utc(2026, 9, 11, 12, 0, 1),
      ),
      caseReference: ModerationCaseReference.parse(
        'MOD-550e8400-e29b-41d4-a716-446655440000',
      ),
    );
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) =>
              Scaffold(body: NotificationRow(notification: notification)),
        ),
        GoRoute(
          path: '/profile/settings/moderation',
          builder: (_, _) => const Scaffold(body: Text('Moderation case')),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Review an update to your account standing'));
    await tester.pumpAndSettle();
    expect(find.text('Moderation case'), findsOneWidget);

    router.pop();
    await tester.pumpAndSettle();

    expect(
      find.text('Review an update to your account standing'),
      findsOneWidget,
    );
    expect(find.text('Moderation case'), findsNothing);
  });
}

class _TestApp extends StatelessWidget {
  const new({required this.home, this.topPadding = 0});

  final Widget home;
  final double topPadding;

  @override
  Widget build(BuildContext context) => ProviderScope(
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: appLocalizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Builder(
        builder: (context) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(padding: EdgeInsets.only(top: topPadding)),
          child: home,
        ),
      ),
    ),
  );
}

FollowNotification _follow(
  String rkey, {
  String? displayName = 'Alice',
  bool includeAvatarUrl = true,
  String avatarCid = 'bafyavatar',
  bool viewerIsFollowing = false,
  List<String> crafts = const [],
}) => CraftskyNotification.fromMap({
  'uri': 'at://did:plc:alice/app.bsky.graph.follow/$rkey',
  'cid': 'bafy$rkey',
  'rkey': rkey,
  'type': 'follow',
  'actor': {
    'did': 'did:plc:alice',
    'handle': 'alice.craftsky.social',
    'displayName': ?displayName,
    if (includeAvatarUrl) 'avatar': 'https://cdn.example/avatar/alice.jpg',
    'avatarCid': avatarCid,
    'viewerIsFollowing': viewerIsFollowing,
    'crafts': crafts,
  },
  'createdAt': '2026-05-28T13:00:00Z',
  'indexedAt': '2026-05-28T13:00:01Z',
}) as FollowNotification;

InstagramMatchNotification _instagramMatch() => CraftskyNotification.fromMap({
  'id': '00000000-0000-0000-0000-000000000031',
  'type': 'instagramMatch',
  'actor': {
    'did': 'did:plc:alice',
    'handle': 'alice.craftsky.social',
    'displayName': 'Alice',
    'avatar': 'https://cdn.example/avatar/alice.jpg',
    'viewerIsFollowing': false,
  },
  'createdAt': '2026-05-28T13:00:00Z',
  'indexedAt': '2026-05-28T13:00:01Z',
}) as InstagramMatchNotification;

LikeNotification _like(String rkey, {Map<String, dynamic>? subjectPost}) =>
    CraftskyNotification.fromMap({
      ..._baseNotification('like', rkey),
      'subjectPost': subjectPost ?? _post(),
    }) as LikeNotification;

RepostNotification _repost(String rkey, {Map<String, dynamic>? subjectPost}) =>
    CraftskyNotification.fromMap({
      ..._baseNotification('repost', rkey),
      'subjectPost': subjectPost ?? _post(),
    }) as RepostNotification;

ReplyNotification _reply(
  String rkey, {
  bool includeFocus = true,
  Map<String, dynamic>? subjectPost,
}) => CraftskyNotification.fromMap({
  ..._baseNotification('reply', rkey),
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
  'subjectPost': subjectPost ?? _post(),
  'preview': {'text': 'new reply text', 'role': 'comment'},
  if (includeFocus)
    'reply': {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/$rkey',
      'cid': 'bafy$rkey',
      'rkey': rkey,
    },
}) as ReplyNotification;

MentionNotification _mention(String rkey) => CraftskyNotification.fromMap({
  ..._baseNotification('mention', rkey),
  'subjectPost': _post(),
}) as MentionNotification;

QuoteNotification _quote(String rkey) => CraftskyNotification.fromMap({
  ..._baseNotification('quote', rkey),
  'subjectPost': _post(),
}) as QuoteNotification;

GenericNotification _generic(String id, {required String type}) =>
    CraftskyNotification.fromMap({
      ..._baseNotification(type, 'generic'),
      'id': id,
    }) as GenericNotification;

UnavailableNotification _unavailable(String id) =>
    CraftskyNotification.fromMap({
      ..._baseNotification('like', 'unavailable'),
      'id': id,
      'actor': {
        'did': 'did:plc:alice',
        'handle': 'alice.craftsky.social',
        'available': false,
      },
    }) as UnavailableNotification;

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
    'customisation': {'colour': 'rose', 'profileBackground': 'none'},
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
  'viewerHasSaved': false,
  'sponsored': false,
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
  const new(this.page);

  final NotificationPage page;

  @override
  Future<NotificationPage> list({
    String? cursor,
    int? limit,
    NotificationCategory? category,
  }) async => page;
}

class _QueueNotificationRepository implements NotificationRepository {
  new(this.responses);

  final List<Future<NotificationPage>> responses;
  final calls = <_Call>[];

  @override
  Future<NotificationPage> list({
    String? cursor,
    int? limit,
    NotificationCategory? category,
  }) {
    calls.add(_Call(cursor: cursor, limit: limit, category: category));
    return responses.removeAt(0);
  }
}

final class _Call {
  const new({this.cursor, this.limit, this.category});
  final NotificationCategory? category;

  final String? cursor;
  final int? limit;
}
