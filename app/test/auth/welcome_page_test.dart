import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('WelcomePage renders title and dev auth toggle', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          home: const WelcomePage(),
        ),
      ),
    );
    expect(find.text('Welcome'), findsWidgets);
    expect(
      find.widgetWithText(OutlinedButton, 'Dev: toggle auth'),
      findsOneWidget,
    );
  });
}
