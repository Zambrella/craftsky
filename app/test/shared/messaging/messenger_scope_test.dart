import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

class _RecordingMessenger implements AppMessenger {
  String? lastMethod;
  String? lastMessage;

  @override
  void info(String message, {MessageAction? action}) {
    lastMethod = 'info';
    lastMessage = message;
  }

  @override
  void warning(String message, {MessageAction? action}) {
    lastMethod = 'warning';
    lastMessage = message;
  }

  @override
  void error(String message, {MessageAction? action}) {
    lastMethod = 'error';
    lastMessage = message;
  }

  @override
  void dismiss() {
    lastMethod = 'dismiss';
  }
}

void main() {
  testWidgets(
    'MessengerScope.of returns the messenger from the nearest scope',
    (tester) async {
      final messenger = _RecordingMessenger();
      AppMessenger? resolved;

      await tester.pumpWidget(
        MessengerScope(
          messenger: messenger,
          child: Builder(
            builder: (context) {
              resolved = MessengerScope.of(context);
              return const SizedBox();
            },
          ),
        ),
      );

      expect(resolved, same(messenger));
    },
  );

  testWidgets('MessengerScope.of asserts when no scope is present', (
    tester,
  ) async {
    await tester.pumpWidget(
      Builder(
        builder: (context) {
          expect(() => MessengerScope.of(context), throwsAssertionError);
          return const SizedBox();
        },
      ),
    );
  });

  testWidgets(
    'updateShouldNotify is true when the messenger reference changes',
    (tester) async {
      final messengerA = _RecordingMessenger();
      final messengerB = _RecordingMessenger();
      var rebuildCount = 0;

      Widget build(AppMessenger m) => MessengerScope(
        messenger: m,
        child: Builder(
          builder: (context) {
            MessengerScope.of(context);
            rebuildCount++;
            return const SizedBox();
          },
        ),
      );

      await tester.pumpWidget(build(messengerA));
      expect(rebuildCount, 1);

      await tester.pumpWidget(build(messengerB));
      expect(rebuildCount, 2);
    },
  );
}
