import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/auth/widgets/active_account_initialization_gate.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

final class _PreferencesRepository implements LanguagePreferencesRepository {
  _PreferencesRepository(this.loaded);

  final Future<LanguagePreferences> loaded;

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) => loaded;

  @override
  Future<LanguagePreferences> load() => loaded;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async => preferences;
}

final class _RetryPreferencesRepository
    implements LanguagePreferencesRepository {
  _RetryPreferencesRepository(this.loadPreferences);

  final Future<LanguagePreferences> Function() loadPreferences;

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) => loadPreferences();

  @override
  Future<LanguagePreferences> load() => loadPreferences();

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async => preferences;
}

final class _RecordingAuthController extends AuthController {
  int signOutCalls = 0;

  @override
  FutureOr<void> build() => null;

  @override
  Future<SignOutResult?> signOut() async {
    signOutCalls++;
    return const SignOutResult.signedOut();
  }
}

void main() {
  testWidgets('signed-out content remains available', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(SessionRegistry.empty()),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ActiveAccountInitializationGate(
            child: Text('signed-out-content'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('signed-out-content'), findsOneWidget);
    expect(find.byType(StitchProgressIndicator), findsNothing);
  });

  testWidgets('restored signed-in content stays unmounted until ready', (
    tester,
  ) async {
    final preferences = Completer<LanguagePreferences>();
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => _PreferencesRepository(preferences.future),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ActiveAccountInitializationGate(
            child: Text('signed-in-content'),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.text('signed-in-content'), findsNothing);

    preferences.complete(
      const LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('signed-in-content'), findsOneWidget);
    expect(find.byType(StitchProgressIndicator), findsNothing);
  });

  testWidgets('failure exposes retry, account switching, and sign out', (
    tester,
  ) async {
    final logRecords = <LogRecord>[];
    final logSubscription = Logger.root.onRecord.listen(logRecords.add);
    addTearDown(logSubscription.cancel);
    final registry = SessionRegistry.empty()
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

    await tester.pumpWidget(
      ProviderScope(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          activeAccountInitializationProvider.overrideWith(
            (ref) async => throw StateError(
              'preferences unavailable for did:plc:secret token-secret',
            ),
          ),
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ActiveAccountInitializationGate(
            child: Text('signed-in-content'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('We couldn’t load this account'), findsOneWidget);
    expect(
      find.text('Try again, switch accounts, or sign out.'),
      findsOneWidget,
    );
    expect(find.widgetWithText(FilledButton, 'Retry'), findsOneWidget);
    expect(
      find.widgetWithText(OutlinedButton, 'Switch account'),
      findsOneWidget,
    );
    expect(find.widgetWithText(TextButton, 'Sign out'), findsOneWidget);
    expect(find.text('signed-in-content'), findsNothing);
    expect(find.textContaining('preferences unavailable'), findsNothing);
    final initializationLogs = logRecords.where(
      (record) =>
          record.loggerName == 'ActiveAccountInitialization' &&
          record.level == Level.SEVERE,
    );
    expect(initializationLogs, hasLength(1));
    expect(
      initializationLogs.single.message,
      'Active account failed to initialize',
    );
    expect(initializationLogs.single.error, isNull);
    expect(initializationLogs.single.stackTrace, isNull);

    await tester.tap(
      find.widgetWithText(OutlinedButton, 'Switch account'),
    );
    await tester.pumpAndSettle();
    expect(find.byType(AccountSwitcherContent), findsOneWidget);
    expect(find.text('alice.test'), findsOneWidget);
    expect(find.text('bob.test'), findsOneWidget);
    expect(find.text('Add account'), findsNothing);

    Navigator.of(
      tester.element(find.byType(AccountSwitcherContent)),
    ).pop();
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Sign out'));
    await tester.pumpAndSettle();
    final authController =
        tester.container().read(authControllerProvider.notifier)
            as _RecordingAuthController;
    expect(authController.signOutCalls, 1);
  });

  testWidgets('retry starts a fresh attempt and remounts ready content', (
    tester,
  ) async {
    const readyPreferences = LanguagePreferences(
      primaryLanguage: 'en',
      contentLanguages: ['en'],
    );
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token-a',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    var loadAttempts = 0;

    await tester.pumpWidget(
      ProviderScope(
        retry: (_, _) => null,
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, account) async => _RetryPreferencesRepository(() async {
              loadAttempts++;
              if (loadAttempts == 1) {
                throw StateError('preferences unavailable');
              }
              return readyPreferences;
            }),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ActiveAccountInitializationGate(
            child: Text('signed-in-content'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('We couldn’t load this account'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Retry'));
    await tester.pumpAndSettle();

    expect(loadAttempts, 2);
    expect(find.text('signed-in-content'), findsOneWidget);
  });
}
