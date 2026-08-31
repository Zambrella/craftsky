import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/event_detail_page.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';

void main() {
  test('AT-006 owner Events route uses the canonical settings location', () {
    expect(
      const BusinessEventsRoute().location,
      '/profile/settings/events',
    );
  });

  test('AT-009 event route has exact typed location and validation', () {
    expect(
      BusinessEventRoute(
        did: 'did:plc:business',
        rkey: '3m4event',
      ).location,
      '/events/did%3Aplc%3Abusiness/3m4event',
    );
    expect(
      () => BusinessEventRoute(did: 'business.example', rkey: '3m4event'),
      throwsA(anything),
    );
    for (final invalid in [
      '',
      '.',
      '..',
      'has/slash',
      'has space',
      'has!bang',
    ]) {
      expect(
        () => BusinessEventRoute(did: 'did:plc:business', rkey: invalid),
        throwsFormatException,
        reason: 'rkey $invalid must be rejected',
      );
    }
  });

  for (final testCase in <({String label, Size size, bool hasRail})>[
    (label: 'compact', size: const Size(500, 800), hasRail: false),
    (label: 'wide', size: const Size(1200, 800), hasRail: true),
  ]) {
    testWidgets(
      'AT-009 ${testCase.label} detail preserves authenticated back stack',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = testCase.size;
        addTearDown(tester.view.resetDevicePixelRatio);
        addTearDown(tester.view.resetPhysicalSize);
        final container = ProviderContainer.test(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            onboardingStatusProvider.overrideWith2(
              (_) => CompletedOnboardingStatus(),
            ),
            businessRepositoryProvider.overrideWithValue(_Repository()),
            postRepositoryProvider.overrideWithValue(
              FakePostRepository(
                onListTimeline: ({cursor, limit}) async =>
                    const TimelinePage(items: []),
              ),
            ),
          ],
          retry: (_, _) => null,
        );
        addTearDown(container.dispose);
        final routerSubscription = container.listen(
          goRouterProvider,
          (_, _) {},
          fireImmediately: true,
        );
        addTearDown(routerSubscription.close);
        final router = routerSubscription.read()..go(RouteLocations.feed);

        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp.router(
              routerConfig: router,
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) =>
                  FormFactorWidget(child: child ?? const SizedBox.shrink()),
            ),
          ),
        );
        await tester.pumpAndSettle();

        router
            .push<void>(
              BusinessEventRoute(
                did: 'did:plc:business',
                rkey: '3m4event',
              ).location,
            )
            .ignore();
        await tester.pumpAndSettle();

        expect(find.byType(EventDetailPage), findsOneWidget);
        expect(find.text('Route test event'), findsOneWidget);
        expect(
          find.byType(NavigationRail),
          testCase.hasRail ? findsOneWidget : findsNothing,
        );

        await tester.pageBack();
        await tester.pumpAndSettle();
        expect(router.state.matchedLocation, RouteLocations.feed);
        expect(find.byType(EventDetailPage), findsNothing);
      },
    );
  }
}

final class _Repository extends Fake implements BusinessRepository {
  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) async =>
      BusinessEvent(
        did: owner.toString(),
        rkey: rkey.toString(),
        uri: 'at://$owner/social.craftsky.business.event/$rkey',
        cid: 'bafy-event',
        name: 'Route test event',
        startsAt: DateTime.utc(2026, 9, 5, 9),
        endsAt: DateTime.utc(2026, 9, 5, 17),
        roles: const [BusinessOpenValue(value: 'vendor', known: true)],
        mode: const BusinessOpenValue(value: 'in-person', known: true),
        status: const BusinessOpenValue(value: 'scheduled', known: true),
        isAllDay: false,
        createdAt: DateTime.utc(2026, 8, 30),
        past: false,
        publicSuppressionReasons: const [],
        upcomingExclusionReasons: const [],
      );
}
