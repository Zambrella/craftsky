// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'image_cache_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(profileImageCacheManager)
final profileImageCacheManagerProvider = ProfileImageCacheManagerProvider._();

final class ProfileImageCacheManagerProvider
    extends
        $FunctionalProvider<
          BaseCacheManager,
          BaseCacheManager,
          BaseCacheManager
        >
    with $Provider<BaseCacheManager> {
  ProfileImageCacheManagerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'profileImageCacheManagerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$profileImageCacheManagerHash();

  @$internal
  @override
  $ProviderElement<BaseCacheManager> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  BaseCacheManager create(Ref ref) {
    return profileImageCacheManager(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(BaseCacheManager value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<BaseCacheManager>(value),
    );
  }
}

String _$profileImageCacheManagerHash() =>
    r'94a853b38c46449dbb9237f8705a8371bd568944';

@ProviderFor(feedImageCacheManager)
final feedImageCacheManagerProvider = FeedImageCacheManagerProvider._();

final class FeedImageCacheManagerProvider
    extends
        $FunctionalProvider<
          BaseCacheManager,
          BaseCacheManager,
          BaseCacheManager
        >
    with $Provider<BaseCacheManager> {
  FeedImageCacheManagerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'feedImageCacheManagerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$feedImageCacheManagerHash();

  @$internal
  @override
  $ProviderElement<BaseCacheManager> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  BaseCacheManager create(Ref ref) {
    return feedImageCacheManager(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(BaseCacheManager value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<BaseCacheManager>(value),
    );
  }
}

String _$feedImageCacheManagerHash() =>
    r'226252145e15f22830639423d046dfb5b08772a3';
