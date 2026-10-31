import 'dart:async';

import 'package:craftsky_app/chat/adapters/flutter_chat_adapter.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_chat_ui/flutter_chat_ui.dart' as ui;
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-010 time sections are centred and break at days and five minutes',
    (tester) async {
      final now = DateTime.now();
      final today = DateTime(now.year, now.month, now.day);
      final times = [
        today.subtract(const Duration(minutes: 2)),
        today.add(const Duration(minutes: 1)),
        today.add(const Duration(minutes: 5)),
        today.add(const Duration(minutes: 10)),
        today.add(const Duration(minutes: 14)),
      ];
      final history = const ChatHistory.empty().merge(
        ChatMessagePage.fromMap({
          'items': [
            for (var i = 0; i < times.length; i++)
              {
                'id': 'timed-$i',
                'revision': 'r',
                'kind': 'text',
                'text': 'Text $i',
                'senderDid': i < 2 ? 'did:plc:peer' : 'did:plc:owner',
                'sentAt': times[i].toUtc().toIso8601String(),
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
              composer: const SizedBox.shrink(),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('Yesterday at'), findsOneWidget);
      expect(find.textContaining('Today at'), findsNWidgets(2));
      expect(
        tester
            .widgetList<ui.ChatMessage>(find.byType(ui.ChatMessage))
            .where((row) => row.headerWidget != null),
        hasLength(3),
      );
      for (final element in find.textContaining(' at ').evaluate()) {
        expect(
          tester.getCenter(find.byWidget(element.widget)).dx,
          closeTo(
            tester.view.physicalSize.width / tester.view.devicePixelRatio / 2,
            1,
          ),
        );
      }
      for (final bubble in tester.widgetList<ui.SimpleTextMessage>(
        find.byType(ui.SimpleTextMessage),
      )) {
        expect(bubble.showTime, false);
      }
      final groupedGap =
          tester.getTopLeft(find.text('Text 4')).dy -
          tester.getBottomLeft(find.text('Text 3')).dy;
      final sectionGap =
          tester.getTopLeft(find.text('Text 3')).dy -
          tester.getBottomLeft(find.text('Text 2')).dy;
      expect(groupedGap, lessThan(sectionGap));
      expect(find.byType(ui.TimeAndStatus), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets(
    'UT-010 prepending history moves the section header without duplicates',
    (tester) async {
      final date = DateTime(2026, 10, 31, 9);
      ChatMessagePage page(List<int> minutes) => ChatMessagePage.fromMap({
        'items': [
          for (final minute in minutes)
            {
              'id': 'minute-$minute',
              'revision': 'r',
              'kind': 'text',
              'text': 'Message $minute',
              'senderDid': 'did:plc:peer',
              'sentAt': date
                  .add(Duration(minutes: minute))
                  .toUtc()
                  .toIso8601String(),
            },
        ],
      });
      var history = const ChatHistory.empty().merge(page([4, 8]));
      late StateSetter update;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: StatefulBuilder(
              builder: (_, setState) {
                update = setState;
                return FlutterChatAdapter(
                  history: history,
                  owner: Did.parse('did:plc:owner'),
                  composer: const SizedBox.shrink(),
                );
              },
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('9:04'), findsOneWidget);
      update(() => history = history.merge(page([0])));
      await tester.pumpAndSettle();
      expect(find.textContaining('9:04'), findsNothing);
      expect(find.textContaining('9:00'), findsOneWidget);
      expect(
        tester
            .widgetList<ui.ChatMessage>(find.byType(ui.ChatMessage))
            .where((row) => row.headerWidget != null),
        hasLength(1),
      );
      update(() => history = const ChatHistory.empty().merge(page([4, 8])));
      await tester.pumpAndSettle();
      expect(find.textContaining('9:04'), findsOneWidget);
      expect(find.textContaining('9:00'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets('sent and received bubbles reserve opposite-side insets', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(360, 640));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final sent = 'Sent ${'long message ' * 20}';
    final received = 'Received ${'long message ' * 20}';
    final history = const ChatHistory.empty().merge(
      ChatMessagePage.fromMap({
        'items': [
          for (final own in [false, true])
            {
              'id': own ? 'sent' : 'received',
              'revision': 'r',
              'kind': 'text',
              'text': own ? sent : received,
              'senderDid': own ? 'did:plc:owner' : 'did:plc:peer',
              'sentAt': own ? '2026-10-30T12:01:00Z' : '2026-10-30T12:00:00Z',
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
            composer: const SizedBox.shrink(),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(ui.SimpleTextMessage), findsNWidgets(2));
    expect(find.byType(ui.ChatMessage), findsNWidgets(2));
    expect(find.byType(ui.TimeAndStatus), findsNothing);
    for (final bubble in tester.widgetList<ui.SimpleTextMessage>(
      find.byType(ui.SimpleTextMessage),
    )) {
      expect(bubble.showTime, isFalse);
      expect(bubble.showStatus, isFalse);
    }
    Rect bubble(String text) => tester.getRect(
      find
          .ancestor(
            of: find.text(text),
            matching: find.byType(ui.SimpleTextMessage),
          )
          .first,
    );
    expect(bubble(sent).left, greaterThan(bubble(received).left));
    expect(bubble(received).right, lessThan(bubble(sent).right));
    expect(tester.takeException(), isNull);
  });
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
      // Settle the drag boundary spring before measuring the loaded rows.
      await tester.pump(const Duration(seconds: 1));
      // The first header adds height when its sliver is laid out. Scroll the
      // remaining distance so this test reaches the actual pagination boundary.
      await tester.drag(find.byType(CustomScrollView), const Offset(0, 150));
      await tester.pumpAndSettle();
      expect(loads, 1);
      // Loading feedback is in the page AppBar, never in history geometry.
      expect(find.byType(StitchProgressIndicator), findsNothing);
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
              messageFooterBuilder: (_) => const Text('guarded-preview'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ui.Chat), findsOneWidget);
      final chat = tester.widget<ui.Chat>(find.byType(ui.Chat));
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
      expect(find.text('guarded-preview'), findsOneWidget);
      expect(find.byType(ui.SimpleTextMessage), findsNWidgets(2));
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
