import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('ProfilePage renders nav buttons', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: ProfilePage()),
      ),
    );
    expect(find.widgetWithText(OutlinedButton, 'Settings'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Saved'), findsOneWidget);
    expect(
      find.widgetWithText(OutlinedButton, 'Open a user profile'),
      findsOneWidget,
    );
    expect(
      find.widgetWithText(OutlinedButton, 'Design playground'),
      findsOneWidget,
    );
  });
}
