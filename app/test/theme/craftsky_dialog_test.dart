import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget pumpHarness(Widget child) {
    return MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: Center(child: child)),
    );
  }

  group('CraftskyDialog', () {
    testWidgets('renders title, body, and actions', (tester) async {
      await tester.pumpWidget(
        pumpHarness(
          const CraftskyDialog(
            title: 'A title',
            body: Text('A body'),
            actions: [Text('Action one'), Text('Action two')],
          ),
        ),
      );

      expect(find.text('A title'), findsOneWidget);
      expect(find.text('A body'), findsOneWidget);
      expect(find.text('Action one'), findsOneWidget);
      expect(find.text('Action two'), findsOneWidget);
    });
  });

  group('showCraftskyConfirmDialog', () {
    testWidgets('returns true when confirm tapped', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'Discard?',
                    message: 'Your changes will be lost.',
                    confirmLabel: 'Discard',
                    cancelLabel: 'Keep editing',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      expect(find.text('Discard?'), findsOneWidget);
      expect(find.text('Your changes will be lost.'), findsOneWidget);
      expect(find.text('Discard'), findsOneWidget);
      expect(find.text('Keep editing'), findsOneWidget);

      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(await resultFuture, isTrue);
      expect(find.byType(CraftskyDialog), findsNothing);
    });

    testWidgets('returns false when cancel tapped', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'Discard?',
                    message: 'Your changes will be lost.',
                    confirmLabel: 'Discard',
                    cancelLabel: 'Keep editing',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();

      expect(await resultFuture, isFalse);
    });

    testWidgets('returns false when barrier tapped', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'Discard?',
                    message: 'Your changes will be lost.',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tapAt(const Offset(10, 10));
      await tester.pumpAndSettle();

      expect(await resultFuture, isFalse);
    });

    testWidgets('falls back to localized labels when none given', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => showCraftskyConfirmDialog(
                  context,
                  title: 'T',
                  message: 'M',
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      expect(find.text('Confirm'), findsOneWidget);
      expect(find.text('Cancel'), findsOneWidget);
    });
  });
}
