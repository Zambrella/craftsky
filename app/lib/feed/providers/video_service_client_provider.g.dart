// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'video_service_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(videoServiceConfiguration)
final videoServiceConfigurationProvider = VideoServiceConfigurationProvider._();

final class VideoServiceConfigurationProvider
    extends
        $FunctionalProvider<
          VideoServiceConfiguration,
          VideoServiceConfiguration,
          VideoServiceConfiguration
        >
    with $Provider<VideoServiceConfiguration> {
  VideoServiceConfigurationProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'videoServiceConfigurationProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$videoServiceConfigurationHash();

  @$internal
  @override
  $ProviderElement<VideoServiceConfiguration> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  VideoServiceConfiguration create(Ref ref) {
    return videoServiceConfiguration(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(VideoServiceConfiguration value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<VideoServiceConfiguration>(value),
    );
  }
}

String _$videoServiceConfigurationHash() =>
    r'09c1f02c2cf00bcff977186358632afc479cf90e';

@ProviderFor(videoServiceClient)
final videoServiceClientProvider = VideoServiceClientProvider._();

final class VideoServiceClientProvider
    extends
        $FunctionalProvider<
          VideoServiceClient,
          VideoServiceClient,
          VideoServiceClient
        >
    with $Provider<VideoServiceClient> {
  VideoServiceClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'videoServiceClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$videoServiceClientHash();

  @$internal
  @override
  $ProviderElement<VideoServiceClient> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  VideoServiceClient create(Ref ref) {
    return videoServiceClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(VideoServiceClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<VideoServiceClient>(value),
    );
  }
}

String _$videoServiceClientHash() =>
    r'50a649b3ccf29cb89c7da5ad042862e5fbee0558';
