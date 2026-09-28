import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_guidelines_step.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'shows the community guidelines summary and full document action',
    (tester) async {
      var viewFullCalls = 0;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: OnboardingGuidelinesStep(
                onViewFullGuidelines: () => viewFullCalls++,
              ),
            ),
          ),
        ),
      );

      expect(find.text('Our community guidelines'), findsOneWidget);
      expect(find.text('Be kind and keep CraftSky safe'), findsOneWidget);
      expect(
        find.textContaining('materially created by generative AI'),
        findsOneWidget,
      );
      expect(find.text('Keep it craft-focused'), findsOneWidget);
      expect(
        find.textContaining('Does this make the community a better place?'),
        findsOneWidget,
      );

      final action = find.text('View full guidelines');
      await tester.ensureVisible(action);
      await tester.tap(action);
      expect(viewFullCalls, 1);
    },
  );
}
