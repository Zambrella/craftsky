import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('WelcomePage renders title and dev auth toggle', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: WelcomePage()),
      ),
    );
    expect(find.text('Welcome'), findsWidgets);
    expect(find.widgetWithText(OutlinedButton, 'Dev: toggle auth'), findsOneWidget);
  });

  testWidgets('tapping the dev toggle flips authStatusProvider to true', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    // Keep a live listener so the autoDispose provider isn't torn down
    // between the tap and the assertion.
    final subscription = container.listen<bool>(
      authStatusProvider,
      (_, __) {},
    );
    addTearDown(subscription.close);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: WelcomePage()),
      ),
    );

    await tester.tap(find.widgetWithText(OutlinedButton, 'Dev: toggle auth'));
    await tester.pumpAndSettle(); // NOTE: pumpAndSettle (not pump) — avoids timersPending assertion with autoDispose providers

    expect(container.read(authStatusProvider), isTrue);
  });
}
