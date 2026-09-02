import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
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
          accountType == AccountType.business
              ? _businessSettingsRows
              : _regularSettingsRows,
          reason:
              'REG-003 must preserve existing rows and insert Business only',
        );

        await tester.drag(find.byType(ListView), const Offset(0, -600));
        await tester.pumpAndSettle();

        if (accountType == AccountType.business) {
          expect(find.text('Business'), findsOneWidget);
          expect(find.text('Events'), findsOneWidget);
          expect(find.text('Products'), findsOneWidget);
        } else {
          expect(find.text('Business'), findsNothing);
          expect(find.text('Events'), findsNothing);
          expect(find.text('Products'), findsNothing);
        }
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
      expect(find.text('Languages'), findsOneWidget);
      expect(find.text('Customisation'), findsOneWidget);
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

  testWidgets('AT-005 business Products row opens product management', (
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
  });
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
  SettingsRowId.account,
  SettingsRowId.about,
  SettingsRowId.signOut,
];

const _businessSettingsRows = <SettingsRowId>[
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
  SettingsRowId.businessEvents,
  SettingsRowId.businessProducts,
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
