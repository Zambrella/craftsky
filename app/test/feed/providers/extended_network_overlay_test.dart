import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../test_support/post_fixtures.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  test('new publications enter Following but never Extended network', () async {
    final post = postFixture(rkey: 'new-own-post');
    final seedPublication = Provider<void>((ref) {
      final controller = ref.read(pdsRecordOperationControllerProvider);
      final token = controller.begin(
        scope: postRecordMutationScope(ref, post),
        operationKey: 'new-own-post',
        endpoint: '/v1/posts',
        immutableBody: '{}',
      );
      controller.markAccepted(
        token,
        optimisticValue: post,
        agrees: (_) => false,
      );
    });
    final container = ProviderContainer.test(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postRepositoryProvider.overrideWithValue(
          FakePostRepository(
            onListExtendedNetwork: ({cursor, limit}) async =>
                const TimelinePage(items: []),
            onListTimeline: ({cursor, limit}) async =>
                const TimelinePage(items: []),
          ),
        ),
      ],
    )..read(seedPublication);

    expect(
      (await container.read(extendedNetworkProvider.future)).items,
      isEmpty,
    );
    expect(
      (await container.read(timelineProvider.future)).items.single.post.uri,
      post.uri,
    );
  });

  test(
    'post edits retain network membership, attribution and pagination',
    () async {
      final post = postFixture(rkey: 'network-post');
      final connection = PostAuthor(
        did: 'did:plc:bridge',
        handle: 'bridge.example',
      );
      final applyEdit = Provider<void Function()>(
        (ref) => () {
          final controller = ref.read(pdsRecordOperationControllerProvider);
          final token = controller.begin(
            scope: postRecordMutationScope(ref, post),
            operationKey: 'network-edit',
            endpoint: '/v1/posts/owner/network-post',
            immutableBody: '{}',
          );
          controller.markAccepted(
            token,
            optimisticValue: PostEditOverlay(
              post.copyWith(cid: 'bafynetworkedit', text: 'Edited post'),
            ),
            agrees: (_) => false,
          );
          notifyPostRecordOverlays(ref);
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
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListExtendedNetwork: ({cursor, limit}) async => TimelinePage(
                items: [
                  TimelineItem(
                    itemKey: 'network-item',
                    post: post,
                    followedBy: connection,
                  ),
                ],
                cursor: 'page-two',
              ),
            ),
          ),
        ],
      )..listen(extendedNetworkProvider, (_, _) {});
      await container.read(extendedNetworkProvider.future);

      container.read(applyEdit)();
      final state = container.read(extendedNetworkProvider).value!;
      expect(state.items.single.post.text, 'Edited post');
      expect(state.items.single.followedBy?.did, connection.did);
      expect(state.items.single.itemKey, 'network-item');
      expect(state.cursor, 'page-two');
    },
  );
}
