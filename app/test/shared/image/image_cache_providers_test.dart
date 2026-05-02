import 'package:craftsky_app/shared/image/image_cache_managers.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/services.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:plugin_platform_interface/plugin_platform_interface.dart';
import 'package:sqflite/sqflite.dart';

class _FakePathProvider
    with MockPlatformInterfaceMixin
    implements PathProviderPlatform {
  @override
  Future<String?> getTemporaryPath() async => '/tmp';

  @override
  Future<String?> getApplicationDocumentsPath() async => '/tmp';

  @override
  Future<String?> getApplicationSupportPath() async => '/tmp';

  @override
  Future<String?> getApplicationCachePath() async => '/tmp';

  @override
  Future<String?> getLibraryPath() async => '/tmp';

  @override
  Future<String?> getExternalStoragePath() async => '/tmp';

  @override
  Future<List<String>?> getExternalCachePaths() async => ['/tmp'];

  @override
  Future<List<String>?> getExternalStoragePaths({
    StorageDirectory? type,
  }) async => ['/tmp'];

  @override
  Future<String?> getDownloadsPath() async => '/tmp';
}

void main() {
  setUpAll(() {
    TestWidgetsFlutterBinding.ensureInitialized();
    PathProviderPlatform.instance = _FakePathProvider();

    // Register sqflite with a stub platform channel so CacheManager's
    // background DB open doesn't error after tests complete.
    SqflitePlugin.registerWith();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
          const MethodChannel('com.tekartik.sqflite'),
          (MethodCall call) async {
            return switch (call.method) {
              'getDatabasesPath' => '/tmp',
              'openDatabase' => <String, dynamic>{'id': 1},
              'execute' => null,
              'query' => <dynamic>[],
              'insert' => 1,
              _ => null,
            };
          },
        );
  });

  group('profileImageCacheManagerProvider', () {
    test('returns a ProfileImageCacheManager instance', () {
      final container = ProviderContainer.test();
      final manager = container.read(profileImageCacheManagerProvider);

      expect(manager, isA<ProfileImageCacheManager>());
      expect(manager, isA<BaseCacheManager>());
    });

    test('returns the same singleton on repeat reads', () {
      final container = ProviderContainer.test();
      final first = container.read(profileImageCacheManagerProvider);
      final second = container.read(profileImageCacheManagerProvider);

      expect(identical(first, second), isTrue);
    });
  });

  group('feedImageCacheManagerProvider', () {
    test('returns a FeedImageCacheManager instance', () {
      final container = ProviderContainer.test();
      final manager = container.read(feedImageCacheManagerProvider);

      expect(manager, isA<FeedImageCacheManager>());
      expect(manager, isA<BaseCacheManager>());
    });

    test('returns the same singleton on repeat reads', () {
      final container = ProviderContainer.test();
      final first = container.read(feedImageCacheManagerProvider);
      final second = container.read(feedImageCacheManagerProvider);

      expect(identical(first, second), isTrue);
    });
  });
}
