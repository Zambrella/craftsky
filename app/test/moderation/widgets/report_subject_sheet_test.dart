import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/moderation/widgets/report_subject_sheet.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pump(WidgetTester tester, Widget child) {
  return tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  group('ReportSubjectSheet', () {
    testWidgets('lists approved reasons and blocks submit until valid', (
      tester,
    ) async {
      ReportSubmission? submitted;
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (submission) async => submitted = submission,
        ),
      );

      expect(find.text('Report post'), findsOneWidget);
      expect(find.text('Spam'), findsOneWidget);
      expect(find.text('Other'), findsOneWidget);
      expect(
        tester
            .widget<FilledButton>(find.widgetWithText(FilledButton, 'Submit'))
            .onPressed,
        isNull,
      );

      await tester.tap(find.text('Spam'));
      await tester.pump();
      expect(
        tester
            .widget<FilledButton>(find.widgetWithText(FilledButton, 'Submit'))
            .onPressed,
        isNotNull,
      );

      await tester.ensureVisible(find.widgetWithText(FilledButton, 'Submit'));
      await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
      await tester.pump();

      expect(submitted?.reasonType, ReportReason.spam.reasonType);
      expect(submitted?.details, isNull);
    });

    testWidgets('details over 1000 characters disables submit', (tester) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.profile,
          onSubmit: (_) async {},
        ),
      );

      await tester.tap(find.text('Other'));
      await tester.enterText(find.byType(TextField), 'x' * 1001);
      await tester.pump();

      expect(
        find.text('Details must be 1000 characters or fewer.'),
        findsOneWidget,
      );
      expect(
        tester
            .widget<FilledButton>(find.widgetWithText(FilledButton, 'Submit'))
            .onPressed,
        isNull,
      );
    });

    testWidgets('failed submit keeps input available for retry', (
      tester,
    ) async {
      var calls = 0;
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (_) async {
            calls++;
            if (calls == 1) throw Exception('network failed');
          },
        ),
      );

      await tester.tap(find.text('Spam'));
      await tester.enterText(find.byType(TextField), 'private details');
      await tester.pump();
      await tester.ensureVisible(find.widgetWithText(FilledButton, 'Submit'));
      await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
      await tester.pumpAndSettle();

      expect(calls, 1);
      expect(
        find.text("Couldn't submit report. Please try again."),
        findsOneWidget,
      );
      expect(find.text('private details'), findsOneWidget);

      await tester.tap(find.widgetWithText(FilledButton, 'Submit'));
      await tester.pumpAndSettle();

      expect(calls, 2);
      expect(
        find.text("Couldn't submit report. Please try again."),
        findsNothing,
      );
    });
  });
}
