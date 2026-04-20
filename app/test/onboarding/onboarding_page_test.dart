import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
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
}
