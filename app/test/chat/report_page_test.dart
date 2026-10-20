import 'package:craftsky_app/chat/pages/report_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'IT-004 uncertain report keeps explanation and checks status '
    'without replay',
    (tester) async {
      var submissions = 0;
      var checks = 0;
      await tester.pumpWidget(
        MaterialApp(
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
      await tester.enterText(
        find.byKey(const Key('chat-report-explanation')),
        'private-explanation',
      );
      await tester.ensureVisible(find.text('Report to Bluesky'));
      await tester.tap(find.text('Report to Bluesky'));
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
    'UT-011 / AT-010 names Bluesky and evidence before confirmation',
    (tester) async {
      var calls = 0;
      await tester.pumpWidget(
        MaterialApp(
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
      expect(find.textContaining('Bluesky only'), findsOneWidget);
      expect(find.textContaining('reference to this message'), findsOneWidget);
      expect(find.textContaining('copied transcript'), findsOneWidget);
      expect(find.text('Report sent to Bluesky.'), findsNothing);
      await tester.enterText(
        find.byKey(const Key('chat-report-explanation')),
        'My explanation',
      );
      await tester.ensureVisible(find.text('Report to Bluesky'));
      await tester.tap(find.text('Report to Bluesky'));
      await tester.pumpAndSettle();
      expect(calls, 1);
      expect(find.text('Report sent to Bluesky.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
