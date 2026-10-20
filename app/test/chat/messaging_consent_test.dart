import 'package:craftsky_app/chat/pages/messaging_consent_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('UT-011 reconnect preserves the account push-off choice', (
    tester,
  ) async {
    bool? submitted;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: MessagingConsentPage(
          initialPushEnabled: false,
          onAuthorize: (push) => submitted = push,
        ),
      ),
    );
    expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
    await tester.ensureVisible(find.text('Continue with Bluesky'));
    await tester.tap(find.text('Continue with Bluesky'));
    await tester.pumpAndSettle();
    expect(submitted, isFalse);
  });
  testWidgets(
    'UT-011 / AT-002 discloses shared Bluesky meaning before consent',
    (tester) async {
      bool? selectedPush;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MessagingConsentPage(
            onAuthorize: (push) async {
              selectedPush = push;
            },
          ),
        ),
      );
      expect(selectedPush, isNull);
      expect(find.textContaining('shared Bluesky inbox'), findsOneWidget);
      expect(find.textContaining('other apps'), findsOneWidget);
      expect(find.textContaining('stored by Bluesky'), findsOneWidget);
      expect(find.textContaining('public PDS records'), findsOneWidget);
      expect(find.textContaining('not end-to-end encrypted'), findsOneWidget);
      expect(find.textContaining('Bluesky only'), findsOneWidget);
      expect(find.byType(Switch), findsOneWidget);
      expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
      await tester.ensureVisible(find.text('Continue with Bluesky'));
      await tester.tap(find.text('Continue with Bluesky'));
      await tester.pumpAndSettle();
      expect(selectedPush, isTrue);
      expect(tester.takeException(), isNull);
    },
  );
}
