import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('WelcomePage renders Welcome + Sign in + Create account',
      (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const WelcomePage(),
        ),
      ),
    );
    expect(find.text('Welcome'), findsWidgets);
    expect(find.widgetWithText(ChunkyButton, 'Sign in'), findsOneWidget);
    expect(find.text('Create account on a PDS'), findsOneWidget);
  });
}
