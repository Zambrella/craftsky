import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SettingsPage renders title and sign-out button', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SettingsPage()),
      ),
    );
    expect(find.text('Settings'), findsWidgets);
    expect(
      find.widgetWithText(OutlinedButton, 'Sign out (dev)'),
      findsOneWidget,
    );
  });
}
