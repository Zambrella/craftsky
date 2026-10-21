import 'package:craftsky_app/chat/widgets/chat_message_effect.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'alerts use the existing floating snackbar without moving the page',
    (tester) async {
      final key = GlobalKey<ScaffoldMessengerState>();
      final messenger = ScaffoldMessengerImpl(key);
      var retries = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          scaffoldMessengerKey: key,
          builder: (context, child) =>
              MessengerScope(messenger: messenger, child: child!),
          home: ChatMessageEffect(
            message: 'Messages may be out of date.',
            severity: MessageSeverity.warning,
            action: MessageAction(label: 'Refresh', onPressed: () => retries++),
            child: Scaffold(
              appBar: AppBar(title: const Text('Messages')),
              body: const Text('Inbox'),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.getTopLeft(find.byType(AppBar)), Offset.zero);
      expect(find.byType(SnackBar), findsOneWidget);
      expect(find.byType(CraftskySnackBarContent), findsOneWidget);
      expect(
        tester.widget<SnackBar>(find.byType(SnackBar)).behavior,
        SnackBarBehavior.floating,
      );
      await tester.tap(find.text('Refresh'));
      await tester.pumpAndSettle();
      expect(retries, 1);
      expect(find.byType(SnackBar), findsNothing);
      expect(tester.getTopLeft(find.byType(AppBar)), Offset.zero);
    },
  );

  testWidgets(
    'rebuilds deduplicate alerts and disposed pages cannot retry '
    'or dismiss another message',
    (tester) async {
      final messenger = RecordingMessenger();
      late StateSetter update;
      var message = 'Failed to save.';
      var enabled = true;
      var visible = true;
      var retries = 0;
      await tester.pumpWidget(
        MaterialApp(
          builder: (context, child) =>
              MessengerScope(messenger: messenger, child: child!),
          home: StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return visible
                  ? ChatMessageEffect(
                      message: message,
                      severity: MessageSeverity.error,
                      actionEnabled: enabled,
                      action: MessageAction(
                        label: 'Retry',
                        onPressed: () => retries++,
                      ),
                      child: const Scaffold(body: Text('Draft')),
                    )
                  : const Scaffold(body: Text('Other account'));
            },
          ),
        ),
      );
      await tester.pump();
      expect(messenger.calls.single.$1, 'error');
      update(() {});
      await tester.pump();
      expect(messenger.calls, hasLength(1));
      final retry = messenger.calls.single.$3!;
      update(() => enabled = false);
      await tester.pump();
      retry.onPressed();
      expect(retries, 0);
      update(() {
        enabled = true;
        message = 'Another failure.';
      });
      await tester.pump();
      expect(messenger.calls, hasLength(2));
      retry.onPressed();
      expect(retries, 1);
      messenger.info('Unrelated notification');
      update(() => visible = false);
      await tester.pump();
      retry.onPressed();
      expect(retries, 1);
      expect(messenger.calls.last.$2, 'Unrelated notification');
      expect(messenger.calls.where((call) => call.$1 == 'dismiss'), isEmpty);
    },
  );

  testWidgets(
    'covered routes do not publish or invoke hidden message actions',
    (tester) async {
      final messenger = RecordingMessenger();
      final navigator = GlobalKey<NavigatorState>();
      late StateSetter update;
      String? message;
      var retries = 0;
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: navigator,
          builder: (context, child) =>
              MessengerScope(messenger: messenger, child: child!),
          home: StatefulBuilder(
            builder: (context, setState) {
              update = setState;
              return ChatMessageEffect(
                message: message,
                severity: MessageSeverity.warning,
                action: MessageAction(
                  label: 'Retry',
                  onPressed: () => retries++,
                ),
                child: const Scaffold(body: Text('Inbox')),
              );
            },
          ),
        ),
      );
      navigator.currentState!.push<void>(
        MaterialPageRoute(
          builder: (_) => const Scaffold(body: Text('Settings')),
        ),
      );
      await tester.pumpAndSettle();
      update(() => message = 'Connection interrupted.');
      await tester.pump();
      expect(messenger.calls, isEmpty);
      navigator.currentState!.pop();
      await tester.pumpAndSettle();
      expect(messenger.calls.single.$1, 'warning');
      navigator.currentState!.push<void>(
        MaterialPageRoute(
          builder: (_) => const Scaffold(body: Text('Settings')),
        ),
      );
      await tester.pumpAndSettle();
      messenger.calls.single.$3!.onPressed();
      expect(retries, 0);
    },
  );
}
