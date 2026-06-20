import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/profile_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

void main() {
  test(
    'IT-012 profile provider fetches initial state through repository',
    () async {
      String? seenQ;
      final fake = FakeSearchRepository(
        onSearchProfiles: ({required q, limit, cursor}) async {
          seenQ = q;
          return ProfileSearchPage(
            items: [
              ProfileSearchResult(
                did: 'did:plc:alice',
                handle: 'alice.craftsky.social',
                isCraftskyProfile: true,
                viewerIsFollowing: false,
              ),
            ],
            cursor: 'next',
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );

      final state = await container.read(
        profileSearchProvider(const ProfileSearchQuery(q: 'ali')).future,
      );

      expect(seenQ, 'ali');
      expect(state.items.single.did.toString(), 'did:plc:alice');
      expect(state.hasMore, isTrue);
    },
  );
}
