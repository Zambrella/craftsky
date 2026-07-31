import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/languages/data/device_locale_languages.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/device_locale_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'account_language_preferences_provider.g.dart';

@riverpod
class AccountLanguagePreferences extends _$AccountLanguagePreferences {
  bool _replacing = false;
  int _generation = 0;

  @override
  Future<LanguagePreferences> build(ActiveAccountLease lease) async {
    _generation++;
    _replacing = false;
    final account = lease.session.account;
    final repository = await ref.watch(
      languagePreferencesRepositoryProvider(account).future,
    );
    try {
      return await repository.load();
    } on ApiBadRequest catch (error) {
      if (error.code != 'language_preferences_not_found') rethrow;
      final proposal = deriveInitialLanguages(ref.read(deviceLocalesProvider));
      return repository.initialize(proposal);
    }
  }

  Future<bool> replace(LanguagePreferences candidate) async {
    if (_replacing || !state.hasValue) return false;
    final generation = _generation;
    _replacing = true;
    try {
      final repository = await ref.read(
        languagePreferencesRepositoryProvider(lease.session.account).future,
      );
      final authoritative = await repository.replace(candidate);
      if (!ref.mounted || generation != _generation) return false;
      state = AsyncData(authoritative);
      return true;
    } on Object {
      return false;
    } finally {
      if (generation == _generation) _replacing = false;
    }
  }
}
