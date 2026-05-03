import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
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

  group('showCraftskyConfirmDialog (async)', () {
    testWidgets(
      'shows spinner during onConfirm and pops with true on success',
      (
        tester,
      ) async {
        final completer = Completer<void>();
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
                      title: 'T',
                      message: 'M',
                      confirmLabel: 'Yes',
                      cancelLabel: 'No',
                      onConfirm: () => completer.future,
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

        await tester.tap(find.text('Yes'));
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 50));

        expect(find.byType(CircularProgressIndicator), findsOneWidget);
        expect(find.text('Yes'), findsNothing);

        final cancel = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'No'),
        );
        expect(cancel.onPressed, isNull);

        completer.complete();
        await tester.pumpAndSettle();

        expect(await resultFuture, isTrue);
        expect(find.byType(CraftskyDialog), findsNothing);
      },
    );

    testWidgets(
      'keeps dialog open with re-enabled buttons when onConfirm throws',
      (tester) async {
        final caughtInOnConfirm = <Object>[];

        await tester.pumpWidget(
          MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => Scaffold(
                body: TextButton(
                  onPressed: () {
                    showCraftskyConfirmDialog(
                      context,
                      title: 'T',
                      message: 'M',
                      confirmLabel: 'Yes',
                      cancelLabel: 'No',
                      onConfirm: () async {
                        await Future<void>.delayed(
                          const Duration(milliseconds: 10),
                        );
                        try {
                          throw StateError('nope');
                        } catch (e) {
                          caughtInOnConfirm.add(e);
                          rethrow;
                        }
                      },
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

        await tester.tap(find.text('Yes'));
        await tester.pump(const Duration(milliseconds: 50));
        await tester.pumpAndSettle();

        // Caller's onConfirm did throw — proves the failure path ran.
        expect(caughtInOnConfirm, hasLength(1));
        expect(caughtInOnConfirm.first, isA<StateError>());

        // Dialog stayed mounted with spinner gone and buttons re-enabled.
        expect(find.byType(CraftskyDialog), findsOneWidget);
        expect(find.byType(CircularProgressIndicator), findsNothing);
        expect(find.text('Yes'), findsOneWidget);

        final cancel = tester.widget<TextButton>(
          find.widgetWithText(TextButton, 'No'),
        );
        expect(cancel.onPressed, isNotNull);
      },
    );

    testWidgets('destructive helper paints primary surface red', (
      tester,
    ) async {
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
                  resultFuture = showCraftskyDestructiveConfirmDialog(
                    context,
                    title: 'Delete?',
                    message: 'This cannot be undone.',
                    confirmLabel: 'Delete',
                    cancelLabel: 'Cancel',
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

      final primary = tester.widget<ChunkyButton>(find.byType(ChunkyButton));
      expect(primary.backgroundColor, BrandColors.red);

      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(await resultFuture, isFalse);
    });

    testWidgets('barrier tap during async in flight is suppressed', (
      tester,
    ) async {
      final completer = Completer<void>();

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  showCraftskyConfirmDialog(
                    context,
                    title: 'T',
                    message: 'M',
                    confirmLabel: 'Yes',
                    cancelLabel: 'No',
                    onConfirm: () => completer.future,
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

      await tester.tap(find.text('Yes'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));

      await tester.tapAt(const Offset(10, 10));
      await tester.pump(const Duration(milliseconds: 50));
      expect(find.byType(CraftskyDialog), findsOneWidget);

      completer.complete();
      await tester.pumpAndSettle();
    });
  });

  group('showCraftskyAlertDialog', () {
    testWidgets('renders title, message, single dismiss button', (
      tester,
    ) async {
      late Future<void> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyAlertDialog(
                    context,
                    title: 'Saved',
                    message: 'Your changes are live.',
                    dismissLabel: 'Got it',
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

      expect(find.text('Saved'), findsOneWidget);
      expect(find.text('Your changes are live.'), findsOneWidget);
      expect(find.byType(ChunkyButton), findsOneWidget);
      expect(
        find.descendant(
          of: find.byType(CraftskyDialog),
          matching: find.byType(TextButton),
        ),
        findsNothing,
      );

      await tester.tap(find.text('Got it'));
      await tester.pumpAndSettle();

      await resultFuture;
      expect(find.byType(CraftskyDialog), findsNothing);
    });

    testWidgets('falls back to localized dismiss label', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => showCraftskyAlertDialog(
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

      expect(find.text('OK'), findsOneWidget);
    });
  });
}
