import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('every new Settings and account-deletion string is localized', () {
    final l10n = AppLocalizationsEn();
    final values = <String>[
      l10n.settingsTitle,
      l10n.settingsSwitchAccount,
      l10n.settingsSectionPreferences,
      l10n.settingsSectionConnections,
      l10n.settingsSectionDiscovery,
      l10n.settingsSectionGeneral,
      l10n.settingsNotifications,
      l10n.settingsFollowers,
      l10n.settingsFollowing,
      l10n.settingsAccount,
      l10n.settingsAbout,
      l10n.settingsTerms,
      l10n.settingsPrivacyPolicy,
      l10n.settingsClearImageCache,
      l10n.settingsImageCacheCleared,
      l10n.settingsVersion,
      l10n.settingsSignOut,
      l10n.accountTitle,
      l10n.deleteAccountTitle,
      l10n.deleteAccountAction,
      l10n.deleteAccountContinue,
      l10n.deleteAccountConfirmTitle,
      l10n.deleteAccountTypeHandleLabel,
      l10n.deleteAccountConfirmationPrompt('@alice.test'),
      l10n.actionCancel,
      l10n.accountDeletionPreparing,
      l10n.accountDeletionRemovingPrivateData,
      l10n.accountDeletionRemovingRecords,
      l10n.accountDeletionWaitingForCraftsky,
      l10n.accountDeletionFinalizing,
      l10n.accountDeletionRetrying,
      l10n.accountDeletionNeedsAttention,
      l10n.accountDeletionDeleted,
      l10n.accountDeletionRetry,
      l10n.accountDeletionReauthenticate,
      l10n.accountDeletionSupport,
    ];

    expect(values, everyElement(isNotEmpty));
  });
}
