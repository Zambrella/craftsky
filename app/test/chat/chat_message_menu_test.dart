import 'package:craftsky_app/chat/adapters/flutter_chat_adapter.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

const _peerText = 'Received https://example.test/private-path?q=private-query';
final _owner = Did.parse('did:plc:owner');

ChatHistory _history() => const ChatHistory.empty().merge(
  ChatMessagePage.fromMap({
    'items': [
      for (final kind in ['peer', 'owned', 'deleted'])
        {
          'id': kind,
          'revision': 'r1',
          'kind': kind == 'deleted' ? 'deleted' : 'text',
          if (kind != 'deleted') 'text': kind == 'peer' ? _peerText : 'Owned',
          'senderDid': kind == 'owned' ? _owner.value : 'did:plc:peer',
          'sentAt': '2026-10-30T12:00:00Z',
        },
    ],
  }),
);

Widget _app(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: appLocalizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

void main() {
  for (final secondary in [false, true]) {
    testWidgets(
      'UT-013 message menu opens with '
      '${secondary ? 'secondary click' : 'long press'}',
      (tester) async {
        if (secondary) {
          await tester.binding.setSurfaceSize(const Size(1200, 800));
          addTearDown(() => tester.binding.setSurfaceSize(null));
        }
        Uri? opened;
        ChatMessageId? reported;
        ChatMessageId? deleted;
        String? copied;
        tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          SystemChannels.platform,
          (call) async {
            if (call.method == 'Clipboard.setData') {
              copied = (call.arguments as Map)['text'] as String;
            }
            return null;
          },
        );
        addTearDown(
          () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
            SystemChannels.platform,
            null,
          ),
        );
        await tester.pumpWidget(
          _app(
            FlutterChatAdapter(
              owner: _owner,
              history: _history(),
              composer: const SizedBox.shrink(),
              onLink: (uri) => opened = uri,
              onReport: (id) => reported = id,
              onDelete: (id) => deleted = id,
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(opened, isNull);
        expect(copied, isNull);
        Future<void> menu(String text) async {
          if (secondary) {
            await tester.tap(find.text(text), buttons: kSecondaryMouseButton);
          } else {
            await tester.longPress(find.text(text));
          }
          await tester.pumpAndSettle();
        }

        await menu(_peerText);
        expect(find.text('Delete for me'), findsNothing);
        expect(reported, isNull);
        expect(find.text('Report message'), findsOneWidget);
        expect(
          tester.widget<Text>(find.text('Report message')).style?.color,
          AppTheme.lightThemeData.extension<SemanticColorsTheme>()!.error,
        );
        expect(find.text('Open link'), findsOneWidget);
        await tester.tap(find.text('Open link'));
        await tester.pumpAndSettle();
        expect(
          opened.toString(),
          'https://example.test/private-path?q=private-query',
        );
        expect(reported, isNull);
        await menu(_peerText);
        await tester.tap(find.text('Report message'));
        await tester.pumpAndSettle();
        expect(reported?.value, 'peer');
        await menu('Owned');
        expect(deleted, isNull);
        expect(find.text('Delete for me'), findsOneWidget);
        expect(
          tester.widget<Text>(find.text('Delete for me')).style?.color,
          AppTheme.lightThemeData.extension<SemanticColorsTheme>()!.error,
        );
        await tester.tap(find.text('Delete for me'));
        await tester.pumpAndSettle();
        expect(deleted?.value, 'owned');
        await menu('Owned');
        expect(find.text('Report message'), findsNothing);
        expect(find.text('Open link'), findsNothing);
        await tester.tap(find.text('Copy'));
        await tester.pumpAndSettle();
        expect(copied, 'Owned');
        await menu('Message deleted');
        expect(find.text('Delete for me'), findsNothing);
        expect(find.text('Copy'), findsNothing);
        expect(find.text('Report message'), findsNothing);
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('UT-013 message menu supports keyboard context actions', (
    tester,
  ) async {
    ChatMessageId? reported;
    await tester.pumpWidget(
      _app(
        FlutterChatAdapter(
          owner: _owner,
          history: _history(),
          composer: const SizedBox.shrink(),
          onReport: (id) => reported = id,
        ),
      ),
    );
    await tester.pumpAndSettle();
    Focus.of(tester.element(find.text(_peerText))).requestFocus();
    await tester.pump();
    await tester.sendKeyDownEvent(LogicalKeyboardKey.shiftLeft);
    await tester.sendKeyEvent(LogicalKeyboardKey.f10);
    await tester.sendKeyUpEvent(LogicalKeyboardKey.shiftLeft);
    await tester.pumpAndSettle();
    expect(find.text('Report message'), findsOneWidget);
    expect(reported, isNull);
    await tester.tap(find.text('Report message'));
    await tester.pumpAndSettle();
    expect(reported?.value, 'peer');
    expect(tester.takeException(), isNull);
  });

  for (final deleting in [false, true]) {
    for (final change in ['owner', 'history', 'revision', 'disposal']) {
      testWidgets(
        'UT-013 discards $change changes while a message menu is open '
        'deleting=$deleting',
        (tester) async {
          var owner = _owner;
          var history = _history();
          var visible = true;
          ChatMessageId? reported;
          late StateSetter update;
          await tester.pumpWidget(
            _app(
              StatefulBuilder(
                builder: (context, setState) {
                  update = setState;
                  return visible
                      ? FlutterChatAdapter(
                          owner: owner,
                          history: history,
                          composer: const SizedBox.shrink(),
                          onReport: (id) => reported = id,
                          onDelete: (id) => reported = id,
                        )
                      : const SizedBox.shrink();
                },
              ),
            ),
          );
          await tester.pumpAndSettle();
          final label = deleting ? 'Delete for me' : 'Report message';
          await tester.longPress(find.text(deleting ? 'Owned' : _peerText));
          await tester.pumpAndSettle();
          expect(find.text(label), findsOneWidget);
          update(() {
            switch (change) {
              case 'owner':
                owner = Did.parse('did:plc:other');
              case 'history':
                history = const ChatHistory.empty();
              case 'revision':
                history = const ChatHistory.empty().merge(
                  ChatMessagePage.fromMap({
                    'items': [
                      {
                        'id': deleting ? 'owned' : 'peer',
                        'revision': 'r2',
                        'kind': 'text',
                        'text': deleting ? 'Owned' : _peerText,
                        'senderDid': deleting ? _owner.value : 'did:plc:peer',
                        'sentAt': '2026-10-30T12:00:00Z',
                      },
                    ],
                  }),
                );
              case 'disposal':
                visible = false;
            }
          });
          await tester.pumpAndSettle();
          await tester.tap(find.text(label));
          await tester.pumpAndSettle();
          expect(reported, isNull);
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
}
