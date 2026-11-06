import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../feed/fakes/fake_post_repository.dart';
import '../../test_support/post_fixtures.dart';

class _Registry extends SessionRegistry {
  @override
  Future<model.SessionRegistry> build() async =>
      model.SessionRegistry.empty().upsertAndActivate(
        token: 'test',
        did: 'did:plc:viewer',
        handle: 'viewer.test',
      );
  void guest() => state = AsyncData(model.SessionRegistry.empty());
}

void main() {
  setUpAll(initializeMappers);
  test(
    'UT-005 member cached rows disappear before an anonymous reload completes',
    () async {
      final publicResult = Completer<PostPage>();
      var calls = 0;
      final container = ProviderContainer.test(
        overrides: [
          sessionRegistryProvider.overrideWith(_Registry.new),
          activeContentLanguagePolicyProvider.overrideWithBuild(
            (ref, notifier) => const [],
          ),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListByAuthor: (id, {cursor, limit}) async {
                if (++calls == 1) {
                  return PostPage(items: [postFixture(text: 'MEMBER_CANARY')]);
                }
                return await publicResult.future;
              },
            ),
          ),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      final provider = Provider<AsyncValue<UserPostsState>>(
        (ref) => ref.watch(
          userPostsProvider(
            Did.parse('did:plc:alice'),
            audience: ref.watch(browsingContextProvider),
          ),
        ),
      );
      final sub = container.listen(provider, (_, _) {});
      addTearDown(sub.close);
      expect(
        (await container.read(
          userPostsProvider(
            Did.parse('did:plc:alice'),
            audience: container.read(browsingContextProvider),
          ).future,
        )).items.single.text,
        'MEMBER_CANARY',
      );
      (container.read(sessionRegistryProvider.notifier) as _Registry).guest();
      await container.pump();
      expect(
        sub.read().value?.items.any((post) => post.text == 'MEMBER_CANARY') ??
            false,
        false,
      );
      publicResult.complete(const PostPage(items: []));
      await container.pump();
    },
  );
}
