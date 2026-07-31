import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/languages/providers/account_language_preferences_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'active_account_initialization_provider.g.dart';

/// Resolves the account-critical state for the exact active session lease.
///
/// A signed-out registry is a successful initialization with no account.
/// Loading and failures from either dependency remain visible to the gate.
@Riverpod(keepAlive: true)
FutureOr<ActiveAccountInitialization?> activeAccountInitialization(Ref ref) {
  final registry = ref.watch(sessionRegistryProvider).requireValue;
  final lease = registry.activeLease;
  if (lease == null) return null;

  final preferences = ref
      .watch(
        accountLanguagePreferencesProvider(lease),
      )
      .requireValue;
  return ActiveAccountInitialization(
    lease: lease,
    languagePreferences: preferences,
  );
}
