import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_session_scope.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-006 session scope refreshes lightweight state on resume '
    'without polling',
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
            home: ChatSessionScope(
              child: StatefulBuilder(
                builder: (context, setState) {
                  changePage = setState;
                  return Scaffold(body: Text(page));
                },
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(repository.statusCalls, 1);
      expect(repository.countCalls, 1);
      changePage(() => page = 'Profile');
      await tester.pumpAndSettle();
      expect(repository.statusCalls, 1);
      expect(repository.countCalls, 1);
      [
        AppLifecycleState.inactive,
        AppLifecycleState.hidden,
        AppLifecycleState.paused,
      ].forEach(tester.binding.handleAppLifecycleStateChanged);
      await tester.pump(const Duration(seconds: 90));
      expect(repository.heartbeatCalls, 0);
      [
        AppLifecycleState.hidden,
        AppLifecycleState.inactive,
        AppLifecycleState.resumed,
      ].forEach(tester.binding.handleAppLifecycleStateChanged);
      await tester.pumpAndSettle();
      expect(repository.statusCalls, 2);
      expect(repository.countCalls, 2);
      expect(repository.heartbeatCalls, 0);
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
  @override
  Future<ChatStatus> status() async {
    statusCalls++;
    return ChatStatus.fromMap(const {
      'enabled': true,
      'pushEnabled': true,
      'generation': 1,
      'state': 'ready',
      'capabilities': {
        'canRead': true,
        'canSend': true,
        'canReport': true,
        'canSetPolicy': true,
      },
    });
  }

  @override
  Future<ChatUnreadCounts> unreadCounts() async {
    countCalls++;
    return ChatUnreadCounts.fromMap(const {'accepted': 0, 'requests': 0});
  }

  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    heartbeatCalls++;
    return ChatSyncSnapshot.fromMap(const {'revision': 1, 'state': 'ready'});
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
