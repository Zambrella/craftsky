import 'package:craftsky_app/drafts/widgets/draft_close_dialog.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('uses the CraftSky dialog treatment for draft close choices', (
    tester,
  ) async {
    late Future<DraftCloseChoice> result;

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () {
                result = showDraftCloseDialog(
                  context,
                  existingDraft: false,
                  canSave: true,
                );
              },
              child: const Text('Open'),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    expect(find.byType(CraftskyDialog), findsOneWidget);
    expect(find.byType(AlertDialog), findsNothing);
    expect(find.text('Keep editing'), findsOneWidget);
    expect(find.text('Discard'), findsOneWidget);
    expect(
      find.widgetWithText(ChunkyButton, 'Save draft'),
      findsOneWidget,
    );

    await tester.tap(find.text('Save draft'));
    await tester.pumpAndSettle();

    expect(await result, DraftCloseChoice.save);
  });
}
