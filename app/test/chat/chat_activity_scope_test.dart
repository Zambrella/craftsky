import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/chat/widgets/chat_app_bar_title.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('sync warning uses AppMessenger and retries '
      'without replacing the composer', (tester) async {
    final messenger = RecordingMessenger();
    final account = AccountKey('did:plc:owner');
    final repository = _Repository()..fail = true;
    var now = DateTime(2026, 10, 20);
    final coordinator = ChatSyncCoordinator(
      account: account,
      repository: repository,
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
      onRevision: (_, _, _) async {},
      now: () => now,
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
          builder: (context, child) =>
              MessengerScope(messenger: messenger, child: child!),
          theme: AppTheme.lightThemeData,
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
    await tester.pumpAndSettle();
    expect(
      find.text(
        'Messages may be out of date. Your loaded messages and draft are kept.',
      ),
      findsNothing,
    );
    expect(messenger.calls, hasLength(1));
    expect(messenger.calls.single.$1, 'warning');
    await tester.enterText(find.byType(TextField), 'synthetic draft');
    final original = tester
        .widget<TextField>(find.byType(TextField))
        .controller;
    repository.fail = false;
    now = now.add(const Duration(seconds: 2));
    await tester.pump(const Duration(seconds: 2));
    messenger.calls.single.$3!.onPressed();
    await tester.pumpAndSettle();
    expect(
      find.text(
        'Messages may be out of date. Your loaded messages and draft are kept.',
      ),
      findsNothing,
    );
    final field = tester.widget<TextField>(find.byType(TextField));
    expect(field.controller, same(original));
    expect(field.controller!.text, 'synthetic draft');
    expect(messenger.calls, hasLength(1));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();
  });

  testWidgets(
    'routine syncing preserves the full Scaffold and emits no alert',
    (tester) async {
      final messenger = RecordingMessenger();
      final account = AccountKey('did:plc:owner');
      final repository = _Repository()..syncState = 'syncing';
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
            builder: (context, child) =>
                MessengerScope(messenger: messenger, child: child!),
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ChatActivityScope(
              account: account,
              view: ChatView.inbox,
              child: Scaffold(
                appBar: AppBar(
                  title: const ChatAppBarTitle(
                    loading: false,
                    child: Text('Direct messages'),
                  ),
                ),
                body: const Text('Settings'),
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(tester.getTopLeft(find.byType(Scaffold)), Offset.zero);
      expect(tester.getTopLeft(find.byType(AppBar)), Offset.zero);
      expect(find.text('Updating your shared inbox...'), findsNothing);
      expect(messenger.calls, isEmpty);
      expect(find.byType(StitchProgressIndicator), findsNothing);
      repository.syncState = 'ready';
      coordinator.retry();
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(coordinator.snapshot?.state, 'ready');
      expect(coordinator.busy, false);
      expect(find.byType(StitchProgressIndicator), findsNothing);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();
    },
  );

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
  bool fail = false;
  String syncState = 'ready';
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    views.add(view);
    if (fail) throw const ApiServerError('synthetic failure');
    return ChatSyncSnapshot.fromMap({'revision': 1, 'state': syncState});
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
