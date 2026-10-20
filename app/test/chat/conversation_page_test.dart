import 'dart:async';

import 'package:craftsky_app/chat/adapters/flutter_chat_adapter.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_chat_ui/flutter_chat_ui.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-010 older-page insertion preserves the package scroll anchor',
    (tester) async {
      await tester.binding.setSurfaceSize(const Size(360, 300));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      Map<String, dynamic> item(int index) => {
        'id': 'message-$index',
        'revision': 'rev-$index',
        'kind': 'text',
        'text': 'Message $index',
        'senderDid': 'did:plc:peer',
        'sentAt': DateTime.utc(2026, 10, 19, 12, index).toIso8601String(),
      };
      var history = const ChatHistory.empty().merge(
        ChatMessagePage.fromMap({
          'items': [for (var i = 10; i < 30; i++) item(i)],
          'cursor': '00000000-0000-4000-8000-000000000123',
        }),
      );
      var loads = 0;
      final loaded = Completer<void>();
      late StateSetter setView;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: StatefulBuilder(
              builder: (_, setState) {
                setView = setState;
                return FlutterChatAdapter(
                  history: history,
                  owner: Did.parse('did:plc:owner'),
                  composer: const SizedBox.shrink(),
                  onLoadOlder: () async {
                    loads++;
                    await loaded.future;
                  },
                );
              },
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.drag(find.byType(CustomScrollView), const Offset(0, 1800));
      await tester.pump(const Duration(milliseconds: 300));
      // Settle the drag's boundary spring while the loading spinner runs.
      await tester.pump(const Duration(seconds: 1));
      expect(loads, 1);
      expect(find.byType(LinearProgressIndicator), findsOneWidget);
      final before = tester.getTopLeft(find.text('Message 10')).dy;
      setView(() {
        history = history.merge(
          ChatMessagePage.fromMap({
            'items': [for (var i = 0; i < 10; i++) item(i)],
          }),
        );
      });
      loaded.complete();
      await tester.pumpAndSettle();
      expect(tester.getTopLeft(find.text('Message 10')).dy, closeTo(before, 2));
      expect(loads, 1);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets('UT-004 package viewport reports visible messages only', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(360, 300));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final history = const ChatHistory.empty().merge(
      ChatMessagePage.fromMap({
        'items': [
          for (var index = 0; index < 30; index++)
            {
              'id': 'message-$index',
              'revision': 'rev-$index',
              'kind': 'text',
              'text': 'Message $index',
              'senderDid': 'did:plc:peer',
              'sentAt': DateTime.utc(2026, 10, 19, 12, index).toIso8601String(),
            },
        ],
      }),
    );
    final seen = <List<ChatMessageId>>[];
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: FlutterChatAdapter(
            history: history,
            owner: Did.parse('did:plc:owner'),
            composer: const SizedBox.shrink(),
            onPresented: seen.add,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(seen, isNotEmpty);
    expect(seen.last, contains(ChatMessageId.parse('message-29')));
    expect(seen.last, isNot(contains(ChatMessageId.parse('message-0'))));
    expect(seen.last.length, lessThan(15));
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 250));
    await tester.pumpAndSettle();
    expect(seen.last, isNot(contains(ChatMessageId.parse('message-29'))));
    expect(tester.takeException(), isNull);
  });
  testWidgets(
    'UT-010 selected package renders safe text and tombstones'
    ' with a custom composer',
    (tester) async {
      final history = const ChatHistory.empty().merge(
        ChatMessagePage.fromMap(const {
          'items': [
            {
              'id': 'message-1',
              'revision': 'rev-1',
              'kind': 'text',
              'text': 'private-bubble https://external.test/never-fetch',
              'senderDid': 'did:plc:peer',
              'sentAt': '2026-10-19T12:00:00Z',
            },
            {
              'id': 'message-2',
              'revision': 'rev-2',
              'kind': 'deleted',
              'sentAt': '2026-10-19T12:01:00Z',
            },
          ],
        }),
      );
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: FlutterChatAdapter(
              history: history,
              owner: Did.parse('did:plc:owner'),
              composer: const Text('explicit-request-actions'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(Chat), findsOneWidget);
      final chat = tester.widget<Chat>(find.byType(Chat));
      expect(chat.currentUserId, 'me');
      expect(chat.crossCache, isNotNull);
      await expectLater(
        chat.crossCache!.downloadAndSave(
          'https://external.test/private-target',
        ),
        throwsStateError,
      );
      expect(
        find.text('private-bubble https://external.test/never-fetch'),
        findsOneWidget,
      );
      expect(find.text('Message deleted'), findsOneWidget);
      expect(find.text('explicit-request-actions'), findsOneWidget);
      expect(
        find.byWidgetPredicate(
          (widget) => widget is EditableText && !widget.readOnly,
        ),
        findsNothing,
      );
      expect(tester.takeException(), isNull);
    },
  );
}
