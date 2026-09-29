import 'dart:async';

import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_feature_lock.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  late SharedPreferences preferences;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    preferences = await SharedPreferences.getInstance();
  });

  testWidgets(
    'AT-008 Business Settings entries hide during access refresh',
    (
      tester,
    ) async {
      await tester.binding.setSurfaceSize(const Size(800, 1600));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      var refreshing = false;
      final pending = Completer<SubscriptionAccess>();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            sharedPreferencesProvider.overrideWithValue(preferences),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _SettingsRegistryStorage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'test-token',
                  did: 'did:plc:test',
                  handle: 'test.bsky.social',
                ),
              ),
            ),
            subscriptionAccessProvider.overrideWith((ref, lease) {
              if (refreshing) return pending.future;
              return Future.value(
                SubscriptionAccess(
                  did: lease.account.did,
                  effectiveTier: SubscriptionTier.business,
                  givesAccess: true,
                  assignedTier: SubscriptionTier.business,
                ),
              );
            }),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SettingsPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(SettingsPage)),
      );
      expect(find.widgetWithText(SettingsRowTile, 'Products'), findsOneWidget);
      final lease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!
          .session;
      refreshing = true;
      container.invalidate(subscriptionAccessProvider(lease));
      await tester.pump();
      final access = container.read(subscriptionAccessProvider(lease));
      expect(access.isLoading, isTrue);
      expect(access, isA<AsyncData<SubscriptionAccess>>());
      expect(access.value?.allowsBusiness, isTrue);
      await tester.pump();
      expect(find.widgetWithText(SettingsRowTile, 'Events'), findsNothing);
      expect(find.widgetWithText(SettingsRowTile, 'Products'), findsNothing);
      expect(find.text('Explore subscriptions'), findsOneWidget);
      pending.complete(
        SubscriptionAccess(
          did: lease.account.did,
          effectiveTier: SubscriptionTier.business,
          givesAccess: true,
          assignedTier: SubscriptionTier.business,
        ),
      );
      await tester.pumpAndSettle();
      expect(find.widgetWithText(SettingsRowTile, 'Products'), findsOneWidget);
      expect(find.text('View subscription'), findsOneWidget);
    },
    skip: !subscriptionsEnabled,
  );

  testWidgets('AT-007 Free customisation row shows a labelled Plus lock', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sharedPreferencesProvider.overrideWithValue(preferences),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _SettingsRegistryStorage(
              SessionRegistry.empty().upsertAndActivate(
                token: 'test-token',
                did: 'did:plc:test',
                handle: 'test.bsky.social',
              ),
            ),
          ),
          subscriptionAccessProvider.overrideWith(
            (ref, lease) async => SubscriptionAccess(
              did: Did.parse('did:plc:test'),
              effectiveTier: SubscriptionTier.free,
              givesAccess: false,
            ),
          ),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const SettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      find.byIcon(CraftskyIcons.plusTier),
      subscriptionsEnabled ? findsWidgets : findsNothing,
    );
    await tester.tap(
      find.ancestor(
        of: find.text('Customisation'),
        matching: find.byType(PlusFeatureLock),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      find.text(
        subscriptionsEnabled
            ? 'Customisation requires Plus'
            : 'Customisation is coming soon',
      ),
      findsOneWidget,
    );
  });

  for (final accountType in AccountType.values) {
    testWidgets(
      'REG-003 Settings rows preserve order for authoritative '
      '${accountType.name} type',
      (tester) async {
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = const Size(800, 1600);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              sharedPreferencesProvider.overrideWithValue(preferences),
              activeAccountIdentityProvider.overrideWith(
                (_) async => _identity(accountType),
              ),
            ],
            child: const MaterialApp(
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: SettingsPage(),
            ),
          ),
        );
        await tester.pumpAndSettle();

        final rowIds = tester
            .widgetList<SettingsRowTile>(find.byType(SettingsRowTile))
            .map((tile) => tile.descriptor.id)
            .toList();
        expect(
          rowIds,
          _regularSettingsRows,
          reason:
              'REG-003 must preserve existing rows and insert Business only',
        );

        await tester.drag(find.byType(ListView), const Offset(0, -600));
        await tester.pumpAndSettle();

        expect(find.text('Business'), findsNothing);
        expect(find.text('Events'), findsNothing);
        expect(find.text('Products'), findsNothing);
      },
    );
  }

  testWidgets(
    'SettingsPage renders the expanded hierarchy and SignOutTile',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            sharedPreferencesProvider.overrideWithValue(preferences),
          ],
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: SettingsPage(),
          ),
        ),
      );
      expect(find.text('Settings'), findsWidgets);
      expect(
        find.widgetWithText(SettingsRowTile, 'Subscriptions'),
        findsNothing,
      );
      expect(find.text('Languages'), findsOneWidget);
      expect(find.text('Customisation'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('Growth'),
        180,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('Growth'), findsOneWidget);
      expect(find.text('Followers'), findsOneWidget);
      expect(find.text('Following'), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text('Find people from Instagram'),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.text('Find people from Instagram'), findsOneWidget);
      expect(find.text('Saved posts'), findsNothing);
      expect(find.text('Scheduled posts'), findsNothing);
      expect(find.text('Drafts'), findsNothing);
      expect(find.textContaining(RegExp(r'\d+ followers')), findsNothing);
      expect(find.textContaining(RegExp(r'\d+ following')), findsNothing);
      expect(find.text('Clear image cache'), findsNothing);
      await tester.scrollUntilVisible(
        find.byType(SignOutTile),
        300,
        scrollable: find.byType(Scrollable).first,
      );
      expect(find.byType(SignOutTile), findsOneWidget);
    },
  );

  testWidgets(
    'subscription callout sits below switch account and opens tiers',
    (
      tester,
    ) async {
      final router = GoRouter(
        initialLocation: '/settings',
        routes: [
          GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
          GoRoute(
            path: '/profile/settings/subscriptions',
            builder: (_, _) => const Scaffold(body: Text('Subscription route')),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [sharedPreferencesProvider.overrideWithValue(preferences)],
          child: MaterialApp.router(
            routerConfig: router,
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();

      final callout = find.byKey(const Key('settings-subscription-callout'));
      expect(callout, findsOneWidget);
      final card = tester.widget<Container>(callout);
      expect(
        (card.decoration! as BoxDecoration).boxShadow,
        Theme.of(tester.element(callout)).extension<BrandShadowTheme>()!.dropSm,
      );
      expect(
        tester.getTopLeft(callout).dy,
        greaterThan(tester.getBottomLeft(find.text('Switch account')).dy),
      );
      expect(
        tester.getBottomLeft(callout).dy,
        lessThan(tester.getTopLeft(find.text('Preferences')).dy),
      );
      expect(
        find.widgetWithText(SettingsRowTile, 'Subscriptions'),
        findsNothing,
      );
      expect(find.text('Explore subscriptions'), findsOneWidget);

      await tester.tap(find.text('Make more with CraftSky'));
      await tester.pumpAndSettle();
      expect(router.state.uri.path, '/profile/settings/subscriptions');
      expect(find.text('Subscription route'), findsOneWidget);
    },
    skip: !subscriptionsEnabled,
  );

  testWidgets(
    'subscription callout fits a narrow screen with large text',
    (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 568);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [sharedPreferencesProvider.overrideWithValue(preferences)],
          child: MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context).copyWith(
                textScaler: const TextScaler.linear(2),
              ),
              child: child!,
            ),
            home: const SettingsPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      expect(find.text('Explore subscriptions'), findsOneWidget);
    },
    skip: !subscriptionsEnabled,
  );

  testWidgets('Instagram settings entry opens the typed migration location', (
    tester,
  ) async {
    final router = GoRouter(
      initialLocation: '/settings',
      routes: [
        GoRoute(
          path: '/settings',
          builder: (_, _) => const SettingsPage(),
        ),
        GoRoute(
          path: '/profile/settings/instagram',
          builder: (_, _) => const Scaffold(body: Text('Instagram route')),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sharedPreferencesProvider.overrideWithValue(preferences),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('Find people from Instagram'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Find people from Instagram'));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/profile/settings/instagram');
    expect(find.text('Instagram route'), findsOneWidget);
  });

  testWidgets('Account standing row opens the typed moderation location', (
    tester,
  ) async {
    final router = GoRouter(
      initialLocation: '/settings',
      routes: [
        GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
        GoRoute(
          path: '/profile/settings/moderation',
          builder: (_, _) => const Scaffold(body: Text('Standing route')),
        ),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [sharedPreferencesProvider.overrideWithValue(preferences)],
        child: MaterialApp.router(
          routerConfig: router,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text('Account standing'),
      300,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.text('Account standing'));
    await tester.pumpAndSettle();

    expect(router.state.uri.path, '/profile/settings/moderation');
    expect(find.text('Standing route'), findsOneWidget);
  });

  testWidgets(
    'AT-005 business Products row opens product management',
    (
      tester,
    ) async {
      final router = GoRouter(
        initialLocation: '/settings',
        routes: [
          GoRoute(path: '/settings', builder: (_, _) => const SettingsPage()),
          GoRoute(
            path: '/profile/settings/products',
            builder: (_, _) => const Scaffold(body: Text('Products route')),
          ),
        ],
      );
      addTearDown(router.dispose);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            sharedPreferencesProvider.overrideWithValue(preferences),
            activeAccountIdentityProvider.overrideWith(
              (_) async => _identity(AccountType.business),
            ),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _SettingsRegistryStorage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'test-token',
                  did: 'did:plc:test',
                  handle: 'test.bsky.social',
                ),
              ),
            ),
            subscriptionAccessProvider.overrideWith(
              (ref, lease) async => SubscriptionAccess(
                did: Did.parse('did:plc:test'),
                effectiveTier: SubscriptionTier.business,
                givesAccess: true,
                assignedTier: SubscriptionTier.business,
              ),
            ),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text('Products'),
        300,
        scrollable: find.byType(Scrollable).first,
      );

      await tester.ensureVisible(find.text('Products'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Products'));
      await tester.pumpAndSettle();

      expect(router.state.uri.path, '/profile/settings/products');
      expect(find.text('Products route'), findsOneWidget);
    },
    skip: !subscriptionsEnabled,
  );

  testWidgets(
    'beta hides subscription callout and Business settings',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [sharedPreferencesProvider.overrideWithValue(preferences)],
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: SettingsPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(const Key('settings-subscription-callout')),
        findsNothing,
      );
      expect(find.text('Explore subscriptions'), findsNothing);
      expect(find.text('Products'), findsNothing);
    },
    skip: subscriptionsEnabled,
  );
}

const _regularSettingsRows = <SettingsRowId>[
  SettingsRowId.switchAccount,
  SettingsRowId.appearance,
  SettingsRowId.customisation,
  SettingsRowId.languages,
  SettingsRowId.notifications,
  SettingsRowId.growth,
  SettingsRowId.followers,
  SettingsRowId.following,
  SettingsRowId.mutedAccounts,
  SettingsRowId.blockedAccounts,
  SettingsRowId.findPeopleFromInstagram,
  SettingsRowId.accountStanding,
  SettingsRowId.account,
  SettingsRowId.about,
  SettingsRowId.signOut,
];

ActiveAccountIdentity _identity(AccountType type) => ActiveAccountIdentity(
  lease: AccountSessionLease(
    account: AccountKey('did:plc:test'),
    sessionGeneration: 1,
  ),
  profile: Profile(
    did: 'did:plc:test',
    handle: 'test.bsky.social',
    crafts: const [],
    accountType: type,
  ),
);

final class _SettingsRegistryStorage implements SessionRegistryStorage {
  _SettingsRegistryStorage(this.registry);
  SessionRegistry registry;

  @override
  Future<SessionRegistry> read() async => registry;

  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}
