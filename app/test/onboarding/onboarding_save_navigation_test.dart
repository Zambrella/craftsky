import 'dart:async';

import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../profile/fakes/fake_profile_repository.dart';

final class _Storage implements SessionRegistryStorage {
  _Storage(this.value);
  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  testWidgets('AT-006 save is single-flight and advances only after success', (
    tester,
  ) async {
    final save = Completer<Profile>();
    var saveCalls = 0;
    await _pumpPage(
      tester,
      onUpdate: () {
        saveCalls++;
        return save.future;
      },
    );

    await tester.enterText(
      find.byKey(const Key('onboarding-display-name')),
      'Alicia',
    );
    await tester.pump();
    await tester.tap(find.text('Save & next'));
    await tester.pump();

    expect(saveCalls, 1);
    expect(
      tester
          .widget<TextButton>(
            find.widgetWithText(TextButton, 'Skip'),
          )
          .onPressed,
      isNull,
    );
    expect(find.text('Step 1 of 3'), findsOneWidget);

    save.complete(_profile(displayName: 'Alicia'));
    await tester.pumpAndSettle();
    expect(find.text('Step 2 of 3'), findsOneWidget);
    expect(saveCalls, 1);
  });

  testWidgets('AT-006 failed save keeps the draft and enables retry', (
    tester,
  ) async {
    final save = Completer<Profile>();
    await _pumpPage(tester, onUpdate: () => save.future);

    await tester.enterText(
      find.byKey(const Key('onboarding-display-name')),
      'Alicia',
    );
    await tester.pump();
    await tester.tap(find.text('Save & next'));
    await tester.pump();
    save.completeError(StateError('offline'));
    await tester.pumpAndSettle();

    expect(find.text('Step 1 of 3'), findsOneWidget);
    expect(find.text('Alicia'), findsOneWidget);
    expect(
      find.text(
        "We couldn't save your profile. Your changes are still here; "
        'try again.',
      ),
      findsOneWidget,
    );
    expect(find.text('Save & next'), findsOneWidget);
    expect(
      tester
          .widget<TextButton>(
            find.widgetWithText(TextButton, 'Skip'),
          )
          .onPressed,
      isNotNull,
    );
  });
}

Future<void> _pumpPage(
  WidgetTester tester, {
  required Future<Profile> Function() onUpdate,
}) async {
  final registry = SessionRegistry.empty().upsertAndActivate(
    token: 'token',
    did: 'did:plc:alice',
    handle: 'alice.test',
  );
  final repository = FakeProfileRepository(
    onFetchMe: () async => _profile(displayName: 'Alice'),
    onUpdateMe:
        ({
          displayName,
          description,
          crafts,
          avatar,
          clearAvatar = false,
          banner,
          clearBanner = false,
        }) => onUpdate(),
  );
  final container = ProviderContainer.test(
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(
        _Storage(registry),
      ),
      accountProfileRepositoryProvider.overrideWith(
        (ref, lease) async => repository,
      ),
      activeAccountInitializationProvider.overrideWith(
        (ref) => ActiveAccountInitialization(
          lease: registry.activeLease!,
          languagePreferences: const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
          onboardingComplete: false,
        ),
      ),
    ],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const OnboardingPage(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Profile _profile({required String displayName}) => Profile(
  did: 'did:plc:alice',
  handle: 'alice.test',
  displayName: displayName,
  crafts: const [],
);
