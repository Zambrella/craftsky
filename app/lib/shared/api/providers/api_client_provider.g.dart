// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(craftskyApiClient)
final craftskyApiClientProvider = CraftskyApiClientProvider._();

final class CraftskyApiClientProvider
    extends
        $FunctionalProvider<
          CraftskyApiClient,
          CraftskyApiClient,
          CraftskyApiClient
        >
    with $Provider<CraftskyApiClient> {
  CraftskyApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'craftskyApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$craftskyApiClientHash();

  @$internal
  @override
  $ProviderElement<CraftskyApiClient> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  CraftskyApiClient create(Ref ref) {
    return craftskyApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(CraftskyApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<CraftskyApiClient>(value),
    );
  }
}

String _$craftskyApiClientHash() => r'0c24d2ad6f99c0784a0bbed6c68b9cd43a8fe8e0';
