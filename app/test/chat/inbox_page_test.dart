import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/messaging_inbox_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'IT-001/UT-010 inbox fits large text with shared buckets and native counts',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(320, 568));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final account = AccountKey('did:plc:alice');
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic-token',
        did: account.did.value,
        handle: 'alice.test',
      );
      final repository = _Repository();
      ChatConversationId? opened;
      var newMessage = 0;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
          ],
          child: MaterialApp(
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context)
                  .copyWith(textScaler: const TextScaler.linear(2)),
              child: child!,
            ),
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ChatActivityScope(
              account: account,
              view: ChatView.inbox,
              child: MessagingInboxPage(
                account: account,
                onOpen: (id) => opened = id,
                onNewMessage: () => newMessage++,
                onSettings: () {},
                onAuthorize: (_) {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('external.test'), findsOneWidget);
      expect(find.text('99+'), findsOneWidget);
      expect(repository.heartbeats, 1);
      await tester.pump(const Duration(seconds: 30));
      await tester.pumpAndSettle();
      expect(repository.heartbeats, 2);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump(const Duration(seconds: 60));
      expect(repository.heartbeats, 2);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();
      expect(repository.heartbeats, 3);

      await tester.tap(find.text('external.test'));
      expect(opened?.value, 'external-conversation');
      await tester.tap(find.byTooltip('New message'));
      expect(newMessage, 1);
      await tester.tap(find.text('Requests'));
      await tester.pumpAndSettle();
      expect(find.text('request.external.test'), findsOneWidget);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(MessagingInboxPage)),
      );
      await container
          .read(sessionRegistryProvider.notifier)
          .upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
      await tester.pumpAndSettle();
      expect(find.text('request.external.test'), findsNothing);
      expect(find.text('external.test'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
}

class _Repository implements ChatRepository {
  int heartbeats = 0;
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    heartbeats++;
    return ChatSyncSnapshot.fromMap({'revision': heartbeats, 'state': 'ready'});
  }

  @override
  Future<ChatStatus> status() async => ChatStatus.fromMap(const {
    'generation': 2,
    'enabled': true,
    'state': 'ready',
    'pushEnabled': true,
    'capabilities': {
      'canRead': true,
      'canSend': true,
      'canReport': true,
      'canSetPolicy': true,
    },
  });
  @override
  Future<ChatUnreadCounts> unreadCounts() async =>
      ChatUnreadCounts.fromMap(const {'accepted': 100, 'requests': 3});
  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async => ChatConversationPage.fromMap({
    'items': [
      {
        'id': bucket == ChatBucket.accepted
            ? 'external-conversation'
            : 'external-request',
        'revision': 'rev-1',
        'status': bucket.name,
        'muted': false,
        'unreadCount': 1,
        'canSend': bucket == ChatBucket.accepted,
        'peer': {
          'did': 'did:plc:external',
          'handle': bucket == ChatBucket.accepted
              ? 'external.test'
              : 'request.external.test',
        },
      },
    ],
  });
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
