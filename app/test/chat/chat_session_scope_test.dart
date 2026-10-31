import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/chat/widgets/chat_session_scope.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final boundary in [
    'logout',
    'account switch',
    'disabled',
    'revoked',
    'background',
    'disposed',
  ]) {
    testWidgets('FG-003 session fences pending sync after $boundary', (
      tester,
    ) async {
      final repository = _Repository();
      final otherRepository = _Repository();
      final account = AccountKey('did:plc:owner');
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(_Storage()),
          accountChatRepositoryProvider.overrideWith(
            (ref, owner) async =>
                owner == account ? repository : otherRepository,
          ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: const MaterialApp(
            home: ChatSessionScope(child: Scaffold(body: Text('Feed'))),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 1);
      final counts = repository.countCalls;
      final lists = repository.listCalls;
      final held = Completer<ChatSyncSnapshot>();
      repository.held = held;
      await tester.pump(const Duration(seconds: 30));
      expect(repository.heartbeatCalls, 2);
      switch (boundary) {
        case 'logout':
          final lease = container
              .read(sessionRegistryProvider)
              .requireValue
              .activeLease!;
          await container
              .read(sessionRegistryProvider.notifier)
              .removeConfirmed(lease.session);
        case 'account switch':
          await container
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'other',
                did: 'did:plc:other',
                handle: 'other.test',
              );
        case 'disabled':
        case 'revoked':
          if (boundary == 'disabled') repository.enabled = false;
          repository.canRead = false;
          container
              .read(chatStatusProvider(account).notifier)
              .publishConfirmedStatus(await repository.status());
        case 'background':
          [
            AppLifecycleState.inactive,
            AppLifecycleState.hidden,
            AppLifecycleState.paused,
          ].forEach(tester.binding.handleAppLifecycleStateChanged);
        case 'disposed':
          await tester.pumpWidget(const SizedBox.shrink());
      }
      await tester.pumpAndSettle();
      held.complete(
        ChatSyncSnapshot.fromMap(const {'revision': 2, 'state': 'ready'}),
      );
      await tester.pumpAndSettle();
      await tester.pump(const Duration(seconds: 90));
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 2);
      expect(repository.countCalls, counts);
      expect(repository.listCalls, lists);
      if (boundary == 'account switch') {
        expect(otherRepository.heartbeatCalls, greaterThan(0));
        expect(
          container
              .read(chatUnreadCountsProvider(AccountKey('did:plc:other')))
              .requireValue
              .accepted,
          0,
        );
      }
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
      if (boundary == 'background') {
        [
          AppLifecycleState.hidden,
          AppLifecycleState.inactive,
          AppLifecycleState.resumed,
        ].forEach(tester.binding.handleAppLifecycleStateChanged);
      }
    });
  }

  testWidgets(
    'FG-001 session refreshes inbox and unread while browsing Feed',
    (tester) async {
      final repository = _Repository();
      late StateSetter changePage;
      var page = 'Feed';
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(_Storage()),
            accountChatRepositoryProvider.overrideWith(
              (ref, account) async => repository,
            ),
          ],
          child: MaterialApp(
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ChatSessionScope(
              child: StatefulBuilder(
                builder: (context, setState) {
                  changePage = setState;
                  final body = Scaffold(body: Text(page));
                  return page == 'Conversation'
                      ? ChatActivityScope(
                          account: AccountKey('did:plc:owner'),
                          view: ChatView.conversation,
                          conversationId: ChatConversationId.parse(
                            'conversation',
                          ),
                          child: body,
                        )
                      : body;
                },
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(repository.statusCalls, 1);
      expect(repository.heartbeatCalls, 1);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(ChatSessionScope)),
      );
      final account = AccountKey('did:plc:owner');
      final inboxFamily = chatInboxControllerProvider(
        account,
        ChatBucket.accepted,
      );
      final inbox = container.read(inboxFamily).requireValue;
      expect(inbox.items.single.unreadCount, 0);
      repository.revision = 2;
      await tester.pump(const Duration(seconds: 30));
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 2);
      expect(inbox.items.single.unreadCount, 7);
      expect(
        container.read(chatUnreadCountsProvider(account)).requireValue.accepted,
        7,
      );
      final loadedLists = repository.listCalls;
      changePage(() => page = 'Profile');
      await tester.pumpAndSettle();
      expect(repository.statusCalls, 1);
      expect(repository.listCalls, loadedLists);
      changePage(() => page = 'Conversation');
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 3);
      repository.revision = 3;
      await tester.pump(const Duration(seconds: 4));
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 4);
      expect(inbox.items.single.revision, '3');
      changePage(() => page = 'Profile');
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 5);
      await tester.pump(const Duration(seconds: 4));
      expect(repository.heartbeatCalls, 5);
      await tester.pump(const Duration(seconds: 26));
      await tester.pumpAndSettle();
      expect(repository.heartbeatCalls, 6);
      [
        AppLifecycleState.inactive,
        AppLifecycleState.hidden,
        AppLifecycleState.paused,
      ].forEach(tester.binding.handleAppLifecycleStateChanged);
      await tester.pump(const Duration(seconds: 90));
      expect(repository.heartbeatCalls, 6);
      [
        AppLifecycleState.hidden,
        AppLifecycleState.inactive,
        AppLifecycleState.resumed,
      ].forEach(tester.binding.handleAppLifecycleStateChanged);
      await tester.pumpAndSettle();
      expect(repository.statusCalls, 2);
      expect(repository.heartbeatCalls, 7);
      expect(repository.listCalls, greaterThan(loadedLists));
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    },
  );
}

class _Repository implements ChatRepository {
  int statusCalls = 0;
  int countCalls = 0;
  int heartbeatCalls = 0;
  int listCalls = 0;
  int revision = 1;
  bool enabled = true;
  bool canRead = true;
  Completer<ChatSyncSnapshot>? held;
  @override
  Future<ChatStatus> status() async {
    statusCalls++;
    return ChatStatus.fromMap({
      'enabled': enabled,
      'pushEnabled': true,
      'generation': 1,
      'state': !enabled
          ? 'disabled'
          : canRead
          ? 'ready'
          : 'authorizationNeeded',
      'capabilities': {
        'canRead': canRead,
        'canSend': true,
        'canReport': true,
        'canSetPolicy': true,
      },
    });
  }

  @override
  Future<ChatUnreadCounts> unreadCounts() async {
    countCalls++;
    return ChatUnreadCounts.fromMap({
      'accepted': revision == 1 ? 0 : 7,
      'requests': 0,
    });
  }

  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    heartbeatCalls++;
    if (held != null) return await held!.future;
    return ChatSyncSnapshot.fromMap({'revision': revision, 'state': 'ready'});
  }

  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    listCalls++;
    return ChatConversationPage.fromMap({
      'items': [
        {
          'id': 'conversation',
          'revision': '$revision',
          'status': bucket.name,
          'muted': false,
          'unreadCount': revision == 1 ? 0 : 7,
          'canSend': true,
          'peer': const {'did': 'did:plc:peer', 'handle': 'peer.test'},
        },
      ],
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Storage implements SessionRegistryStorage {
  SessionRegistry value = SessionRegistry.empty().upsertAndActivate(
    token: 'synthetic',
    did: 'did:plc:owner',
    handle: 'owner.test',
  );
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}
