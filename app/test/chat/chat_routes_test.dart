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
import 'package:craftsky_app/chat/widgets/profile_message_button.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

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
    'cold',
    'new message',
    'notification',
    'profile',
  ]) {
    testWidgets('IT-013 conversation Back returns to Messages from $entry', (
      tester,
    ) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final repository = _Repository(enabled: true, hasConversation: true);
      final inbox = _allRoutes($appRoutes)
          .whereType<GoRoute>()
          .singleWhere((route) => route.name == 'messages');
      const details = ChatConversationRoute(conversationId: 'conversation');
      String? openedProfileDid;
      Object? profilePresentation;
      final router = GoRouter(
        initialLocation: switch (entry) {
          'cold' => details.location,
          'new message' => const NewMessageRoute(
            recipientDid: 'did:plc:peer',
          ).location,
          'notification' => '/feed',
          'profile' => '/profile',
          _ => '/messages',
        },
        routes: [
          inbox,
          GoRoute(
            path: '/profiles/:did',
            builder: (context, state) {
              openedProfileDid = state.pathParameters['did'];
              profilePresentation = state.extra;
              return Scaffold(
                appBar: AppBar(title: const Text('Compact profile route')),
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
            chatDraftStoreProvider.overrideWithValue(
              ChatDraftStore(const FlutterSecureStorage()),
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
      if (entry == 'inbox') {
        await tester.tap(find.text('peer.test'));
      } else if (entry == 'new message') {
        await tester.tap(find.text('Start conversation'));
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
      expect(router.canPop(), true);

      if (entry == 'inbox') {
        await tester.enterText(find.byType(TextField), 'Unsent draft');
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
          expect(find.text('Compact profile route'), findsOneWidget);
          expect(openedProfileDid, 'did:plc:peer');
          expect(profilePresentation, isA<ProfilePresentationRequest>());
          expect(
            (profilePresentation! as ProfilePresentationRequest).startsCompact,
            true,
          );
          await tester.tap(find.byType(BackButton));
          await tester.pumpAndSettle();
          expect(
            tester.widget<TextField>(find.byType(TextField)).controller!.text,
            'Unsent draft',
          );
          expect(find.text('peer.test'), findsOneWidget);
        }
      }

      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();
      expect(find.widgetWithText(AppBar, 'Messages'), findsOneWidget);
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
        expect(router.canPop(), true);
        await tester.tap(find.byTooltip('Back'));
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
    testWidgets('UT-010 cold Messages route can return home enabled=$enabled', (
      tester,
    ) async {
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
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();
      expect(router.routeInformationProvider.value.uri.path, '/feed');
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    });
  }

  testWidgets(
    'SETUP-001 / ROUTE-001 disabled Messages shows enablement before preferences',
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
      expect(find.text('Direct messages'), findsOneWidget);
      expect(find.text('Push notifications'), findsOneWidget);
      expect(find.text('Who can message me'), findsOneWidget);
      expect(find.text('Open messages'), findsOneWidget);
      expect(find.byType(Switch), findsNWidgets(2));
      expect(tester.widget<Switch>(find.byType(Switch).first).value, true);
      expect(tester.widget<Switch>(find.byType(Switch).last).value, true);
      await tester.ensureVisible(find.byType(Switch).first);
      await tester.tap(find.byType(Switch).first);
      await tester.pumpAndSettle();
      expect(repository.effects.last, 'connection:false');
      expect(find.text('Push notifications'), findsNothing);
      expect(find.text('Who can message me'), findsNothing);
      expect(find.text('Open messages'), findsNothing);
      expect(tester.widget<Switch>(find.byType(Switch)).value, false);
      await tester.ensureVisible(find.byType(Switch));
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Open messages'));
      await tester.tap(find.text('Open messages'));
      await tester.pumpAndSettle();
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
  new({required this.enabled, this.requests = 0, this.hasConversation = false});
  bool enabled;
  final bool hasConversation;
  final effects = <String>[];
  final int requests;
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
      'canReport': enabled,
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
  }) async => ChatConversationPage(
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
  }) async => ChatMessagePage.fromMap(const {'items': <Object>[]});
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
