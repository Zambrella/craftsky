import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/moderation/widgets/report_subject_sheet.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pump(
  WidgetTester tester,
  Widget child, {
  EdgeInsets viewInsets = EdgeInsets.zero,
}) {
  return tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        body: Builder(
          builder: (context) => MediaQuery(
            data: MediaQuery.of(context).copyWith(viewInsets: viewInsets),
            child: child,
          ),
        ),
      ),
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
            .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Submit'))
            .onPressed,
        isNull,
      );

      await tester.tap(find.text('Spam'));
      await tester.pump();
      expect(
        tester
            .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Submit'))
            .onPressed,
        isNotNull,
      );

      await tester.ensureVisible(find.widgetWithText(ChunkyButton, 'Submit'));
      await tester.tap(find.widgetWithText(ChunkyButton, 'Submit'));
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
            .widget<ChunkyButton>(find.widgetWithText(ChunkyButton, 'Submit'))
            .onPressed,
        isNull,
      );
    });

    testWidgets('reason list does not paint a filled input background', (
      tester,
    ) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (_) {},
        ),
      );

      final group = tester.widget<FormBuilderRadioGroup<ReportReason>>(
        find.byType(FormBuilderRadioGroup<ReportReason>),
      );

      expect(group.decoration.filled, isFalse);
    });

    testWidgets('details input uses the shared brand text field', (
      tester,
    ) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.profile,
          onSubmit: (_) async {},
        ),
      );

      expect(
        find.widgetWithText(BrandTextField, 'Details (optional)'),
        findsOneWidget,
      );
    });

    testWidgets('submit uses chunky primary button with loading state', (
      tester,
    ) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          isSubmitting: true,
          onSubmit: (_) {},
        ),
      );

      expect(find.byType(ChunkyButton), findsOneWidget);
      expect(find.byType(StitchProgressIndicator), findsOneWidget);
      expect(find.text('Submitting…'), findsNothing);
    });

    testWidgets('external submit error keeps input available for retry', (
      tester,
    ) async {
      var calls = 0;
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (_) => calls++,
        ),
      );

      await tester.tap(find.text('Spam'));
      await tester.enterText(find.byType(TextField), 'private details');
      await tester.pump();
      await tester.ensureVisible(find.widgetWithText(ChunkyButton, 'Submit'));
      await tester.tap(find.widgetWithText(ChunkyButton, 'Submit'));
      await tester.pumpAndSettle();

      expect(calls, 1);

      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          submitError: "Couldn't submit report. Please try again.",
          onSubmit: (_) => calls++,
        ),
      );

      expect(
        find.text("Couldn't submit report. Please try again."),
        findsOneWidget,
      );
      expect(find.text('private details'), findsOneWidget);

      await tester.tap(find.widgetWithText(ChunkyButton, 'Submit'));
      await tester.pumpAndSettle();

      expect(calls, 2);

      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (_) => calls++,
        ),
      );

      expect(
        find.text("Couldn't submit report. Please try again."),
        findsNothing,
      );
    });

    testWidgets('adds keyboard inset below scrollable content', (tester) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (_) async {},
        ),
        viewInsets: const EdgeInsets.only(bottom: 300),
      );

      final scrollView = tester
          .widgetList<SingleChildScrollView>(
            find.byKey(const ValueKey('reportSubjectSheetScrollView')),
          )
          .single;
      final padding = scrollView.padding! as EdgeInsets;

      expect(padding.left, 16);
      expect(padding.top, 16);
      expect(padding.right, 16);
      expect(padding.bottom, 316);
    });
  });
}
