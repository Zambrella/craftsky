import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/pages/languages_page.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/pages/notification_settings_page.dart';
import 'package:craftsky_app/notifications/providers/notification_permission_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/settings/pages/appearance_page.dart';
import 'package:craftsky_app/settings/pages/profile_customisation_page.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../fakes/auth_session_fakes.dart';
import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';
import '../support/native_reference_capture.dart';

// Explicitly opt in: temporary review artifacts, not golden baselines.
const _capture = bool.fromEnvironment('CAPTURE_NATIVE_REFERENCES');

final SessionRegistry _registry = SessionRegistry.empty().upsertAndActivate(
  token: 'visual-fixture-token',
  did: 'did:plc:test',
  handle: 'alice.example',
);
final _profile = Profile(
  did: 'did:plc:test',
  handle: 'alice.example',
  displayName: 'Alice',
  crafts: const ['knitting', 'sewing'],
);

void main() {
  for (final entry in <String, Widget>{
    'settings-index': const SettingsPage(),
    'settings-appearance': const AppearancePage(),
    'settings-languages': const LanguagesPage(),
    'settings-notifications': const NotificationSettingsPage(),
    'settings-customisation': const ProfileCustomisationPage(),
  }.entries) {
    testWidgets(
      'capture native ${entry.key}',
      (tester) async {
        await prepareNativeReference(tester);
        SharedPreferences.setMockInitialValues({});
        final preferences = await SharedPreferences.getInstance();
        final languages = _Languages();
        final notifications = _Notifications();
        final profile = FakeProfileRepository(
          onFetchMe: () async => _profile,
          onUpdateCustomisation: (value) async => value,
        );
        final nested = entry.key != 'settings-index';
        await tester.pumpWidget(
          NativeReferenceFrame(
            child: ProviderScope(
              overrides: [
                sharedPreferencesProvider.overrideWithValue(preferences),
                secureSessionRegistryStorageProvider.overrideWithValue(
                  _RegistryStorage(),
                ),
                authSessionProvider.overrideWith(SignedInAuthSession.new),
                activeAccountIdentityProvider.overrideWith(
                  (ref) async => ActiveAccountIdentity(
                    lease: _registry.activeLease!.session,
                    profile: _profile,
                  ),
                ),
                activeAccountInitializationProvider.overrideWith(
                  (ref) => ActiveAccountInitialization(
                    lease: _registry.activeLease!,
                    languagePreferences: languages.value,
                    onboardingComplete: true,
                  ),
                ),
                languagePreferencesRepositoryProvider.overrideWith(
                  (ref, account) async => languages,
                ),
                profileRepositoryProvider.overrideWithValue(profile),
                subscriptionAccessProvider.overrideWith(
                  (ref, lease) async => SubscriptionAccess(
                    did: lease.account.did,
                    effectiveTier: SubscriptionTier.business,
                    givesAccess: true,
                    assignedTier: SubscriptionTier.business,
                  ),
                ),
                notificationPermissionProvider.overrideWith(
                  (ref) async => NotificationPermission.authorized,
                ),
                notificationPreferencesRepositoryProvider.overrideWithValue(
                  notifications,
                ),
                accountNotificationPreferencesRepositoryProvider.overrideWith(
                  (ref, account) async => notifications,
                ),
              ],
              child: MessengerScope(
                messenger: RecordingMessenger(),
                child: MaterialApp(
                  debugShowCheckedModeBanner: false,
                  theme: AppTheme.lightThemeData,
                  localizationsDelegates: appLocalizationsDelegates,
                  supportedLocales: AppLocalizations.supportedLocales,
                  initialRoute: nested ? '/capture' : '/',
                  routes: {
                    '/': (_) => nested ? const Scaffold() : entry.value,
                    if (nested) '/capture': (_) => entry.value,
                  },
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        await saveNativeReference(tester, entry.key);
        if (entry.key == 'settings-index' ||
            entry.key == 'settings-notifications') {
          await tester.drag(
            find.byType(Scrollable).first,
            const Offset(0, -1400),
          );
          await tester.pumpAndSettle();
          await saveNativeReference(tester, '${entry.key}-lower');
        }
      },
      skip: !_capture,
      variant: TargetPlatformVariant.only(TargetPlatform.iOS),
    );
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  @override
  Future<SessionRegistry> read() async => _registry;
  @override
  Future<void> write(SessionRegistry registry) async {}
}

final class _Languages implements LanguagePreferencesRepository {
  LanguagePreferences value = const LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en', 'fr'],
  );
  @override
  Future<LanguagePreferences> load() async => value;
  @override
  Future<LanguagePreferences> initialize(LanguagePreferences proposal) async =>
      value;
  @override
  Future<LanguagePreferences> replace(LanguagePreferences preferences) async =>
      value = preferences;
}

final class _Notifications implements NotificationPreferencesRepository {
  NotificationPreferences value = NotificationPreferences(
    known: {
      for (final category in NotificationCategory.preferenceValues)
        category: const NotificationPreference(
          scope: NotificationPreferenceScope.everyone,
          pushEnabled: true,
        ),
    },
    unknown: const {},
  );
  @override
  Future<NotificationPreferences> load() async => value;
  @override
  Future<NotificationPreferences> patch(
    NotificationPreferencePatch patch,
  ) async => value;
}
