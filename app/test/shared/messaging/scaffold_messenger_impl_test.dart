import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Future<
  ({GlobalKey<ScaffoldMessengerState> key, ScaffoldMessengerImpl impl})
>
_pumpHarness(WidgetTester tester) async {
  final key = GlobalKey<ScaffoldMessengerState>();
  final impl = ScaffoldMessengerImpl(key);

  await tester.pumpWidget(
    MaterialApp(
      scaffoldMessengerKey: key,
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: const Scaffold(body: SizedBox()),
    ),
  );

  return (key: key, impl: impl);
}

void main() {
  group('ScaffoldMessengerImpl', () {
    testWidgets('info shows a SnackBar with a 4-second duration', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.info('Hello');
      await tester.pump(); // schedule the snackbar

      final snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
      expect(snackBar.duration, const Duration(seconds: 4));
      expect(find.text('Hello'), findsOneWidget);
      expect(find.byIcon(Icons.info_outline), findsOneWidget);
      expect(find.byIcon(Icons.close), findsNothing);
    });

    testWidgets(
      'warning shows a SnackBar with sticky duration + close icon',
      (tester) async {
        final h = await _pumpHarness(tester);
        h.impl.warning('Watch out');
        await tester.pump();

        final snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
        expect(snackBar.duration, const Duration(days: 365));
        expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
        expect(find.byIcon(Icons.close), findsOneWidget);
      },
    );

    testWidgets('error shows a SnackBar with sticky duration + close icon', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.error('Boom');
      await tester.pump();

      final snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
      expect(snackBar.duration, const Duration(days: 365));
      expect(find.byIcon(Icons.error_outline), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);
    });

    testWidgets('a second call replaces the first (always-replace policy)', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);

      h.impl.info('First');
      await tester.pump();
      expect(find.text('First'), findsOneWidget);

      h.impl.error('Second');
      await tester.pump();
      // The first should be gone; the second is now showing.
      expect(find.text('First'), findsNothing);
      expect(find.text('Second'), findsOneWidget);
      // Exactly one SnackBar is on screen.
      expect(find.byType(SnackBar), findsOneWidget);
    });

    testWidgets('action onPressed runs and dismisses by default', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      var taps = 0;
      h.impl.error(
        'Boom',
        action: MessageAction(label: 'Retry', onPressed: () => taps++),
      );
      // Floating SnackBar wraps its content in IgnorePointer during the
      // slide-in entrance animation (~250ms). Without advancing past it,
      // tester.tap() hits the IgnorePointer layer and the action's
      // onPressed never fires. 300ms covers the animation with margin.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

      await tester.tap(find.widgetWithText(TextButton, 'Retry'));
      // Drive the dismiss animation through.
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(taps, 1);
      expect(find.byType(SnackBar), findsNothing);
    });

    testWidgets(
      'action with dismissOnTap: false leaves the snackbar in place',
      (tester) async {
        final h = await _pumpHarness(tester);
        h.impl.error(
          'Boom',
          action: MessageAction(
            label: 'Retry',
            onPressed: () {},
            dismissOnTap: false,
          ),
        );
        // Advance past the entrance IgnorePointer (see first test for details).
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));

        await tester.tap(find.widgetWithText(TextButton, 'Retry'));
        await tester.pump();

        // The SnackBar should still be visible.
        expect(find.byType(SnackBar), findsOneWidget);
        expect(find.text('Boom'), findsOneWidget);
      },
    );

    testWidgets('tapping the close icon dismisses the message', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.error('Boom');
      // Advance past the entrance IgnorePointer (see first test for details).
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));

      await tester.tap(find.byIcon(Icons.close));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(SnackBar), findsNothing);
    });

    testWidgets('dismiss() hides the current message', (tester) async {
      final h = await _pumpHarness(tester);
      h.impl.error('Boom');
      await tester.pump();
      expect(find.byType(SnackBar), findsOneWidget);

      h.impl.dismiss();
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(SnackBar), findsNothing);
    });

    testWidgets(
      'SnackBar background matches the severity surface from the theme',
      (tester) async {
        final h = await _pumpHarness(tester);

        // info → infoSurface
        h.impl.info('Hello');
        await tester.pump();
        var snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
        expect(snackBar.backgroundColor, BrandColors.cobaltSoft);

        // warning → warningSurface
        h.impl.warning('Watch out');
        await tester.pump();
        snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
        expect(snackBar.backgroundColor, BrandColors.butter);

        // error → errorSurface
        h.impl.error('Boom');
        await tester.pump();
        snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
        expect(snackBar.backgroundColor, BrandColors.redSoft);
      },
    );
  });
}
