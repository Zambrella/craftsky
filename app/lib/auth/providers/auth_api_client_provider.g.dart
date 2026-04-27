// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(authApiClient)
final authApiClientProvider = AuthApiClientProvider._();

final class AuthApiClientProvider
    extends $FunctionalProvider<AuthApiClient, AuthApiClient, AuthApiClient>
    with $Provider<AuthApiClient> {
  AuthApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authApiClientHash();

  @$internal
  @override
  $ProviderElement<AuthApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  AuthApiClient create(Ref ref) {
    return authApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AuthApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AuthApiClient>(value),
    );
  }
}

String _$authApiClientHash() => r'4d4b1c0d7c2a203cc99d9cd90df7d7d6e5c95410';
