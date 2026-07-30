import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/languages/data/device_locale_languages.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/content_language_invalidation.dart';
import 'package:craftsky_app/languages/providers/device_locale_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'language_preferences_provider.g.dart';

@riverpod
class AccountLanguagePreferences extends _$AccountLanguagePreferences {
  bool _replacing = false;
  int _generation = 0;

  @override
  Future<LanguagePreferences> build(AccountKey account) async {
    _generation++;
    _replacing = false;
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
    final previous = state.requireValue;
    final generation = _generation;
    _replacing = true;
    try {
      final repository = await ref.read(
        languagePreferencesRepositoryProvider(account).future,
      );
      final authoritative = await repository.replace(candidate);
      if (!ref.mounted || generation != _generation) return false;
      state = AsyncData(authoritative);
      if (!listEquals(
        previous.contentLanguages,
        authoritative.contentLanguages,
      )) {
        ref.read(contentLanguageCacheInvalidatorProvider)();
      }
      return true;
    } on Object {
      return false;
    } finally {
      if (generation == _generation) _replacing = false;
    }
  }
}

@Riverpod(keepAlive: true)
FutureOr<LanguagePreferences> activeLanguagePreferences(Ref ref) {
  final registry = ref.watch(sessionRegistryProvider);
  if (!registry.hasValue || registry.requireValue.activeDid == null) {
    // Signed-in routes are not reachable without an active registry account.
    // This keeps isolated widget/provider harnesses deterministic without
    // weakening preference loading for an actual activated account.
    return const LanguagePreferences(
      primaryLanguage: 'en',
      contentLanguages: ['en'],
    );
  }
  final active = registry.requireValue.activeDid!;
  return ref.watch(
    accountLanguagePreferencesProvider(AccountKey(active.value)).future,
  );
}

@Riverpod(keepAlive: true)
FutureOr<List<String>> activeContentLanguagePolicy(Ref ref) {
  final preferences = ref.watch(activeLanguagePreferencesProvider);
  if (preferences.hasValue) {
    return preferences.requireValue.contentLanguages;
  }
  return ref
      .watch(activeLanguagePreferencesProvider.future)
      .then((value) => value.contentLanguages);
}
