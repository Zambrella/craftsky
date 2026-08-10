import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/profile_pins_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_customisation_provider.dart';
import 'package:craftsky_app/profile/providers/profile_identity_cache_invalidator.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/projects/providers/project_feed_provider.dart';
import 'package:craftsky_app/search/providers/recent_searches_provider.dart';
import 'package:craftsky_app/search/providers/search_suggestions_provider.dart';
import 'package:craftsky_app/settings/providers/relationship_list_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_profile_repository.dart';

void main() {
  test('IR-002 inventory includes representative retained identity caches', () {
    expect(
      profileIdentityBearingProviderFamilies,
      containsAll(<Object>[
        postProvider,
        postCommentSectionProvider,
        profilePinsProvider,
        searchSuggestionsProvider,
        recentSearchPageProvider,
        relationshipListProvider,
        projectFeedProvider,
      ]),
    );
  });

  test(
    'IR-002 successful authoritative save invokes invalidation once',
    () async {
      var invalidations = 0;
      final repository = FakeProfileRepository(
        onFetchMe: () async => _profile,
        onUpdateCustomisation: (draft) async => draft,
      );
      final container = ProviderContainer(
        overrides: [
          profileRepositoryProvider.overrideWithValue(repository),
          profileIdentityCacheInvalidatorProvider.overrideWithValue(
            () => invalidations += 1,
          ),
        ],
      );
      addTearDown(container.dispose);
      await container.read(profileCustomisationEditorProvider.future);

      final notifier = container.read(
        profileCustomisationEditorProvider.notifier,
      )..selectColour('teal');
      await notifier.save();

      expect(invalidations, 1);
    },
  );
}

final _profile = Profile(
  did: 'did:plc:alice',
  handle: 'alice.example',
  crafts: const [],
);
