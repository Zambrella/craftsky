import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/report_reason.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/moderation/widgets/report_subject_sheet.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
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
      expect(find.text('Reason'), findsOneWidget);
      expect(find.text('Spam'), findsOneWidget);
      expect(find.text('Other'), findsOneWidget);
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Submit'))
            .onPressed,
        isNull,
      );

      await tester.tap(find.text('Spam'));
      await tester.pump();
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Submit'))
            .onPressed,
        isNotNull,
      );

      await tester.tap(find.widgetWithText(TextButton, 'Submit'));
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
            .widget<TextButton>(find.widgetWithText(TextButton, 'Submit'))
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
      expect(group.orientation, OptionsOrientation.vertical);
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
        find.widgetWithText(BrandTextField, 'Details'),
        findsOneWidget,
      );
      expect(find.text('0/1000'), findsOneWidget);
    });

    testWidgets('details input shows a live character count', (tester) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.profile,
          onSubmit: (_) async {},
        ),
      );

      await tester.enterText(find.byType(TextField), 'private details');
      await tester.pump();

      expect(find.text('15/1000'), findsOneWidget);
    });

    testWidgets('details input keeps focus while character count updates', (
      tester,
    ) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.profile,
          onSubmit: (_) async {},
        ),
      );

      final textField = find.byType(TextField);
      await tester.ensureVisible(textField);
      await tester.showKeyboard(textField);
      await tester.pump();
      expect(tester.widget<TextField>(textField).focusNode?.hasFocus, isTrue);

      tester.testTextInput.enterText('a');
      await tester.pump();

      expect(find.text('1/1000'), findsOneWidget);
      expect(tester.widget<TextField>(textField).focusNode?.hasFocus, isTrue);
    });

    testWidgets('submit uses app bar action with loading state', (
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

      expect(find.widgetWithText(TextButton, 'Submit'), findsNothing);
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
      await tester.tap(find.widgetWithText(TextButton, 'Submit'));
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

      await tester.tap(find.widgetWithText(TextButton, 'Submit'));
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

    testWidgets('uses full-screen route spacing for scrollable content', (
      tester,
    ) async {
      await _pump(
        tester,
        ReportSubjectSheet(
          subjectType: ReportSubjectType.post,
          onSubmit: (_) async {},
        ),
      );

      final scrollView = tester
          .widgetList<SingleChildScrollView>(
            find.byKey(const ValueKey('reportSubjectRouteScrollView')),
          )
          .single;
      final padding = scrollView.padding! as EdgeInsets;

      expect(padding.left, 16);
      expect(padding.top, 16);
      expect(padding.right, 16);
      expect(padding.bottom, 32);
    });
  });
}
