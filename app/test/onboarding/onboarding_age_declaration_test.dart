import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_guidelines_step.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-010 collects only an explicit 16+ declaration', (
    tester,
  ) async {
    var accepted = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: StatefulBuilder(
          builder: (context, setState) => Scaffold(
            body: SingleChildScrollView(
              child: OnboardingGuidelinesStep(
                onViewFullGuidelines: () {},
                meetsMinimumAge: accepted,
                onMeetsMinimumAgeChanged: (value) =>
                    setState(() => accepted = value),
              ),
            ),
          ),
        ),
      ),
    );

    expect(find.textContaining('at least 16 years old'), findsOneWidget);
    expect(find.textContaining('date of birth'), findsNothing);
    expect(find.textContaining('age band'), findsNothing);
    final declaration = find.byKey(
      const Key('onboarding-minimum-age-declaration'),
    );
    await tester.ensureVisible(declaration);
    await tester.tap(declaration);
    await tester.pump();
    expect(accepted, isTrue);
  });
}
