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
  });
}
