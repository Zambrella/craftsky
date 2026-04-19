import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('OnboardingPage renders Finish button', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const OnboardingPage(),
        ),
      ),
    );
    expect(find.widgetWithText(ChunkyButton, 'Finish'), findsOneWidget);
  });

  testWidgets('Finish flips onboardingStatusProvider to true', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    // Pin the autoDispose provider. See Task 6 review notes.
    final subscription = container.listen<bool>(
      onboardingStatusProvider,
      (_, _) {},
    );
    addTearDown(subscription.close);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const OnboardingPage(),
        ),
      ),
    );

    await tester.tap(find.widgetWithText(ChunkyButton, 'Finish'));
    await tester.pumpAndSettle();

    expect(container.read(onboardingStatusProvider), isTrue);
  });
}
