// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'search_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(searchApiClient)
final searchApiClientProvider = SearchApiClientProvider._();

final class SearchApiClientProvider
    extends
        $FunctionalProvider<SearchApiClient, SearchApiClient, SearchApiClient>
    with $Provider<SearchApiClient> {
  SearchApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'searchApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$searchApiClientHash();

  @$internal
  @override
  $ProviderElement<SearchApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  SearchApiClient create(Ref ref) {
    return searchApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SearchApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SearchApiClient>(value),
    );
  }
}

String _$searchApiClientHash() => r'a16a756d6e610fc5a6f2b35657ccefe76f59bdd8';
