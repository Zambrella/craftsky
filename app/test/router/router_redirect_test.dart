import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pumpApp(WidgetTester tester, ProviderContainer container) async {
  final router = container.read(goRouterProvider);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        routerConfig: router,
        builder: (context, child) =>
            FormFactorWidget(child: child ?? const SizedBox.shrink()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  group('router redirect', () {
    testWidgets('unauthenticated user lands on WelcomePage', (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      await _pumpApp(tester, container);

      expect(find.byType(WelcomePage), findsOneWidget);
    });

    testWidgets('authed but not onboarded → OnboardingPage', (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(authStatusProvider.notifier).signIn();

      await _pumpApp(tester, container);

      expect(find.byType(OnboardingPage), findsOneWidget);
    });

    testWidgets('authed and onboarded → FeedPage', (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(authStatusProvider.notifier).signIn();
      container.read(onboardingStatusProvider.notifier).finish();

      await _pumpApp(tester, container);

      expect(find.byType(FeedPage), findsOneWidget);
    });
  });
}
