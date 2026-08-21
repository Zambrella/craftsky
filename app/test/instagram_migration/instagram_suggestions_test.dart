import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as auth;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_repository.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/pages/instagram_migration_page.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_suggestions_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'IT-029 lists private suggestions and requires explicit Follow or Dismiss',
    (tester) async {
      tester.view.physicalSize = const Size(800, 1600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final initial = _twoAccountRegistry();
      final suggestion = _suggestion();
      final dismissible = _suggestion(
        id: 'suggestion-b',
        did: 'did:plc:dismissible',
        handle: 'dismissible.synthetic.invalid',
        displayName: 'Dismissible Maker',
      );
      final accepted = <String>[];
      final dismissed = <String>[];
      final repository = _SuggestionRepository(
        suggestions: InstagramSuggestionPage(
          items: [suggestion, dismissible],
          cursor: null,
        ),
        onAccept: (id) async {
          accepted.add(id);
          return InstagramSuggestionActionResult(
            suggestionId: id,
            state: InstagramSuggestionState.followed,
          );
        },
        onDismiss: (id) async => dismissed.add(id),
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(initial),
            ),
            instagramVerificationStorageProvider.overrideWithValue(
              _EmptyVerificationStorage(),
            ),
            instagramMigrationRepositoryProvider.overrideWith(
              (ref, _) async => repository,
            ),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const InstagramMigrationPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text(
          'Your imports find possible CraftSky accounts privately. Nobody is '
          'followed until you choose Follow.',
        ),
        findsOneWidget,
      );
      expect(find.text('Synthetic Maker'), findsOneWidget);
      expect(find.text('@maker.synthetic.invalid'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Follow'), findsNWidgets(2));
      expect(find.widgetWithText(TextButton, 'Dismiss'), findsNWidgets(2));
      expect(accepted, isEmpty);
      expect(dismissed, isEmpty);

      final follow = find.widgetWithText(FilledButton, 'Follow').first;
      await tester.tap(follow);
      await tester.pumpAndSettle();

      expect(accepted, [suggestion.suggestionId]);
      expect(dismissed, isEmpty);
      expect(find.text('Synthetic Maker'), findsNothing);
      expect(find.text('Dismissible Maker'), findsOneWidget);

      await tester.tap(find.widgetWithText(TextButton, 'Dismiss'));
      await tester.pumpAndSettle();

      expect(dismissed, [dismissible.suggestionId]);
      expect(find.text('Dismissible Maker'), findsNothing);
    },
  );

  test(
    'IT-029 late Follow result cannot mutate a later account activation',
    () async {
      final initial = _twoAccountRegistry();
      final aliceLease = initial.activeLease!;
      final completion = Completer<InstagramSuggestionActionResult>();
      final repository = _SuggestionRepository(
        suggestions: InstagramSuggestionPage(
          items: [_suggestion()],
          cursor: null,
        ),
        onAccept: (_) => completion.future,
      );
      final container = ProviderContainer.test(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(initial),
          ),
          instagramMigrationRepositoryProvider.overrideWith(
            (ref, _) async => repository,
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await container.read(instagramSuggestionsProvider(aliceLease).future);

      final accept = container
          .read(instagramSuggestionsProvider(aliceLease).notifier)
          .accept('suggestion-a');
      await Future<void>.delayed(Duration.zero);
      final bobLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:bob'))!;
      await container.read(sessionRegistryProvider.notifier).activate(bobLease);
      final retainedAlice = container
          .read(sessionRegistryProvider)
          .requireValue
          .leaseFor(AccountKey('did:plc:alice'))!;
      await container
          .read(sessionRegistryProvider.notifier)
          .activate(retainedAlice);
      final laterAliceLease = container
          .read(sessionRegistryProvider)
          .requireValue
          .activeLease!;

      completion.complete(
        const InstagramSuggestionActionResult(
          suggestionId: 'suggestion-a',
          state: InstagramSuggestionState.followed,
        ),
      );

      expect(await accept, isFalse);
      expect(laterAliceLease, isNot(aliceLease));
      final laterPage = await container.read(
        instagramSuggestionsProvider(laterAliceLease).future,
      );
      expect(laterPage.items.single.suggestionId, 'suggestion-a');
    },
  );
}

auth.SessionRegistry _twoAccountRegistry() => auth.SessionRegistry.empty()
    .upsertAndActivate(
      token: 'token-b',
      did: 'did:plc:bob',
      handle: 'bob.test',
    )
    .upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );

InstagramSuggestion _suggestion({
  String id = 'suggestion-a',
  String did = 'did:plc:maker',
  String handle = 'maker.synthetic.invalid',
  String displayName = 'Synthetic Maker',
}) => InstagramSuggestion(
  suggestionId: id,
  target: InstagramSuggestionTarget(
    did: did,
    handle: handle,
    displayName: displayName,
  ),
  createdAt: DateTime.utc(2026, 8, 14),
);

final class _SuggestionRepository implements InstagramMigrationRepository {
  const _SuggestionRepository({
    required this.suggestions,
    this.onAccept,
    this.onDismiss,
  });

  final InstagramSuggestionPage suggestions;
  final Future<InstagramSuggestionActionResult> Function(String id)? onAccept;
  final Future<void> Function(String id)? onDismiss;

  @override
  Future<InstagramAccountStatus> getAccount() async => InstagramAccountStatus(
    integrationAvailable: true,
    account: InstagramAccountLink(
      state: InstagramAccountLinkState.active,
      username: 'alice_instagram',
      discoverable: true,
      conflictPending: false,
      reactivationRequired: false,
      verifiedAt: DateTime.utc(2026, 8, 14),
    ),
  );

  @override
  Future<InstagramImportPage> listImports({int? limit, String? cursor}) async =>
      InstagramImportPage(items: const [], cursor: null);

  @override
  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) async => suggestions;

  @override
  Future<InstagramSuggestionActionResult> acceptSuggestion(String id) =>
      onAccept!.call(id);

  @override
  Future<void> dismissSuggestion(String id) async {
    await onDismiss?.call(id);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  auth.SessionRegistry value;

  @override
  Future<auth.SessionRegistry> read() async => value;

  @override
  Future<void> write(auth.SessionRegistry registry) async => value = registry;
}

final class _EmptyVerificationStorage implements InstagramVerificationStorage {
  @override
  Future<void> delete(AccountKey account, {String? verificationId}) async {}

  @override
  Future<InstagramVerificationSnapshot?> read(AccountKey account) async => null;

  @override
  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  ) async {}
}
