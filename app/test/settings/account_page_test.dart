import 'dart:async';
import 'dart:ui' show SemanticsAction, Tristate;

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/settings/pages/account_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../business/accessibility_test_helpers.dart';
import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';

void main() {
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
              authSessionProvider.overrideWith(SignedInAuthSession.new),
              activeAccountIdentityProvider.overrideWith(
                (_) async => _identity(AccountType.business),
              ),
              businessRepositoryProvider.overrideWithValue(
                _AccountTypeRepository(),
              ),
            ],
            child: MaterialApp(
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: AccountPage(onDeleteConfirmed: (_) async {}),
            ),
          ),
        );
        await tester.pumpAndSettle();

        final selector = find.byType(SegmentedButton<AccountType>);
        expect(selector, findsOneWidget);
        expect(
          tester.getSemantics(find.text('Business')).flagsCollection.isSelected,
          Tristate.isTrue,
        );
        expect(
          tester
              .getSemantics(find.text('Regular'))
              .getSemanticsData()
              .hasAction(SemanticsAction.tap),
          isTrue,
        );
        await expectKeyboardFocus(tester);

        final delete = find.text('Delete account');
        await tester.ensureVisible(delete);
        await tester.tap(delete);
        await tester.pumpAndSettle();
        expect(find.text('Delete CraftSky account?'), findsOneWidget);
        expect(
          tester
              .getSemantics(find.widgetWithText(FilledButton, 'Continue'))
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
    'AT-001 account selector is authoritative, single-flight, and reversible',
    (tester) async {
      final repository = _AccountTypeRepository();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
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
      expect(find.text('Regular'), findsOneWidget);
      expect(find.text('Business'), findsOneWidget);

      await tester.tap(find.text('Regular'));
      await tester.pump();

      final selector = tester.widget<SegmentedButton<AccountType>>(
        find.byType(SegmentedButton<AccountType>),
      );
      expect(selector.onSelectionChanged, isNull);
      expect(repository.accountTypeUpdates, [AccountType.regular]);
      expect(repository.businessProfilePuts, 0);
      expect(repository.eventDeletes, 0);
      expect(find.byType(AlertDialog), findsNothing);

      repository.complete(AccountType.regular);
      await tester.pumpAndSettle();

      final settled = tester.widget<SegmentedButton<AccountType>>(
        find.byType(SegmentedButton<AccountType>),
      );
      expect(settled.selected, {AccountType.regular});
      expect(settled.onSelectionChanged, isNotNull);
    },
  );

  testWidgets('AT-001 failure keeps selection and restores input', (
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

    await tester.tap(find.text('Business'));
    await tester.pump();
    repository.fail(StateError('private failure'));
    await tester.pumpAndSettle();

    final selector = tester.widget<SegmentedButton<AccountType>>(
      find.byType(SegmentedButton<AccountType>),
    );
    expect(selector.selected, {AccountType.regular});
    expect(selector.onSelectionChanged, isNotNull);
    expect(messenger.calls, [
      ('error', "That didn't work. Please try again.", null),
    ]);
  });

  testWidgets('Delete account requires both warning and exact typed handle', (
    tester,
  ) async {
    String? confirmedHandle;
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authSessionProvider.overrideWith(SignedInAuthSession.new),
        ],
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AccountPage(
            onDeleteConfirmed: (handle) async => confirmedHandle = handle,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Delete account'));
    await tester.pumpAndSettle();
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
    await tester.enterText(find.byType(TextField), '@Test.bsky.social');
    await tester.pump();
    FilledButton deleteButton() => tester.widget<FilledButton>(
      find.widgetWithText(FilledButton, 'Delete account'),
    );
    expect(deleteButton().onPressed, isNull);

    await tester.enterText(find.byType(TextField), '@test.bsky.social');
    await tester.pump();
    expect(deleteButton().onPressed, isNotNull);
    await tester.tap(find.widgetWithText(FilledButton, 'Delete account'));
    await tester.pumpAndSettle();
    expect(confirmedHandle, '@test.bsky.social');
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

final class _AccountTypeRepository extends Fake implements BusinessRepository {
  final accountTypeUpdates = <AccountType>[];
  int businessProfilePuts = 0;
  int eventDeletes = 0;
  late Completer<AccountType> _completion;

  @override
  Future<AccountType> updateAccountType(AccountType value) {
    accountTypeUpdates.add(value);
    _completion = Completer<AccountType>();
    return _completion.future;
  }

  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required Cid? expectedCid,
  }) async {
    businessProfilePuts++;
    throw StateError('unexpected business profile mutation');
  }

  @override
  Future<RecordMutationResult> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid,
  ) async {
    eventDeletes++;
    throw StateError('unexpected event deletion');
  }

  void complete(AccountType value) => _completion.complete(value);

  void fail(Object error) => _completion.completeError(error);
}
