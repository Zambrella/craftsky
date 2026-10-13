// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(postApiClient)
final postApiClientProvider = PostApiClientProvider._();

final class PostApiClientProvider
    extends $FunctionalProvider<PostApiClient, PostApiClient, PostApiClient>
    with $Provider<PostApiClient> {
  PostApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'postApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$postApiClientHash();

  @$internal
  @override
  $ProviderElement<PostApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  PostApiClient create(Ref ref) {
    return postApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PostApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PostApiClient>(value),
    );
  }
}

String _$postApiClientHash() => r'26c40c5db3daa8186b28a397ae29546d36f27b2c';
