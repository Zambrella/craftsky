import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

Future<void> _pumpUnderScope(
  WidgetTester tester,
  RecordingMessenger messenger,
  void Function(BuildContext context) onContext,
) async {
  await tester.pumpWidget(
    MessengerScope(
      messenger: messenger,
      child: MaterialApp(
        home: Builder(
          builder: (context) {
            onContext(context);
            return const SizedBox();
          },
        ),
      ),
    ),
  );
}

void main() {
  group('AppMessengerX', () {
    testWidgets('showInfo routes to messenger.info', (tester) async {
      final messenger = RecordingMessenger();
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.showInfo('Saved'),
      );
      expect(messenger.calls, [('info', 'Saved', null)]);
    });

    testWidgets('showWarning routes to messenger.warning', (tester) async {
      final messenger = RecordingMessenger();
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.showWarning('Heads up'),
      );
      expect(messenger.calls, [('warning', 'Heads up', null)]);
    });

    testWidgets(
      'showError routes to messenger.error and forwards action',
      (tester) async {
        final messenger = RecordingMessenger();
        final action = MessageAction(label: 'Retry', onPressed: () {});
        await _pumpUnderScope(
          tester,
          messenger,
          (c) => c.showError('Boom', action: action),
        );
        expect(messenger.calls.length, 1);
        expect(messenger.calls.first.$1, 'error');
        expect(messenger.calls.first.$2, 'Boom');
        expect(messenger.calls.first.$3, same(action));
      },
    );

    testWidgets('dismissMessage routes to messenger.dismiss', (tester) async {
      final messenger = RecordingMessenger();
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.dismissMessage(),
      );
      expect(messenger.calls, [('dismiss', '', null)]);
    });
  });
}
