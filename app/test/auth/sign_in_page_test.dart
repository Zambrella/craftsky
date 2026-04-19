import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SignInPage renders a handle field and Continue button', (
    tester,
  ) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SignInPage()),
      ),
    );
    expect(find.byType(TextField), findsOneWidget);
    expect(find.widgetWithText(ElevatedButton, 'Continue'), findsOneWidget);
  });

  testWidgets('Continue flips authStatusProvider to true', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    // Pin the autoDispose authStatusProvider so it survives the
    // tap→pumpAndSettle cycle and the post-pump container.read assertion.
    // See Task 6 review notes.
    final subscription = container.listen<bool>(
      authStatusProvider,
      (_, _) {},
    );
    addTearDown(subscription.close);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: SignInPage()),
      ),
    );

    await tester.tap(find.widgetWithText(ElevatedButton, 'Continue'));
    await tester.pumpAndSettle();

    expect(container.read(authStatusProvider), isTrue);
  });
}
