import 'dart:async';

import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _RecordingAuthController extends AuthController {
  final List<String> signInCalls = [];

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> signIn({required String handle}) async {
    signInCalls.add(handle);
  }
}

void main() {
  testWidgets('renders a handle field and a Continue button', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const SignInPage(),
        ),
      ),
    );
    expect(find.byType(BrandTextField), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Continue'), findsOneWidget);
  });

  testWidgets(
    'tapping Continue dispatches AuthController.signIn with text',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(_RecordingAuthController.new),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            home: const SignInPage(),
          ),
        ),
      );
      await tester.enterText(
        find.byType(BrandTextField),
        '  @alice.bsky.social ',
      );
      await tester.tap(find.widgetWithText(ChunkyButton, 'Continue'));
      await tester.pump();

      final fake = tester.container().read(authControllerProvider.notifier)
          as _RecordingAuthController;
      expect(fake.signInCalls, ['  @alice.bsky.social ']);
      // (Controller trims — that's unit-tested in auth_controller_test.dart.)
    },
  );
}
