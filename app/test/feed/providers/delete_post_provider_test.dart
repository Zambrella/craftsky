import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/providers/delete_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/fake_post_repository.dart';

final _aliceDid = Did.parse('did:plc:alice');

Map<String, dynamic> _postMap({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  Project? project,
}) => {
  'uri': 'at://$did/social.craftsky.feed.post/$rkey',
  'cid': 'bafy_$rkey',
  'rkey': rkey,
  'text': 'post $rkey',
  'tags': <String>[],
  'likeCount': 0,
  'repostCount': 0,
  'replyCount': 0,
  'viewerHasLiked': false,
  'viewerHasReposted': false,
  'viewerHasSaved': false,
  'sponsored': false,
  'createdAt': '2026-05-04T18:23:45.000Z',
  'indexedAt': '2026-05-04T18:23:47.000Z',
  'author': {'did': did, 'handle': handle},
  if (project != null) 'project': project.toMap(),
};

Post _post({
  required String rkey,
  String did = 'did:plc:alice',
  String handle = 'alice.craftsky.social',
  Project? project,
}) => PostMapper.fromMap(
  _postMap(rkey: rkey, did: did, handle: handle, project: project),
);

const _project = Project(
  common: ProjectCommon(craftType: 'social.craftsky.feed.defs#embroidery'),
);

void main() {
  setUpAll(initializeMappers);

  group('DeletePost', () {
    test('retries ambiguity with one key and unchanged CID guard', () async {
      var calls = 0;
      final fake = FakePostRepository(
        onDelete: (did, rkey) async {
          calls++;
          if (calls < 3) {
            throw const PdsMutationAmbiguousException(retryAfterSeconds: 1);
          }
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(fake),
          pdsMutationDelayProvider.overrideWithValue((_) async {}),
          pdsMutationJitterProvider.overrideWithValue((_) => 0),
        ],
      );
      final post = _post(rkey: 'guarded');

      await container.read(deletePostProvider.notifier).delete(post: post);

      expect(fake.deleteOperationKeys, hasLength(3));
      expect(fake.deleteOperationKeys.toSet(), hasLength(1));
      expect(fake.deleteExpectedCids, everyElement(post.cid.toString()));
      expect(
        isCanonicalPdsMutationOperationKey(fake.deleteOperationKeys.first),
        isTrue,
      );
    });

    test(
      'exhaustion is not success and explicit retry keeps the key',
      () async {
        var calls = 0;
        final fake = FakePostRepository(
          onDelete: (did, rkey) async {
            calls++;
            if (calls <= 7) {
              throw const PdsMutationAmbiguousException(retryAfterSeconds: 1);
            }
          },
        );
        final container = ProviderContainer.test(
          overrides: [
            postRepositoryProvider.overrideWithValue(fake),
            pdsMutationDelayProvider.overrideWithValue((_) async {}),
            pdsMutationJitterProvider.overrideWithValue((_) => 0),
          ],
        );
        final post = _post(rkey: 'unresolved');

        await container.read(deletePostProvider.notifier).delete(post: post);

        expect(calls, 7);
        expect(
          container.read(deletePostProvider).error,
          isA<PdsMutationUnresolvedException>(),
        );

        await container.read(deletePostProvider.notifier).delete(post: post);

        expect(calls, 8);
        expect(fake.deleteOperationKeys.toSet(), hasLength(1));
        expect(fake.deleteExpectedCids, everyElement(post.cid.toString()));
        expect(container.read(deletePostProvider).value, post);
      },
    );

    test('idle build returns null', () async {
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(FakePostRepository()),
        ],
      );

      expect(container.read(deletePostProvider).value, isNull);
    });

    test('successful delete removes from the live DID family entry', () async {
      final deleted = <(String, String)>[];
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async => PostPage(
          items: [
            _post(rkey: 'a'),
            _post(rkey: 'b'),
          ],
        ),
        onDelete: (did, rkey) async {
          deleted.add((did, rkey));
        },
      );
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(fake),
        ],
      );

      await container.read(userPostsProvider(_aliceDid).future);

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));

      container.invalidate(userPostsProvider(_aliceDid));
      final staleRefresh = await container.read(
        userPostsProvider(_aliceDid).future,
      );
      expect(staleRefresh.items.map((post) => post.rkey), ['b']);

      expect(deleted, [('did:plc:alice', 'a')]);

      final didList = container.read(userPostsProvider(_aliceDid)).value!;
      final handleList = container.read(userPostsProvider(_aliceDid)).value!;
      expect(didList.items.map((p) => p.rkey), ['b']);
      expect(handleList.items.map((p) => p.rkey), ['b']);
    });

    test('failure surfaces as AsyncError, cache untouched', () async {
      final fake = FakePostRepository(
        onListByAuthor: (id, {cursor, limit}) async =>
            PostPage(items: [_post(rkey: 'a')]),
        onDelete: (did, rkey) async => throw Exception('boom'),
      );
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(fake),
        ],
      );

      await container.read(userPostsProvider(_aliceDid).future);

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));

      expect(container.read(deletePostProvider).hasError, isTrue);
      final list = container.read(userPostsProvider(_aliceDid)).value!;
      expect(list.items.map((p) => p.rkey), ['a']);
    });

    test('reset() returns to AsyncData(null)', () async {
      final fake = FakePostRepository(onDelete: (did, rkey) async {});
      final container = ProviderContainer.test(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          postRepositoryProvider.overrideWithValue(fake),
        ],
      );

      await container
          .read(deletePostProvider.notifier)
          .delete(post: _post(rkey: 'a'));
      expect(container.read(deletePostProvider).value?.rkey, 'a');

      container.read(deletePostProvider.notifier).reset();
      expect(container.read(deletePostProvider).value, isNull);
    });

    test(
      'IT-008 removes project posts from the DID project cache only',
      () async {
        final fake = FakePostRepository(
          onListByAuthor: (id, {cursor, limit}) async =>
              PostPage(items: [_post(rkey: 'a')]),
          onListProjectsByAuthor: (id, {cursor, limit}) async => PostPage(
            items: [_post(rkey: 'project', project: _project)],
          ),
          onDelete: (did, rkey) async {},
        );
        final container = ProviderContainer.test(
          overrides: [
            activeLanguagePreferencesProvider.overrideWith(
              (ref) => const LanguagePreferences(
                primaryLanguage: 'en',
                contentLanguages: ['en'],
              ),
            ),
            postRepositoryProvider.overrideWithValue(fake),
          ],
        );
        await container.read(userPostsProvider(_aliceDid).future);
        await container.read(userProjectsProvider(_aliceDid).future);

        await container
            .read(deletePostProvider.notifier)
            .delete(
              post: _post(rkey: 'project', project: _project),
            );
        await container.read(userProjectsProvider(_aliceDid).future);

        expect(
          container.read(userProjectsProvider(_aliceDid)).value!.items,
          isEmpty,
        );
        expect(
          container.read(userProjectsProvider(_aliceDid)).value!.items,
          isEmpty,
        );
        expect(
          container
              .read(userPostsProvider(_aliceDid))
              .value!
              .items
              .map((post) => post.rkey),
          ['a'],
        );
        expect(
          container
              .read(userPostsProvider(_aliceDid))
              .value!
              .items
              .map((post) => post.rkey),
          ['a'],
        );
      },
    );
  });
}
