import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
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

  testWidgets('tapping sign-out button flips authStatusProvider to false', (
    tester,
  ) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    // Pin the autoDispose provider so it survives the tap→pumpAndSettle
    // cycle and the post-pump container.read assertion.
    final subscription = container.listen<bool>(
      authStatusProvider,
      (_, _) {},
    );
    addTearDown(subscription.close);

    container.read(authStatusProvider.notifier).signIn();

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: SettingsPage()),
      ),
    );

    await tester.tap(find.widgetWithText(OutlinedButton, 'Sign out (dev)'));
    await tester.pumpAndSettle();

    expect(container.read(authStatusProvider), isFalse);
  });
}
