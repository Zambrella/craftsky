import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/providers/content_language_invalidation.dart';
import 'package:craftsky_app/languages/providers/device_locale_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../feed/fakes/fake_post_repository.dart';

final class _FlowPreferencesRepository
    implements LanguagePreferencesRepository {
  LanguagePreferences? stored;

  @override
  Future<LanguagePreferences> load() async {
    final value = stored;
    if (value == null) {
      throw const ApiBadRequest('language_preferences_not_found');
    }
    return value;
  }

  @override
  Future<LanguagePreferences> initialize(
    LanguagePreferences proposal,
  ) async => stored ??= proposal;

  @override
  Future<LanguagePreferences> replace(
    LanguagePreferences preferences,
  ) async => stored = preferences;
}

Post _createdPost() => PostMapper.fromMap({
  'uri': 'at://did:plc:alice/social.craftsky.feed.post/new',
  'cid': 'bafy-new',
  'rkey': 'new',
  'text': 'Hola, bonjour',
  'langs': <String>[],
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'createdAt': '2026-07-29T10:00:00.000Z',
  'indexedAt': '2026-07-29T10:00:01.000Z',
  'author': {
    'did': 'did:plc:alice',
    'handle': 'alice.craftsky.social',
  },
});

void main() {
  setUpAll(initializeMappers);

  test(
    'AT-015 composes bootstrap, preferences, publish, and browse readiness',
    () async {
      final account = AccountKey('did:plc:alice');
      final preferencesRepository = _FlowPreferencesRepository();
      final postRepository = FakePostRepository(
        onCreate: ({required text, reply, images}) async => _createdPost(),
        onListTimeline: ({cursor, limit}) async =>
            const TimelinePage(items: []),
      );
      var invalidations = 0;
      final observedPolicies = <List<String>>[];

      final container = ProviderContainer.test(
        overrides: [
          deviceLocalesProvider.overrideWithValue(const [
            Locale('fr', 'CA'),
            Locale('en', 'GB'),
          ]),
          languagePreferencesRepositoryProvider.overrideWith(
            (ref, requestedAccount) async {
              expect(requestedAccount, account);
              return preferencesRepository;
            },
          ),
          contentLanguageCacheInvalidatorProvider.overrideWithValue(
            () => invalidations++,
          ),
          activeContentLanguagePolicyProvider.overrideWith((ref) async {
            final value = await ref.watch(
              accountLanguagePreferencesProvider(account).future,
            );
            observedPolicies.add(value.contentLanguages);
            return value.contentLanguages;
          }),
          postRepositoryProvider.overrideWithValue(postRepository),
        ],
      );

      expect(
        await container.read(
          accountLanguagePreferencesProvider(account).future,
        ),
        const LanguagePreferences(
          primaryLanguage: 'fr',
          contentLanguages: ['fr', 'en'],
        ),
      );

      const changed = LanguagePreferences(
        primaryLanguage: 'es',
        contentLanguages: ['es', 'fr'],
      );
      expect(
        await container
            .read(accountLanguagePreferencesProvider(account).notifier)
            .replace(changed),
        isTrue,
      );
      expect(preferencesRepository.stored, changed);
      expect(invalidations, 1);

      final selection = PostLanguageSelection.fromPrimary(
        changed.primaryLanguage,
      ).add('fr');
      await container
          .read(createPostProvider.notifier)
          .create(text: 'Hola, bonjour', langs: selection.values);
      expect(postRepository.lastCreateLangs, ['es', 'fr']);
      expect(container.read(createPostProvider).value?.langs, ['es', 'fr']);

      await container.read(timelineProvider.future);
      expect(observedPolicies.last, ['es', 'fr']);
    },
  );
}
