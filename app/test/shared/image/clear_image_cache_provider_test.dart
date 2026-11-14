import 'dart:async';

import 'package:craftsky_app/shared/image/clear_image_cache_provider.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

void main() {
  group('ClearImageCache', () {
    test('starts in an idle (data) state', () {
      final container = ProviderContainer.test();
      expect(container.read(clearImageCacheProvider), isA<AsyncData<void>>());
    });

    test('calls emptyCache() on both cache managers', () async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager();
      final container = ProviderContainer.test(
        overrides: [
          profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
          feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
        ],
      );

      await container.read(clearImageCacheProvider.notifier).clear();

      expect(profileFake.emptyCacheCalls, 1);
      expect(feedFake.emptyCacheCalls, 1);
      expect(container.read(clearImageCacheProvider), isA<AsyncData<void>>());
    });

    test('reports AsyncError when the profile cache fails', () async {
      final profileFake = FakeBaseCacheManager()
        ..throwOnEmptyCache = StateError('boom');
      final feedFake = FakeBaseCacheManager();
      final container = ProviderContainer.test(
        overrides: [
          profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
          feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
        ],
      );

      await container.read(clearImageCacheProvider.notifier).clear();

      expect(container.read(clearImageCacheProvider), isA<AsyncError<void>>());
    });

    test('reports AsyncError when the feed cache fails', () async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager()
        ..throwOnEmptyCache = StateError('boom');
      final container = ProviderContainer.test(
        overrides: [
          profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
          feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
        ],
      );

      await container.read(clearImageCacheProvider.notifier).clear();

      expect(container.read(clearImageCacheProvider), isA<AsyncError<void>>());
    });

    for (final rebuild in [false, true]) {
      test(
        'pending clear discards failure after '
        '${rebuild ? 'rebuild' : 'disposal'}',
        () async {
          final profile = _PendingCacheManager();
          final container = ProviderContainer.test(
            overrides: [
              profileImageCacheManagerProvider.overrideWith((ref) => profile),
              feedImageCacheManagerProvider.overrideWith(
                (ref) => FakeBaseCacheManager(),
              ),
            ],
          );
          final subscription = container.listen(
            clearImageCacheProvider,
            (_, _) {},
          );
          addTearDown(subscription.close);
          final clearing = container
              .read(clearImageCacheProvider.notifier)
              .clear();
          final completed = expectLater(clearing, completes);

          if (rebuild) {
            container.invalidate(clearImageCacheProvider);
            await container.read(clearImageCacheProvider.future);
          } else {
            subscription.close();
            await container.pump();
            expect(container.exists(clearImageCacheProvider), isFalse);
          }
          profile.pending.completeError(StateError('old cache failure'));
          await completed;

          if (rebuild) {
            expect(
              container.read(clearImageCacheProvider),
              isA<AsyncData<void>>(),
            );
          }
        },
      );
    }
  });
}

class _PendingCacheManager extends FakeBaseCacheManager {
  final pending = Completer<void>();

  @override
  Future<void> emptyCache() => pending.future;
}
