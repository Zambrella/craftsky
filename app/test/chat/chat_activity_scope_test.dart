import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-006 covered inbox cannot stop the visible conversation heartbeat',
    (tester) async {
      final account = AccountKey('did:plc:owner');
      final repository = _Repository();
      final coordinator = ChatSyncCoordinator(
        account: account,
        repository: repository,
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onRevision: (_, _, _) async {},
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            chatSyncCoordinatorProvider(account).overrideWith((ref) async {
              ref.onDispose(coordinator.dispose);
              return coordinator;
            }),
          ],
          child: MaterialApp(
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ChatActivityScope(
              account: account,
              view: ChatView.inbox,
              child: Builder(
                builder: (context) => Scaffold(
                  body: TextButton(
                    onPressed: () => Navigator.of(context).push<void>(
                      MaterialPageRoute(
                        builder: (context) => ChatActivityScope(
                          account: account,
                          view: ChatView.conversation,
                          conversationId: ChatConversationId.parse(
                            'conversation',
                          ),
                          child: Scaffold(
                            appBar: AppBar(),
                            body: const Text('Conversation'),
                          ),
                        ),
                      ),
                    ),
                    child: const Text('Open conversation'),
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(repository.views.last, ChatView.inbox);
      await tester.tap(find.text('Open conversation'));
      await tester.pumpAndSettle();
      final before = repository.views.length;
      await tester.pump(const Duration(seconds: 4));
      await tester.pumpAndSettle();
      expect(repository.views.length, greaterThan(before));
      expect(repository.views.last, ChatView.conversation);
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(repository.views.last, ChatView.inbox);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );

  testWidgets(
    'UT-010 synchronization admission preserves composer state and keyboard '
    'focus',
    (tester) async {
      final account = AccountKey('did:plc:owner');
      final admitted = Completer<ChatSyncCoordinator>();
      final coordinator = ChatSyncCoordinator(
        account: account,
        repository: _Repository(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onRevision: (_, _, _) async {},
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            chatSyncCoordinatorProvider(account).overrideWith((ref) async {
              ref.onDispose(coordinator.dispose);
              return await admitted.future;
            }),
          ],
          child: MaterialApp(
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ChatActivityScope(
              account: account,
              view: ChatView.inbox,
              child: const _Composer(),
            ),
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), 'private-draft');
      final original = tester
          .widget<TextField>(find.byType(TextField))
          .controller;
      admitted.complete(coordinator);
      await tester.pumpAndSettle();
      final field = tester.widget<TextField>(find.byType(TextField));
      expect(field.controller, same(original));
      expect(field.controller!.text, 'private-draft');
      expect(
        tester
            .state<EditableTextState>(find.byType(EditableText))
            .widget
            .focusNode
            .hasFocus,
        true,
      );
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );
}

class _Composer extends StatefulWidget {
  const new();
  @override
  State<_Composer> createState() => _ComposerState();
}

class _ComposerState extends State<_Composer> {
  final text = TextEditingController();
  @override
  void dispose() {
    text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) =>
      Scaffold(body: TextField(controller: text));
}

class _Repository implements ChatRepository {
  final views = <ChatView>[];
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    views.add(view);
    return ChatSyncSnapshot.fromMap(const {'revision': 1, 'state': 'ready'});
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
