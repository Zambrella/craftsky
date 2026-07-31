import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

export 'package:craftsky_app/languages/providers/account_language_preferences_provider.dart';

part 'language_preferences_provider.g.dart';

@Riverpod(keepAlive: true)
LanguagePreferences activeLanguagePreferences(Ref ref) {
  final initialized = ref
      .watch(activeAccountInitializationProvider)
      .requireValue;
  if (initialized == null) {
    throw StateError(
      'Active language preferences require an initialized active account',
    );
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
