// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'blob_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(blobApiClient)
final blobApiClientProvider = BlobApiClientProvider._();

final class BlobApiClientProvider
    extends $FunctionalProvider<BlobApiClient, BlobApiClient, BlobApiClient>
    with $Provider<BlobApiClient> {
  BlobApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'blobApiClientProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$blobApiClientHash();

  @$internal
  @override
  $ProviderElement<BlobApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  BlobApiClient create(Ref ref) {
    return blobApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(BlobApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<BlobApiClient>(value),
    );
  }
}

String _$blobApiClientHash() => r'08d89a2c5a96dfe773a19d78314c7d197094a562';
