import 'package:craftsky_app/chat/widgets/messaging_explanation.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets(
    'UT-011 / AT-002 setup explains shared Bluesky privacy and reporting',
    (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(body: MessagingExplanation()),
        ),
      );
      expect(find.textContaining('shared Bluesky inbox'), findsOneWidget);
      expect(find.textContaining('other apps'), findsOneWidget);
      expect(find.textContaining('stored by Bluesky'), findsOneWidget);
      expect(find.textContaining('public PDS records'), findsOneWidget);
      expect(find.textContaining('not end-to-end encrypted'), findsOneWidget);
      expect(find.textContaining('Bluesky only'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
