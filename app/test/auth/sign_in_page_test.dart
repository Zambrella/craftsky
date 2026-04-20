import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SignInPage renders a handle field and Continue button', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const SignInPage(),
        ),
      ),
    );
    expect(find.byType(BrandTextField), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Continue'), findsOneWidget);
  });
}
