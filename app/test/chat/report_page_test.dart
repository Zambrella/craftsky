import 'package:craftsky_app/chat/pages/report_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'IT-004 uncertain report keeps explanation and checks status '
    'without replay',
    (tester) async {
      var submissions = 0;
      var checks = 0;
      await tester.pumpWidget(
        MaterialApp(
          builder: (context, child) =>
              MessengerScope(messenger: RecordingMessenger(), child: child!),
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ReportPage(
            onSubmit: (type, reason) async {
              submissions++;
              return ReportSubmissionState.uncertain;
            },
            onCheckStatus: () async {
              checks++;
              return ReportSubmissionState.confirmed;
            },
          ),
        ),
      );
      await _selectReason(tester, 'Other');
      await tester.enterText(
        find.byKey(const Key('chat-report-explanation')),
        'private-explanation',
      );
      await tester.ensureVisible(find.text('Submit'));
      await tester.tap(find.text('Submit'));
      await tester.pumpAndSettle();
      expect(find.text('Report sent to Bluesky.'), findsNothing);
      expect(
        tester
            .widget<TextField>(find.byKey(const Key('chat-report-explanation')))
            .controller!
            .text,
        'private-explanation',
      );
      await tester.ensureVisible(find.text('Check status'));
      await tester.tap(find.text('Check status'));
      await tester.pumpAndSettle();
      expect(submissions, 1);
      expect(checks, 1);
      expect(find.text('Report sent to Bluesky.'), findsOneWidget);
    },
  );
  testWidgets(
    'UT-011 / AT-010 names Bluesky before confirmation without extra paragraph',
    (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        MaterialApp(
          builder: (context, child) =>
              MessengerScope(messenger: RecordingMessenger(), child: child!),
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ReportPage(
            onSubmit: (type, reason) async {
              calls++;
              expect(type, 'com.atproto.moderation.defs#reasonOther');
              expect(reason, 'My explanation');
              return ReportSubmissionState.confirmed;
            },
          ),
        ),
      );
      expect(calls, 0);
      final submit = find.widgetWithText(TextButton, 'Submit');
      expect(tester.widget<TextButton>(submit).onPressed, isNull);
      expect(find.text('Other'), findsNothing);
      expect(
        tester
            .widget<TextField>(
              find.byKey(const Key('chat-report-reason-search-input')),
            )
            .decoration
            ?.hintText,
        'Select Reason',
      );
      expect(
        find.descendant(of: find.byType(AppBar), matching: find.text('Submit')),
        findsOneWidget,
      );
      expect(find.textContaining('Bluesky only'), findsOneWidget);
      expect(find.textContaining('reference to this message'), findsNothing);
      expect(find.textContaining('copied transcript'), findsNothing);
      expect(find.text('Report sent to Bluesky.'), findsNothing);
      await tester.enterText(
        find.byKey(const Key('chat-report-explanation')),
        'My explanation',
      );
      expect(tester.widget<TextButton>(submit).onPressed, isNull);
      await _selectReason(tester, 'Other');
      expect(calls, 0);
      expect(tester.widget<TextButton>(submit).onPressed, isNotNull);
      await tester.tap(submit);
      await tester.pumpAndSettle();
      expect(calls, 1);
      expect(find.text('Report sent to Bluesky.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}

Future<void> _selectReason(WidgetTester tester, String label) async {
  final search = find.byKey(const Key('chat-report-reason-search-input'));
  await tester.tap(search);
  await tester.enterText(search, label);
  await tester.pumpAndSettle();
  await tester.tap(
    find.descendant(
      of: find.byKey(const Key('chat-report-reason-options-panel')),
      matching: find.text(label),
    ),
  );
  await tester.pumpAndSettle();
}
