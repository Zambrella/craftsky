import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/messaging_route_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_floating_action_button.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

void main() {
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
            builder: (_, _) =>
                const Scaffold(body: Text('New conversation destination')),
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

class _Repository implements ChatRepository {
  new({required this.enabled, this.requests = 0});
  bool enabled;
  final effects = <String>[];
  final int requests;
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
  }) async => ChatConversationPage(items: const []);
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async => ChatSyncSnapshot.fromMap(const {'revision': 1, 'state': 'ready'});
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
