import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/languages/data/device_locale_languages.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/device_locale_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_language_preferences_provider.g.dart';

typedef AccountLanguagePreferencesState = ({
  LanguagePreferences preferences,
  AsyncValue<void> replacement,
});

@riverpod
class AccountLanguagePreferences extends _$AccountLanguagePreferences {
  int _generation = 0;

  @override
  Future<AccountLanguagePreferencesState> build(
    ActiveAccountLease lease,
  ) async {
    _generation++;
    final account = lease.session.account;
    final repository = await ref.watch(
      languagePreferencesRepositoryProvider(account).future,
    );
    late final LanguagePreferences preferences;
    try {
      preferences = await repository.load();
    } on ApiBadRequest catch (error) {
      if (error.code != 'language_preferences_not_found') rethrow;
      final proposal = deriveInitialLanguages(ref.read(deviceLocalesProvider));
      preferences = await repository.initialize(proposal);
    }
    return (
      preferences: preferences,
      replacement: const AsyncData(null),
    );
  }

  Future<void> replace(LanguagePreferences candidate) async {
    final previous = state.value;
    if (previous == null || previous.replacement.isLoading) return;
    final generation = _generation;
    state = AsyncData((
      preferences: previous.preferences,
      replacement: const AsyncLoading(),
    ));
    try {
      final repository = await ref.read(
        languagePreferencesRepositoryProvider(lease.session.account).future,
      );
      final authoritative = await repository.replace(candidate);
      if (!ref.mounted || generation != _generation) return;
      state = AsyncData((
        preferences: authoritative,
        replacement: const AsyncData(null),
      ));
    } on Object catch (error, stackTrace) {
      if (!ref.mounted || generation != _generation) return;
      state = AsyncData((
        preferences: previous.preferences,
        replacement: AsyncError<void>(error, stackTrace),
      ));
    }
  }
}
