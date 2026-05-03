import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'clear_image_cache_provider.g.dart';

/// Mutation that empties both image caches in parallel. Idle by default;
/// transitions through AsyncLoading on each invocation of [clear].
@riverpod
class ClearImageCache extends _$ClearImageCache {
  @override
  FutureOr<void> build() => null;

  Future<void> clear() async {
    final profileCache = ref.read(profileImageCacheManagerProvider);
    final feedCache = ref.read(feedImageCacheManagerProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      await Future.wait(<Future<void>>[
        profileCache.emptyCache(),
        feedCache.emptyCache(),
      ]);
    });
  }
}
