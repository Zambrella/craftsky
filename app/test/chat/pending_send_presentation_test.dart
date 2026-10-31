import 'package:craftsky_app/chat/adapters/flutter_chat_adapter.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_outgoing_message.dart';
import 'package:craftsky_app/chat/widgets/chat_time_header.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_chat_ui/flutter_chat_ui.dart' as ui;
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final historyFirst in [false, true]) {
    testWidgets('UT-009 reconciles by receipt ID historyFirst=$historyFirst', (
      tester,
    ) async {
      final attempt = _draft().pending!;
      var outgoing = [ChatOutgoingMessage(attempt: attempt)];
      var history = const ChatHistory.empty();
      late StateSetter update;
      var previews = 0;
      final presented = <ChatMessageId>[];
      await tester.pumpWidget(
        _app(
          StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return FlutterChatAdapter(
                history: history,
                outgoing: outgoing,
                owner: Did.parse('did:plc:owner'),
                composer: const SizedBox.shrink(),
                onPresented: presented.addAll,
                messageFooterBuilder: (_) {
                  previews++;
                  return const Text('Preview');
                },
                onReport: (_) => fail('Pending bubbles cannot report'),
                onDelete: (_) => fail('Pending bubbles cannot be deleted'),
              );
            },
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 300));
      final localId = tester
          .widget<ui.SimpleTextMessage>(find.byType(ui.SimpleTextMessage))
          .message
          .id;
      expect(previews, 0);
      expect(presented, isEmpty);
      await tester.longPress(find.text('Repeated text'));
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.text('Copy'), findsNothing);
      if (historyFirst) {
        update(() => history = history.merge(_page()));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));
        // Matching text alone never resolves the pending operation.
        expect(find.byType(ui.SimpleTextMessage), findsNWidgets(2));
      }
      update(
        () => outgoing = [
          ChatOutgoingMessage(
            attempt: attempt,
            messageId: ChatMessageId.parse('confirmed'),
          ),
        ],
      );
      await tester.pumpAndSettle();
      expect(find.byType(ui.SimpleTextMessage), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsNothing);
      if (!historyFirst) {
        expect(
          tester
              .widget<ui.SimpleTextMessage>(find.byType(ui.SimpleTextMessage))
              .showTime,
          false,
        );
        update(() => history = history.merge(_page()));
        await tester.pumpAndSettle();
      }
      final bubble = tester.widget<ui.SimpleTextMessage>(
        find.byType(ui.SimpleTextMessage),
      );
      expect(bubble.message.id, localId);
      expect(bubble.showTime, false);
      expect(find.byType(ChatTimeHeader), findsOneWidget);
      expect(bubble.showStatus, false);
      expect(previews, greaterThan(0));
      expect(presented, everyElement(ChatMessageId.parse('confirmed')));
      update(() => history = history.merge(_page(id: 'another')));
      await tester.pumpAndSettle();
      expect(find.byType(ui.SimpleTextMessage), findsNWidgets(2));
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('UT-009 restored attempt is uncertain and stays isolated', (
    tester,
  ) async {
    final restored = ChatDraft.fromStorage(_draft().toStorage());
    expect(restored.pending!.state, ChatDraftSendState.uncertain);
    var outgoing = [ChatOutgoingMessage(attempt: restored.pending!)];
    var owner = Did.parse('did:plc:owner');
    late StateSetter update;
    await tester.pumpWidget(
      _app(
        StatefulBuilder(
          builder: (_, setState) {
            update = setState;
            return FlutterChatAdapter(
              history: const ChatHistory.empty(),
              outgoing: outgoing,
              owner: owner,
              composer: const SizedBox.shrink(),
            );
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    final bubble = tester.widget<ui.SimpleTextMessage>(
      find.byType(ui.SimpleTextMessage),
    );
    expect(bubble.message.text, 'Repeated text');
    expect(bubble.showTime, false);
    expect(bubble.showStatus, false);
    expect(find.byType(CircularProgressIndicator), findsNothing);
    update(() {
      outgoing = [];
      owner = Did.parse('did:plc:other');
    });
    await tester.pumpAndSettle();
    expect(find.text('Repeated text'), findsNothing);
    expect(tester.takeException(), isNull);
  });
}

ChatDraft _draft() => const ChatDraft.empty()
    .edit('Repeated text')
    .beginSend(
      ChatOperationId.parse('00000000-0000-4000-8000-000000000123'),
    );

ChatMessagePage _page({String id = 'confirmed'}) => ChatMessagePage.fromMap({
  'items': [
    {
      'id': id,
      'revision': 'r',
      'kind': 'text',
      'text': 'Repeated text',
      'senderDid': 'did:plc:owner',
      'sentAt': '2026-10-31T12:00:00Z',
    },
  ],
});

Widget _app(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: appLocalizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);
