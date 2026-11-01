import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_conversation_details.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/messaging_route_page.dart';
import 'package:craftsky_app/chat/providers/chat_draft_store_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_recipient_picker.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/chat/widgets/profile_message_button.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  test('IT-013 messaging destinations belong to the Messages route', () {
    final inbox = _allRoutes($appRoutes)
        .whereType<GoRoute>()
        .singleWhere((route) => route.name == 'messages');
    expect(
      inbox.routes.whereType<GoRoute>().map((route) => route.name),
      containsAll(['message-conversation', 'message-requests', 'new-message']),
    );
  });
  for (final entry in [
    'inbox',
    'cached return',
    'cold',
    'new message',
    'notification',
    'profile',
    'reporting disabled',
  ]) {
    testWidgets('IT-013 conversation Back returns to Messages from $entry', (
      tester,
    ) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final repository = _Repository(
        enabled: true,
        hasConversation: true,
        hasMessage: entry == 'inbox' || entry == 'cached return',
        canReport: entry != 'reporting disabled',
      );
      final draftStore = ChatDraftStore(const FlutterSecureStorage());
      final inbox = _allRoutes($appRoutes)
          .whereType<GoRoute>()
          .singleWhere((route) => route.name == 'messages');
      const details = ChatConversationRoute(conversationId: 'conversation');
      String? openedProfileDid;
      Object? profilePresentation;
      final router = GoRouter(
        navigatorKey: ChatConversationRoute.$parentNavigatorKey,
        initialLocation: switch (entry) {
          'cold' || 'reporting disabled' => details.location,
          'new message' => const NewMessageRoute(
            recipientDid: 'did:plc:peer',
          ).location,
          'notification' => '/feed',
          'profile' => '/profile',
          _ => '/messages',
        },
        routes: [
          ShellRoute(
            builder: (_, _, child) => Scaffold(
              body: child,
              bottomNavigationBar: const Text('Bottom navigation'),
            ),
            routes: [inbox],
          ),
          GoRoute(
            path: '/profiles/:did',
            builder: (context, state) {
              openedProfileDid = state.pathParameters['did'];
              profilePresentation = state.extra;
              return Scaffold(
                appBar: AppBar(title: const Text('Profile route')),
              );
            },
          ),
          GoRoute(
            path: '/feed',
            builder: (_, _) => const Scaffold(body: Text('Feed')),
          ),
          GoRoute(
            path: '/profile',
            builder: (context, _) => Scaffold(
              body: ProfileMessageButton(
                recipient: Did.parse('did:plc:peer'),
                onOpen: (id) =>
                    ChatConversationRoute(conversationId: id.value).go(context),
                onEnable: () => fail('Messaging is already enabled'),
              ),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'synthetic',
                  did: account.did.value,
                  handle: 'owner.test',
                ),
              ),
            ),
            profileRepositoryProvider.overrideWithValue(
              FakeProfileRepository(
                onFetch: (did) async => Profile(
                  did: did,
                  handle: 'peer.test',
                  crafts: const ['knitting'],
                ),
              ),
            ),
            accountChatRepositoryProvider(account).overrideWith(
              (ref) async => repository,
            ),
            chatRecipientControllerProvider(account).overrideWith((ref) async {
              final controller = ChatRecipientController(
                account: account,
                repository: repository,
                searchRepository: _NoSearch(),
                operation: ActiveAccountOperation.withGuard(
                  ownership: null,
                  isCurrent: () => true,
                ),
              );
              ref.onDispose(controller.dispose);
              return controller;
            }),
            chatDraftStoreProvider.overrideWithValue(draftStore),
          ],
          child: MaterialApp.router(
            builder: (context, child) =>
                MessengerScope(messenger: RecordingMessenger(), child: child!),
            routerConfig: router,
            theme: AppTheme.lightThemeData.copyWith(
              platform: entry == 'inbox'
                  ? TargetPlatform.iOS
                  : TargetPlatform.android,
            ),
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      if (entry == 'inbox' || entry == 'cached return') {
        if (entry == 'cached return') {
          repository.pendingConversations = Completer<ChatConversationPage>();
        }
        await tester.tap(find.text('peer.test'));
      } else if (entry == 'profile') {
        await tester.tap(find.byTooltip('Message'));
      } else if (entry == 'notification') {
        navigateToNotificationOutcome(
          tester.element(find.text('Feed')),
          router,
          NotificationOpenOutcome(
            destination: ChatConversationDestination(
              ChatConversationId.parse('conversation'),
            ),
            isCurrent: () => true,
          ),
        );
      }
      await tester.pumpAndSettle();
      expect(find.byType(TextField), findsOneWidget);
      expect(find.text('Bottom navigation'), findsNothing);
      expect(router.canPop(), true);
      if (entry == 'cached return') {
        expect(repository.effects, contains('read:message'));
        await tester.tap(find.byTooltip('Back'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 400));
        expect(find.byType(ChatListSkeleton), findsNothing);
        expect(find.widgetWithText(ListTile, 'peer.test'), findsOneWidget);
        repository.pendingConversations!.complete(
          ChatConversationPage(
            items: [
              await repository.conversation(
                ChatConversationId.parse('conversation'),
              ),
            ],
          ),
        );
        await tester.pumpAndSettle();
        expect(find.widgetWithText(ListTile, 'peer.test'), findsOneWidget);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
        return;
      }
      if (entry == 'reporting disabled') {
        await tester.tap(find.byTooltip('Conversation actions'));
        await tester.pumpAndSettle();
        expect(find.text('Report conversation'), findsNothing);
        await tester.tapAt(const Offset(10, 10));
        await tester.pumpAndSettle();
        expect(repository.effects, isNot(contains('report-conversation')));
      }

      if (entry == 'inbox') {
        await tester.enterText(find.byType(TextField), 'Unsent draft');
        await tester.tap(find.byTooltip('Conversation actions'));
        await tester.pumpAndSettle();
        expect(
          tester.widget<Text>(find.text('Report conversation')).style?.color,
          AppTheme.lightThemeData.extension<SemanticColorsTheme>()!.error,
        );
        await tester.tap(find.text('Report conversation'));
        await tester.pumpAndSettle();
        expect(find.text('Report conversation'), findsOneWidget);
        expect(
          find.textContaining('Conversation reports go to Bluesky only'),
          findsOneWidget,
        );
        expect(
          tester
              .widget<TextButton>(find.widgetWithText(TextButton, 'Submit'))
              .onPressed,
          isNull,
        );
        expect(repository.effects, isNot(contains('report-conversation')));
        final reasonSearch = find.byKey(
          const Key('chat-report-reason-search-input'),
        );
        await tester.tap(reasonSearch);
        await tester.enterText(reasonSearch, 'Spam');
        await tester.pumpAndSettle();
        await tester.tap(
          find.descendant(
            of: find.byKey(const Key('chat-report-reason-options-panel')),
            matching: find.text('Spam'),
          ),
        );
        await tester.pumpAndSettle();
        await tester.enterText(
          find.byKey(const Key('chat-report-explanation')),
          'My conversation concern',
        );
        await tester.tap(find.text('Submit'));
        await tester.pumpAndSettle();
        expect(find.text('Report sent to Bluesky.'), findsOneWidget);
        expect(
          repository.effects.where((effect) => effect == 'report-conversation'),
          hasLength(1),
        );
        expect(repository.effects, isNot(contains('leave')));
        await tester.tap(find.byType(CloseButton));
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          'Unsent draft',
        );
        await tester.longPress(find.text('Reportable message'));
        await tester.pumpAndSettle();
        await tester.tap(find.text('Report message'));
        await tester.pumpAndSettle();
        expect(find.byType(BottomSheet), findsNothing);
        expect(find.byType(Dialog), findsNothing);
        final reportContext = tester.element(find.text('Report message'));
        expect(ModalRoute.of(reportContext), isA<FullscreenModalRoute<void>>());
        expect(
          tester
              .getSize(
                find.ancestor(
                  of: find.text('Report message'),
                  matching: find.byType(Scaffold),
                ),
              )
              .height,
          tester.view.physicalSize.height / tester.view.devicePixelRatio,
        );
        expect(find.textContaining('Bluesky only'), findsOneWidget);
        expect(find.textContaining('reference to this message'), findsNothing);
        addTearDown(tester.view.resetViewInsets);
        tester.view.viewInsets = FakeViewPadding(
          bottom: 250 * tester.view.devicePixelRatio,
        );
        await tester.pumpAndSettle();
        await tester.enterText(
          find.byKey(const Key('chat-report-explanation')),
          'Unsubmitted report details',
        );
        await tester.ensureVisible(find.text('Submit'));
        await tester.pumpAndSettle();
        expect(find.text('Submit').hitTestable(), findsOneWidget);
        expect(tester.takeException(), isNull);
        tester.view.resetViewInsets();
        await tester.pumpAndSettle();

        await tester.tap(find.byType(CloseButton));
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          'Unsent draft',
        );

        expect(find.byType(CraftIcon), findsOneWidget);
        for (final avatar in [true, false]) {
          await tester.tap(
            find.descendant(
              of: find.byType(AppBar),
              matching: avatar
                  ? find.byType(ProfileAvatar)
                  : find.text('peer.test'),
            ),
          );
          await tester.pumpAndSettle();
          expect(find.text('Profile route'), findsOneWidget);
          expect(openedProfileDid, 'did:plc:peer');
          expect(profilePresentation, isA<ProfileAvatarTransition>());
          expect((profilePresentation! as ProfileAvatarTransition).crafts, [
            'knitting',
          ]);
          await tester.tap(find.byType(BackButton));
          await tester.pumpAndSettle();
          expect(
            tester.widget<TextField>(find.byType(TextField)).controller!.text,
            'Unsent draft',
          );
          expect(find.text('peer.test'), findsOneWidget);
        }
      }

      if (entry == 'inbox') {
        await tester.enterText(find.byType(TextField), 'Fresh swipe draft');
        await tester.dragFrom(const Offset(1, 300), const Offset(650, 0));
      } else {
        await tester.tap(find.byTooltip('Back'));
      }
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
      if (entry == 'inbox') {
        expect(
          (await draftStore.load(
            draftStore.lease(account),
            ChatConversationId.parse('conversation'),
          )).text,
          'Fresh swipe draft',
        );
        await tester.tap(find.text('peer.test'));
        await tester.pumpAndSettle();
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          'Fresh swipe draft',
        );
        await tester.tap(find.byTooltip('Back'));
        await tester.pumpAndSettle();
      }
      expect(find.text('peer.test'), findsOneWidget);
      await tester.tap(find.text('peer.test'));
      await tester.pumpAndSettle();
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
      expect(find.text('peer.test'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    });
  }
  for (final destination in ['Requests', 'New message']) {
    for (final entry in ['inbox', 'cold']) {
      testWidgets('IT-013 $destination returns to Messages from $entry', (
        tester,
      ) async {
        final account = AccountKey('did:plc:owner');
        final inbox = _allRoutes($appRoutes)
            .whereType<GoRoute>()
            .singleWhere((route) => route.name == 'messages');
        final repository = _Repository(enabled: true);
        final childPath = destination == 'Requests'
            ? const ChatRequestsRoute().location
            : const NewMessageRoute().location;
        final router = GoRouter(
          navigatorKey: ChatConversationRoute.$parentNavigatorKey,
          initialLocation: switch (entry) {
            'cold' => childPath,
            _ => '/messages',
          },
          routes: [inbox],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              secureSessionRegistryStorageProvider.overrideWithValue(
                _Storage(
                  SessionRegistry.empty().upsertAndActivate(
                    token: 'synthetic',
                    did: account.did.value,
                    handle: 'owner.test',
                  ),
                ),
              ),
              accountChatRepositoryProvider(account)
                  .overrideWith((ref) async => repository),
              chatRecipientControllerProvider(account)
                  .overrideWith((ref) async {
                    final controller = ChatRecipientController(
                      account: account,
                      repository: repository,
                      searchRepository: _NoSearch(),
                      operation: ActiveAccountOperation.withGuard(
                        ownership: null,
                        isCurrent: () => true,
                      ),
                    );
                    ref.onDispose(controller.dispose);
                    return controller;
                  }),
            ],
            child: MaterialApp.router(
              routerConfig: router,
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
            ),
          ),
        );
        await tester.pumpAndSettle();
        if (entry == 'inbox') {
          await tester.tap(find.byTooltip(destination));
          await tester.pumpAndSettle();
        }
        expect(find.widgetWithText(AppBar, destination), findsOneWidget);
        if (destination == 'New message') {
          expect(find.byType(ChatRecipientPicker), findsOneWidget);
          expect(
            ModalRoute.of(tester.element(find.byType(ChatRecipientPicker))),
            isA<FullscreenModalRoute<void>>(),
          );
        }
        expect(router.canPop(), true);
        await tester.tap(
          destination == 'New message'
              ? find.byType(CloseButton)
              : find.byTooltip('Back'),
        );
        await tester.pumpAndSettle();
        expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
        await tester.tap(find.byTooltip(destination));
        await tester.pumpAndSettle();
        await tester.binding.handlePopRoute();
        await tester.pumpAndSettle();
        expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump();
      });
    }
  }
  testWidgets(
    'UT-013 chats use a FAB and a badged request route without tabs',
    (
      tester,
    ) async {
      final account = AccountKey('did:plc:owner');
      final router = GoRouter(
        initialLocation: '/messages',
        routes: [
          GoRoute(
            path: '/messages',
            builder: (_, _) => const MessagingRoutePage(
              destination: MessagingDestination.inbox,
            ),
          ),
          GoRoute(
            path: '/messages/requests',
            builder: (_, _) => const MessagingRoutePage(
              destination: MessagingDestination.requests,
            ),
          ),
          GoRoute(
            path: '/messages/new',
            builder: (_, _) => Scaffold(
              appBar: AppBar(title: const Text('New conversation destination')),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'synthetic',
                  did: account.did.value,
                  handle: 'owner.test',
                ),
              ),
            ),
            accountChatRepositoryProvider(account).overrideWith(
              (ref) async => _Repository(enabled: true, requests: 3),
            ),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(TabBar), findsNothing);
      expect(find.byType(TabBarView), findsNothing);
      expect(find.byType(CraftskyFloatingActionButton), findsOneWidget);
      final requests = find.byTooltip('Requests');
      expect(
        find.descendant(of: requests, matching: find.text('3')),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: find.byType(AppBar),
          matching: find.byTooltip('New message'),
        ),
        findsNothing,
      );
      await tester.tap(requests);
      await tester.pumpAndSettle();
      expect(
        GoRouterState.of(tester.element(find.byType(MessagingRoutePage)))
            .uri
            .path,
        '/messages/requests',
      );
      expect(find.widgetWithText(AppBar, 'Requests'), findsOneWidget);
      expect(find.byType(TabBar), findsNothing);
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();
      expect(router.routeInformationProvider.value.uri.path, '/messages');
      await tester.tap(find.byTooltip('New message'));
      await tester.pumpAndSettle();
      expect(find.text('New conversation destination'), findsOneWidget);
      expect(router.canPop(), true);
      expect(
        find.byType(MessagingRoutePage, skipOffstage: false),
        findsOneWidget,
      );
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
      await tester.tap(find.byTooltip('New message'));
      await tester.pumpAndSettle();
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );
  for (final enabled in [false, true]) {
    testWidgets(
      'UT-010 Messages root exposes drawer instead of Back enabled=$enabled',
      (
        tester,
      ) async {
        var drawerOpens = 0;
        final account = AccountKey('did:plc:owner');
        final repository = _Repository(enabled: enabled);
        final router = GoRouter(
          initialLocation: '/messages',
          routes: [
            GoRoute(
              path: '/messages',
              builder: (context, state) => const MessagingRoutePage(
                destination: MessagingDestination.inbox,
              ),
            ),
            GoRoute(
              path: '/feed',
              builder: (context, state) => const Scaffold(body: Text('Feed')),
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              secureSessionRegistryStorageProvider.overrideWithValue(
                _Storage(
                  SessionRegistry.empty().upsertAndActivate(
                    token: 'synthetic',
                    did: account.did.value,
                    handle: 'owner.test',
                  ),
                ),
              ),
              accountChatRepositoryProvider(account)
                  .overrideWith((ref) async => repository),
            ],
            child: MaterialApp.router(
              routerConfig: router,
              builder: (_, child) => AppShellDrawerScope(
                openDrawer: () => drawerOpens++,
                isDrawerOpen: false,
                child: child!,
              ),
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(
          find.text(enabled ? 'Messages' : 'Direct messages'),
          findsOneWidget,
        );
        expect(
          find.text('Enable messaging'),
          enabled ? findsNothing : findsOneWidget,
        );
        expect(find.byType(BackButton), findsNothing);
        await tester.tap(find.byType(AppShellDrawerButton));
        await tester.pumpAndSettle();
        expect(drawerOpens, 1);
        expect(router.routeInformationProvider.value.uri.path, '/messages');
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump();
      },
    );
  }

  testWidgets(
    'SETUP-001 / ROUTE-001 first enablement opens the conversation list',
    (tester) async {
      final account = AccountKey('did:plc:owner');
      final repository = _Repository(enabled: false);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'synthetic',
                  did: account.did.value,
                  handle: 'owner.test',
                ),
              ),
            ),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const MessagingRoutePage(
              destination: MessagingDestination.inbox,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Direct messages'), findsOneWidget);
      expect(find.text('Enable messaging'), findsOneWidget);
      expect(find.byType(Switch), findsOneWidget);
      expect(tester.widget<Switch>(find.byType(Switch)).value, false);
      expect(find.text('Push notifications'), findsNothing);
      expect(find.text('Who can message me'), findsNothing);
      expect(find.text('Continue with Bluesky'), findsNothing);
      expect(find.textContaining('shared Bluesky inbox'), findsOneWidget);
      expect(find.textContaining('not end-to-end encrypted'), findsOneWidget);
      expect(find.textContaining('Bluesky only'), findsOneWidget);
      expect(repository.effects, isEmpty);
      await tester.ensureVisible(find.byType(Switch));
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();
      expect(
        repository.effects.where((effect) => !effect.startsWith('read:')),
        ['connection:true'],
      );
      expect(repository.effects, containsAll(['read:push', 'read:policy']));
      expect(find.text('Messages'), findsOneWidget);
      expect(find.text('Direct messages'), findsNothing);
      expect(find.text('New message'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );

  test(
    'IT-013 messaging routes reconstruct from canonical paths without extras',
    () {
      expect(const ChatInboxRoute().location, '/messages');
      expect(const ChatRequestsRoute().location, '/messages/requests');
      expect(const NewMessageRoute().location, '/messages/new');
      expect(
        const ChatConversationRoute(conversationId: 'conversation-1').location,
        '/messages/conversations/conversation-1',
      );
      expect(const ChatSettingsRoute().location, '/settings/messaging');
    },
  );
}

Iterable<RouteBase> _allRoutes(Iterable<RouteBase> routes) sync* {
  for (final route in routes) {
    yield route;
    yield* _allRoutes(route.routes);
  }
}

class _Repository implements ChatRepository {
  new({
    required this.enabled,
    this.requests = 0,
    this.hasConversation = false,
    this.hasMessage = false,
    this.canReport = true,
  });
  bool enabled;
  Completer<ChatConversationPage>? pendingConversations;
  final bool hasConversation;
  final bool hasMessage;
  final bool canReport;
  final effects = <String>[];
  final int requests;
  @override
  Future<ChatReceipt> reportConversation(
    ChatOperationId operation,
    ChatConversationId id,
    String reasonType,
    String? reason,
  ) async {
    effects.add('report-conversation');
    expect(reasonType, 'com.atproto.moderation.defs#reasonSpam');
    expect(reason, 'My conversation concern');
    return ChatReceipt.fromMap({
      'operationId': operation.value,
      'kind': 'report',
      'state': 'confirmed',
    });
  }

  @override
  Future<ChatAvailability> availability(Did recipient) async =>
      ChatAvailability.fromMap(const {'canChat': true});
  @override
  Future<ChatConversation> start(Did recipient) =>
      conversation(ChatConversationId.parse('conversation'));
  @override
  Future<ChatStatus> status() async => ChatStatus.fromMap({
    'generation': 1,
    'enabled': enabled,
    'pushEnabled': true,
    'state': enabled ? 'ready' : 'disabled',
    'capabilities': {
      'canRead': enabled,
      'canSend': enabled,
      'canReport': enabled && canReport,
      'canSetPolicy': enabled,
    },
  });
  @override
  Future<ChatStatus> setConnection({required bool enabled}) async {
    effects.add('connection:$enabled');
    this.enabled = enabled;
    return await status();
  }

  @override
  Future<bool> pushPreference() async {
    effects.add('read:push');
    return true;
  }

  @override
  Future<ChatPolicySetting> incomingPolicy() async {
    effects.add('read:policy');
    return ChatPolicySetting.fromMap(const {
      'allowIncoming': 'following',
      'persisted': true,
    });
  }

  @override
  Future<ChatUnreadCounts> unreadCounts() async =>
      ChatUnreadCounts.fromMap({'accepted': 0, 'requests': requests});
  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async => pendingConversations != null
      ? await pendingConversations!.future
      : ChatConversationPage(
          items: [
            if (hasConversation)
              await conversation(ChatConversationId.parse('conversation')),
          ],
        );
  @override
  Future<ChatConversationDetails> details(ChatConversationId id) async =>
      ChatConversationDetails(
        conversation: await conversation(id),
        messages: await history(id),
      );

  @override
  Future<ChatConversation> conversation(ChatConversationId id) async =>
      ChatConversation.fromMap({
        'id': id.value,
        'revision': 'r',
        'status': 'accepted',
        'muted': false,
        'unreadCount': 0,
        'canSend': true,
        'peer': const {'did': 'did:plc:peer', 'handle': 'peer.test'},
      });
  @override
  Future<ChatMessagePage> history(
    ChatConversationId id, {
    ChatCursor? cursor,
    int limit = 50,
  }) async => ChatMessagePage.fromMap({
    'items': [
      if (hasMessage)
        {
          'id': 'peer-message',
          'revision': 'r',
          'kind': 'text',
          'text': 'Reportable message',
          'senderDid': 'did:plc:peer',
          'sentAt': '2026-10-30T12:00:00Z',
        },
    ],
  });
  @override
  Future<ChatConversation> updateRead(
    ChatConversationId id,
    ChatMessageId message,
  ) {
    effects.add('read:message');
    return conversation(id);
  }

  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async => ChatSyncSnapshot.fromMap(const {'revision': 1, 'state': 'ready'});
  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _NoSearch implements SearchRepository {
  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry value) async {
    this.value = value;
  }
}
