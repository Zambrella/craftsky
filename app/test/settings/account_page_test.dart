import 'dart:ui' show SemanticsAction;

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/settings/pages/account_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show Override;
import 'package:flutter_test/flutter_test.dart';

import '../business/accessibility_test_helpers.dart';
import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';

void main() {
  testWidgets(
    'AT-003 lapsed Business account type is regular despite cached identity',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            activeAccountIdentityProvider.overrideWith(
              (_) async => _identity(AccountType.business),
            ),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _AccountPageRegistryStorage(
                SessionRegistry.empty().upsertAndActivate(
                  token: 'test-token',
                  did: 'did:plc:test',
                  handle: 'test.bsky.social',
                ),
              ),
            ),
            subscriptionAccessProvider.overrideWith(
              (ref, lease) async => SubscriptionAccess(
                did: lease.account.did,
                effectiveTier: SubscriptionTier.free,
                givesAccess: false,
              ),
            ),
          ],
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: AccountPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Regular'), findsOneWidget);
      expect(find.text('Business'), findsNothing);
    },
  );
  for (final constraint in businessAccessibilityMatrix) {
    testWidgets(
      'AT-012 REG-010 account type and delete confirmation fit '
      '${businessConstraintLabel(constraint)}',
      (tester) async {
        await setBusinessAccessibilityConstraint(tester, constraint);
        final semantics = tester.ensureSemantics();
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              ..._businessAccessOverrides(),
              authSessionProvider.overrideWith(SignedInAuthSession.new),
              activeAccountIdentityProvider.overrideWith(
                (_) async => _identity(AccountType.business),
              ),
              businessRepositoryProvider.overrideWithValue(
                _AccountTypeRepository(),
              ),
            ],
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: AccountPage(onDeleteConfirmed: (_) async {}),
            ),
          ),
        );
        await tester.pumpAndSettle();

        expect(find.byType(SegmentedButton<AccountType>), findsNothing);
        expect(find.text('Business'), findsOneWidget);
        await expectKeyboardFocus(tester);

        final delete = find.text('Delete account');
        await tester.ensureVisible(delete);
        await tester.tap(delete);
        await tester.pumpAndSettle();
        expect(find.text('Delete CraftSky account?'), findsOneWidget);
        expect(
          tester
              .getSemantics(find.widgetWithText(ChunkyButton, 'Continue'))
              .getSemanticsData()
              .hasAction(SemanticsAction.tap),
          isTrue,
        );
        expectNoAccessibilityLayoutException(tester);
        semantics.dispose();
      },
    );
  }

  testWidgets(
    'AT-003 account type is read-only and follows loaded profile',
    (tester) async {
      final repository = _AccountTypeRepository();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            ..._businessAccessOverrides(),
            authSessionProvider.overrideWith(SignedInAuthSession.new),
            activeAccountIdentityProvider.overrideWith(
              (_) async => _identity(AccountType.business),
            ),
            businessRepositoryProvider.overrideWithValue(repository),
          ],
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: AccountPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Account type'), findsOneWidget);
      expect(find.text('Business'), findsOneWidget);
      expect(find.byType(SegmentedButton<AccountType>), findsNothing);
      expect(repository.accountTypeUpdates, isEmpty);
      expect(repository.businessProfilePuts, 0);
      expect(repository.eventDeletes, 0);
      expect(find.byType(AlertDialog), findsNothing);
    },
  );

  testWidgets('AT-003 regular account cannot set business type', (
    tester,
  ) async {
    final repository = _AccountTypeRepository();
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          activeAccountIdentityProvider.overrideWith(
            (_) async => _identity(AccountType.regular),
          ),
          businessRepositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MessengerScope(
            messenger: messenger,
            child: child!,
          ),
          home: const AccountPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Regular'), findsOneWidget);
    expect(find.text('Business'), findsNothing);
    expect(find.byType(SegmentedButton<AccountType>), findsNothing);
    expect(repository.accountTypeUpdates, isEmpty);
    expect(messenger.calls, isEmpty);
  });

  testWidgets('Delete account requires both warning and exact typed DID', (
    tester,
  ) async {
    String? confirmedDid;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountPage(
            onDeleteConfirmed: (did) async => confirmedDid = did,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Delete account'));
    await tester.pumpAndSettle();
    expect(find.byType(CraftskyDialog), findsOneWidget);
    expect(find.byType(AlertDialog), findsNothing);
    expect(find.text('Delete CraftSky account?'), findsOneWidget);
    expect(
      find.textContaining('all your CraftSky data from your PDS'),
      findsOneWidget,
    );
    expect(
      find.textContaining('won’t delete your PDS, DID'),
      findsOneWidget,
    );

    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();
    expect(find.byType(CraftskyDialog), findsOneWidget);
    expect(find.byType(AlertDialog), findsNothing);
    await tester.enterText(find.byType(TextField), 'did:plc:tesu');
    await tester.pump();
    ChunkyButton deleteButton() => tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Delete account'),
    );
    expect(deleteButton().onPressed, isNull);

    await tester.enterText(find.byType(TextField), 'did:plc:test');
    await tester.pump();
    expect(deleteButton().onPressed, isNotNull);
    await tester.tap(find.widgetWithText(ChunkyButton, 'Delete account'));
    await tester.pumpAndSettle();
    expect(confirmedDid, 'did:plc:test');
  });
}

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

final class _AccountPageRegistryStorage implements SessionRegistryStorage {
  _AccountPageRegistryStorage(this.registry);
  SessionRegistry registry;
  @override
  Future<SessionRegistry> read() async => registry;
  @override
  Future<void> write(SessionRegistry registry) async =>
      this.registry = registry;
}

List<Override> _businessAccessOverrides() => [
  secureSessionRegistryStorageProvider.overrideWithValue(
    _AccountPageRegistryStorage(
      SessionRegistry.empty().upsertAndActivate(
        token: 'test-token',
        did: 'did:plc:test',
        handle: 'test.bsky.social',
      ),
    ),
  ),
  subscriptionAccessProvider.overrideWith(
    (ref, lease) async => SubscriptionAccess(
      did: lease.account.did,
      effectiveTier: SubscriptionTier.business,
      givesAccess: true,
      assignedTier: SubscriptionTier.business,
    ),
  ),
];

final class _AccountTypeRepository extends Fake implements BusinessRepository {
  final accountTypeUpdates = <AccountType>[];
  int businessProfilePuts = 0;
  int eventDeletes = 0;

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required String operationKey,
    required Cid? expectedCid,
  }) async {
    businessProfilePuts++;
    throw StateError('unexpected business profile mutation');
  }

  @override
  Future<void> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid, {
    required String operationKey,
  }) async {
    eventDeletes++;
    throw StateError('unexpected event deletion');
  }
}
