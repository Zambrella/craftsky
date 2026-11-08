import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

export 'package:craftsky_app/languages/providers/account_language_preferences_provider.dart';

part 'language_preferences_provider.g.dart';

@Riverpod(keepAlive: true)
LanguagePreferences activeLanguagePreferences(Ref ref) {
  final initialization = ref.watch(activeAccountInitializationProvider);
  // Logout/account activation has no active preferences until initialization
  // finishes. Keep watching it so dependents rebuild when the account is ready.
  if (!initialization.hasValue && !initialization.hasError) {
    throw const ApiCanceled();
  }
  final initialized = initialization.requireValue;
  if (initialized == null) {
    throw const ApiCanceled();
  }
  return initialized.languagePreferences;
}

@Riverpod(keepAlive: true)
class ActiveContentLanguagePolicy extends _$ActiveContentLanguagePolicy {
  @override
  List<String> build() => List.unmodifiable(
    ref.watch(activeLanguagePreferencesProvider).contentLanguages,
  );

  @override
  bool updateShouldNotify(List<String> previous, List<String> next) =>
      !listEquals(previous, next);
}
