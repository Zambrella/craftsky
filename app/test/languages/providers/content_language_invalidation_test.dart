import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/content_language_invalidation.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

final class _ReplacingRepository implements LanguagePreferencesRepository {
  LanguagePreferences value = const LanguagePreferences(
    primaryLanguage: 'en',
    contentLanguages: ['en'],
  );
  bool failNextReplacement = false;

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) async => value;

  @override
  Future<LanguagePreferences> load() async => value;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async {
    if (failNextReplacement) {
      failNextReplacement = false;
      throw StateError('unavailable');
    }
    return value = preferences;
  }
}

void main() {
  test('UT-013 enumerates exactly the eight filtered cache families', () {
    expect(contentLanguageCacheInventory, [
      ContentLanguageCache.timeline,
      ContentLanguageCache.projectsBrowse,
      ContentLanguageCache.postSearch,
      ContentLanguageCache.projectSearch,
      ContentLanguageCache.hashtagPosts,
      ContentLanguageCache.profilePosts,
      ContentLanguageCache.profileProjects,
      ContentLanguageCache.profileComments,
    ]);
  });

  test(
    'IT-018 invalidates only after authoritative Content replacement',
    () async {
      final repository = _ReplacingRepository();
      var invalidations = 0;
      final account = AccountKey('did:plc:alice');
      final container = ProviderContainer.test(
        overrides: [
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, requestedAccount) async => repository,
          ),
          contentLanguageCacheInvalidatorProvider.overrideWithValue(
            () => invalidations++,
          ),
        ],
      );
      final provider = accountLanguagePreferencesProvider(account);
      await container.read(provider.future);

      expect(
        await container
            .read(provider.notifier)
            .replace(
              const LanguagePreferences(
                primaryLanguage: 'fr',
                contentLanguages: ['en'],
              ),
            ),
        isTrue,
      );
      expect(invalidations, 0);

      expect(
        await container
            .read(provider.notifier)
            .replace(
              const LanguagePreferences(
                primaryLanguage: 'fr',
                contentLanguages: ['fr'],
              ),
            ),
        isTrue,
      );
      expect(invalidations, 1);
      expect(container.read(provider).requireValue, repository.value);

      repository.failNextReplacement = true;
      expect(
        await container
            .read(provider.notifier)
            .replace(
              const LanguagePreferences(
                primaryLanguage: 'cy',
                contentLanguages: ['cy'],
              ),
            ),
        isFalse,
      );
      expect(invalidations, 1);
      expect(
        container.read(provider).requireValue,
        const LanguagePreferences(
          primaryLanguage: 'fr',
          contentLanguages: ['fr'],
        ),
      );
    },
  );
}
