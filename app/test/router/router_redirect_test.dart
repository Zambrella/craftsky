import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';

Future<void> _pumpRouter(
  WidgetTester tester,
  ProviderContainer container, {
  String initialLocation = RouteLocations.welcome,
}) async {
  // Drive the router to a specific initial location before pumping
  // the app, so deep-link-style tests can start on /auth/complete.
  final router = container.read(goRouterProvider)..go(initialLocation);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
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
    testWidgets('SignedOut + /feed → WelcomePage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedOutAuthSession.new),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.feed,
      );
      expect(find.byType(WelcomePage), findsOneWidget);
    });

    testWidgets(
      'SignedOut + /auth/complete stays on AuthCompletePage',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedOutAuthSession.new),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=t',
        );
        expect(find.byType(AuthCompletePage), findsOneWidget);
      },
    );

    testWidgets('SignedIn + not onboarded → OnboardingPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith(PendingOnboardingStatus.new),
        ],
      );
      await _pumpRouter(
        tester,
        container,
        initialLocation: RouteLocations.feed,
      );
      expect(find.byType(OnboardingPage), findsOneWidget);
    });

    testWidgets('SignedIn + onboarded + /welcome → FeedPage', (tester) async {
      final container = ProviderContainer.test(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          onboardingStatusProvider.overrideWith(CompletedOnboardingStatus.new),
        ],
      );
      await _pumpRouter(tester, container);
      expect(find.byType(FeedPage), findsOneWidget);
    });

    testWidgets(
      'SignedIn + onboarded + /auth/complete → FeedPage',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            onboardingStatusProvider.overrideWith(
              CompletedOnboardingStatus.new,
            ),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=t',
        );
        expect(find.byType(FeedPage), findsOneWidget);
      },
    );

    testWidgets(
      'SignedIn + !onboarded + /auth/complete → OnboardingPage',
      (tester) async {
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            onboardingStatusProvider.overrideWith(PendingOnboardingStatus.new),
          ],
        );
        await _pumpRouter(
          tester,
          container,
          initialLocation: '${RouteLocations.authComplete}?token=t',
        );
        expect(find.byType(OnboardingPage), findsOneWidget);
      },
    );
  });
}
