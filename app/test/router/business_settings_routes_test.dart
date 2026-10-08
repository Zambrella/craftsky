import 'dart:async';

import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
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
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../fakes/auth_session_fakes.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  for (final route in _ownerRoutes) {
    testWidgets('AT-008 ${route.label} route closes on Business lapse', (
      tester,
    ) async {
      final tier = StateProvider<SubscriptionTier>(
        (ref) => SubscriptionTier.business,
      );
      final harness = await _pumpRouter(
        tester,
        accountType: AccountType.business,
        initialLocation: route.location,
        size: route.size,
        accessTierProvider: tier,
      );
      expect(find.byType(route.pageType), findsOneWidget);

      harness.container.read(tier.notifier).state = SubscriptionTier.free;
      await tester.pumpAndSettle();
      expect(
        harness.router.state.matchedLocation,
        const SettingsRoute().location,
      );
      expect(find.byType(route.pageType), findsNothing);
      expect(find.byType(SettingsPage), findsOneWidget);
    });
  }

  testWidgets(
    'AT-008 cached Business label cannot open owner page without access',
    (tester) async {
      final harness = await _pumpRouter(
        tester,
        accountType: AccountType.business,
        accessTier: SubscriptionTier.plus,
        initialLocation: const BusinessProductsRoute().location,
        size: const Size(500, 800),
      );
      expect(
        harness.router.state.matchedLocation,
        const SettingsRoute().location,
      );
      expect(find.byType(ProductsSettingsPage), findsNothing);
    },
  );
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
      skip: !subscriptionsEnabled,
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
        expect(
          harness.router.state.matchedLocation,
          const ProfileRoute().location,
        );
        expect(find.byType(Tab), findsWidgets);

        await tester.tap(find.widgetWithText(Tab, route.profileTabLabel));
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
  SubscriptionTier? accessTier,
  StateProvider<SubscriptionTier>? accessTierProvider,
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
      secureSessionRegistryStorageProvider.overrideWithValue(
        _BusinessRegistryStorage(
          SessionRegistry.empty().upsertAndActivate(
            token: 'test-token',
            did: 'did:plc:test',
            handle: 'test.bsky.social',
          ),
        ),
      ),
      subscriptionAccessProvider.overrideWith((ref, lease) async {
        final tier =
            (accessTierProvider == null
                ? null
                : ref.watch(accessTierProvider)) ??
            accessTier ??
            (accountType == AccountType.business
                ? SubscriptionTier.business
                : SubscriptionTier.free);
        return SubscriptionAccess(
          did: Did.parse('did:plc:test'),
          effectiveTier: tier,
          givesAccess: tier != SubscriptionTier.free,
          assignedTier: tier == SubscriptionTier.free ? null : tier,
        );
      }),
      sharedPreferencesProvider.overrideWithValue(preferences),
      authSessionProvider.overrideWith(SignedInAuthSession.new),
      onboardingStatusProvider.overrideWith2(
        (_) => CompletedOnboardingStatus(),
      ),
      activeAccountIdentityProvider.overrideWith(
        (_) async => identity,
      ),
      profileRepositoryProvider.overrideWithValue(
        FakeProfileRepository(
          onFetch: (_) async => profile,
          onFetchMe: () async => profile,
        ),
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
      pdsRecordOperationControllerProvider.overrideWithValue(
        PdsRecordOperationController(schedule: (_, _) {}),
      ),
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

final class _BusinessRegistryStorage implements SessionRegistryStorage {
  _BusinessRegistryStorage(this.registry);
  SessionRegistry registry;
  @override
  Future<SessionRegistry> read() async => registry;
  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
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
    required String operationKey,
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
