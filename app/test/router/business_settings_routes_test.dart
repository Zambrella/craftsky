import 'dart:async';

import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/pages/events_settings_page.dart';
import 'package:craftsky_app/business/pages/products_settings_page.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  test('UT-018 owner routes use canonical settings locations', () {
    expect(
      const BusinessProductsRoute().location,
      '/profile/settings/products',
    );
    expect(
      const BusinessEventsRoute().location,
      '/profile/settings/events',
    );
  });

  for (final route in _ownerRoutes) {
    testWidgets(
      'IT-011 regular account deep link to ${route.label} returns to Settings',
      (tester) async {
        final harness = await _pumpRouter(
          tester,
          accountType: AccountType.regular,
          initialLocation: route.location,
          size: route.size,
        );

        expect(
          harness.router.state.matchedLocation,
          const SettingsRoute().location,
        );
        expect(find.byType(SettingsPage), findsOneWidget);
        expect(find.byType(route.pageType), findsNothing);
      },
    );

    testWidgets('IT-011 business account can reach ${route.label}', (
      tester,
    ) async {
      final harness = await _pumpRouter(
        tester,
        accountType: AccountType.business,
        initialLocation: route.location,
        size: route.size,
      );

      expect(harness.router.state.matchedLocation, route.location);
      expect(find.byType(route.pageType), findsOneWidget);
    });

    testWidgets(
      'IT-011 ${route.layout} Settings row opens ${route.label}',
      (tester) async {
        final harness = await _pumpRouter(
          tester,
          accountType: AccountType.business,
          initialLocation: const SettingsRoute().location,
          size: route.size,
        );

        expect(find.byType(SettingsPage), findsOneWidget);
        await tester.drag(find.byType(ListView), const Offset(0, -700));
        await tester.pumpAndSettle();
        await tester.tap(find.text(route.label));
        await tester.pumpAndSettle();

        expect(harness.router.state.matchedLocation, route.location);
        expect(find.byType(route.pageType), findsOneWidget);
        expect(
          find.byType(NavigationRail),
          route.layout == 'wide' ? findsOneWidget : findsNothing,
        );
      },
    );
  }

  for (final route in _ownerRoutes) {
    testWidgets(
      '${route.acceptanceId} owner empty-state CTA opens ${route.label}',
      (tester) async {
        final harness = await _pumpRouter(
          tester,
          accountType: AccountType.business,
          initialLocation: const ProfileRoute().location,
          size: const Size(800, 900),
        );

        await tester.tap(find.text(route.profileTabLabel));
        await tester.pumpAndSettle();
        await tester.tap(find.text(route.manageLabel));
        await tester.pumpAndSettle();

        expect(harness.router.state.matchedLocation, route.location);
        expect(find.byType(route.pageType), findsOneWidget);
      },
    );
  }

  testWidgets(
    'IT-010 REG-008 system Back leaves immediately persisted Products',
    (tester) async {
      final harness = await _pumpRouter(
        tester,
        accountType: AccountType.business,
        initialLocation: const SettingsRoute().location,
        size: const Size(500, 800),
        products: [_firstProduct, _secondProduct],
      );
      final productsLocation = const BusinessProductsRoute().location;
      unawaited(harness.router.push(productsLocation));
      await tester.pumpAndSettle();

      final controller = harness.container.read(
        productsControllerProvider.notifier,
      );
      expect(
        await controller.move(
          controller.state.requireValue.products.first.id,
          1,
        ),
        isTrue,
      );
      await tester.pumpAndSettle();

      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();

      expect(
        harness.router.state.matchedLocation,
        const SettingsRoute().location,
      );
      expect(find.byType(ProductsSettingsPage), findsNothing);
      expect(
        await harness.container
            .read(unsavedWorkGuardProvider)
            .confirmLeave(harness.identity.lease),
        isTrue,
      );
      expect(find.text('Discard changes?'), findsNothing);
      expect(
        controller.state.requireValue.products.first.title,
        _secondProduct.title,
      );
    },
  );
}

Future<_RouterHarness> _pumpRouter(
  WidgetTester tester, {
  required AccountType accountType,
  required String initialLocation,
  required Size size,
  List<BusinessProductView> products = const [],
}) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.resetDevicePixelRatio);
  addTearDown(tester.view.resetPhysicalSize);

  final business = accountType == AccountType.business
      ? BusinessProfile(cid: 'bafy-business', products: products)
      : null;
  final profile = Profile(
    did: 'did:plc:test',
    handle: 'test.bsky.social',
    crafts: const [],
    accountType: accountType,
    business: business,
  );
  final identity = ActiveAccountIdentity(
    lease: AccountSessionLease(
      account: AccountKey('did:plc:test'),
      sessionGeneration: 1,
    ),
    profile: profile,
  );
  SharedPreferences.setMockInitialValues({});
  final preferences = await SharedPreferences.getInstance();
  final container = ProviderContainer.test(
    overrides: [
      sharedPreferencesProvider.overrideWithValue(preferences),
      authSessionProvider.overrideWith(SignedInAuthSession.new),
      onboardingStatusProvider.overrideWith2(
        (_) => CompletedOnboardingStatus(),
      ),
      activeAccountIdentityProvider.overrideWith(
        (_) async => identity,
      ),
      profileRepositoryProvider.overrideWithValue(
        FakeProfileRepository(onFetch: (_) async => profile),
      ),
      postRepositoryProvider.overrideWithValue(
        FakePostRepository(
          onListByAuthor: (_, {cursor, limit}) async =>
              const PostPage(items: []),
          onListCommentsByAuthor: (_, {cursor, limit}) async =>
              const PostPage(items: []),
        ),
      ),
      businessRepositoryProvider.overrideWithValue(_BusinessRepository()),
    ],
    retry: (_, _) => null,
  );
  addTearDown(container.dispose);
  final subscription = container.listen(
    goRouterProvider,
    (_, _) {},
    fireImmediately: true,
  );
  addTearDown(subscription.close);
  final router = subscription.read()..go(initialLocation);

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
  return _RouterHarness(
    router: router,
    container: container,
    identity: identity,
  );
}

final class _RouterHarness {
  const _RouterHarness({
    required this.router,
    required this.container,
    required this.identity,
  });

  final GoRouter router;
  final ProviderContainer container;
  final ActiveAccountIdentity identity;
}

final _firstProduct = BusinessProductView(
  title: 'One',
  uri: 'https://shop.example/one',
  image: BusinessImageView(
    cid: 'bafy-one',
    mime: 'image/jpeg',
    size: 10,
    alt: 'One',
    thumb: 'https://cdn.example/one/thumb',
    fullsize: 'https://cdn.example/one/full',
  ),
);

final _secondProduct = BusinessProductView(
  title: 'Two',
  uri: 'https://shop.example/two',
  image: BusinessImageView(
    cid: 'bafy-two',
    mime: 'image/jpeg',
    size: 10,
    alt: 'Two',
    thumb: 'https://cdn.example/two/thumb',
    fullsize: 'https://cdn.example/two/full',
  ),
);

final class _BusinessRepository extends Fake implements BusinessRepository {
  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) async => RecordMutationResult(cid: 'bafy-products-accepted');

  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) async => const BusinessEventPage(items: []);

  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) async => const BusinessEventPage(items: []);
}

final _ownerRoutes = <_OwnerRoute>[
  _OwnerRoute(
    label: 'Products',
    location: const BusinessProductsRoute().location,
    pageType: ProductsSettingsPage,
    layout: 'compact',
    size: const Size(500, 800),
    acceptanceId: 'AT-003',
    profileTabLabel: 'Products',
    manageLabel: 'Manage products',
  ),
  _OwnerRoute(
    label: 'Events',
    location: const BusinessEventsRoute().location,
    pageType: EventsSettingsPage,
    layout: 'wide',
    size: const Size(1200, 800),
    acceptanceId: 'AT-009',
    profileTabLabel: 'Upcoming Events',
    manageLabel: 'Manage events',
  ),
];

final class _OwnerRoute {
  const _OwnerRoute({
    required this.label,
    required this.location,
    required this.pageType,
    required this.layout,
    required this.size,
    required this.acceptanceId,
    required this.profileTabLabel,
    required this.manageLabel,
  });

  final String label;
  final String location;
  final Type pageType;
  final String layout;
  final Size size;
  final String acceptanceId;
  final String profileTabLabel;
  final String manageLabel;
}
