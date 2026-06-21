import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/providers/profile_search_provider.dart';
import 'package:craftsky_app/search/providers/search_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_search_repository.dart';

ProfileSearchResult _profile(String suffix) => ProfileSearchResult(
  did: 'did:plc:$suffix',
  handle: '$suffix.craftsky.social',
  isCraftskyProfile: true,
  viewerIsFollowing: false,
);

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

  test(
    'IT-013 profile loadMore passes cursor, appends, de-dupes, '
    'and no-ops at end',
    () async {
      var calls = 0;
      String? seenCursor;
      final fake = FakeSearchRepository(
        onSearchProfiles: ({required q, limit, cursor}) async {
          calls++;
          if (calls == 1) {
            return ProfileSearchPage(
              items: [_profile('alice')],
              cursor: 'opaque:profiles/+/=',
            );
          }
          seenCursor = cursor;
          return ProfileSearchPage(
            items: [_profile('alice'), _profile('bob')],
          );
        },
      );
      final container = ProviderContainer.test(
        overrides: [searchRepositoryProvider.overrideWithValue(fake)],
      );
      final provider = profileSearchProvider(
        const ProfileSearchQuery(q: 'ali'),
      );

      await container.read(provider.future);
      await container.read(provider.notifier).loadMore();
      await container.read(provider.notifier).loadMore();

      final state = container.read(provider).value!;
      expect(seenCursor, 'opaque:profiles/+/=');
      expect(state.items.map((profile) => profile.did.toString()), [
        'did:plc:alice',
        'did:plc:bob',
      ]);
      expect(state.hasMore, isFalse);
      expect(calls, 2);
    },
  );
}
