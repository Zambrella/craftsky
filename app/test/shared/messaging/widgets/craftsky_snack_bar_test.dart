import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _wrap(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

void main() {
  group('CraftskySnackBarContent', () {
    testWidgets(
      'info renders info_outline icon and no close icon',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            const CraftskySnackBarContent(
              severity: MessageSeverity.info,
              message: 'Saved',
            ),
          ),
        );

        expect(find.byIcon(Icons.info_outline), findsOneWidget);
        expect(find.byIcon(Icons.close), findsNothing);
        expect(find.text('Saved'), findsOneWidget);
      },
    );

    testWidgets(
      'warning renders warning_amber_rounded and a close icon',
      (tester) async {
        var dismissed = false;
        await tester.pumpWidget(
          _wrap(
            CraftskySnackBarContent(
              severity: MessageSeverity.warning,
              message: 'Hold up',
              onDismiss: () => dismissed = true,
            ),
          ),
        );

        expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
        expect(find.byIcon(Icons.close), findsOneWidget);

        await tester.tap(find.byIcon(Icons.close));
        expect(dismissed, isTrue);
      },
    );

    testWidgets(
      'error renders error_outline and a close icon',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            CraftskySnackBarContent(
              severity: MessageSeverity.error,
              message: 'Boom',
              onDismiss: () {},
            ),
          ),
        );

        expect(find.byIcon(Icons.error_outline), findsOneWidget);
        expect(find.byIcon(Icons.close), findsOneWidget);
      },
    );

    testWidgets(
      'renders an action button when MessageAction is supplied',
      (tester) async {
        var actionTaps = 0;
        final action = MessageAction(
          label: 'Retry',
          onPressed: () => actionTaps++,
        );

        await tester.pumpWidget(
          _wrap(
            CraftskySnackBarContent(
              severity: MessageSeverity.error,
              message: 'Boom',
              action: action,
              onDismiss: () {},
            ),
          ),
        );

        expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

        await tester.tap(find.widgetWithText(TextButton, 'Retry'));
        expect(actionTaps, 1);
      },
    );

    testWidgets(
      'info with no action and no onDismiss renders neither button',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            const CraftskySnackBarContent(
              severity: MessageSeverity.info,
              message: 'Saved',
            ),
          ),
        );

        expect(find.byType(TextButton), findsNothing);
        expect(find.byIcon(Icons.close), findsNothing);
      },
    );
  });
}
