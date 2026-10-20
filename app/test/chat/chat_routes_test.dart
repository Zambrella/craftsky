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
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

void main() {
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
      await tester.tap(find.byTooltip('Back'));
      await tester.pumpAndSettle();
      expect(router.routeInformationProvider.value.uri.path, '/feed');
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    });
  }

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
  new({required this.enabled});
  final bool enabled;
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
  Future<ChatUnreadCounts> unreadCounts() async =>
      ChatUnreadCounts.fromMap(const {'accepted': 0, 'requests': 0});
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
