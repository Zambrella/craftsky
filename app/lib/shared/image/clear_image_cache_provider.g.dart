// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'clear_image_cache_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Mutation that empties both image caches in parallel. Idle by default;
/// transitions through AsyncLoading on each invocation of [clear].

@ProviderFor(ClearImageCache)
final clearImageCacheProvider = ClearImageCacheProvider._();

/// Mutation that empties both image caches in parallel. Idle by default;
/// transitions through AsyncLoading on each invocation of [clear].
final class ClearImageCacheProvider
    extends $AsyncNotifierProvider<ClearImageCache, void> {
  /// Mutation that empties both image caches in parallel. Idle by default;
  /// transitions through AsyncLoading on each invocation of [clear].
  ClearImageCacheProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'clearImageCacheProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$clearImageCacheHash();

  @$internal
  @override
  ClearImageCache create() => ClearImageCache();
}

String _$clearImageCacheHash() => r'863ebf0ae8d756812bf426cde3c6282b9528bd3f';

/// Mutation that empties both image caches in parallel. Idle by default;
/// transitions through AsyncLoading on each invocation of [clear].

abstract class _$ClearImageCache extends $AsyncNotifier<void> {
  FutureOr<void> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
