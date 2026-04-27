// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(profileApiClient)
final profileApiClientProvider = ProfileApiClientProvider._();

final class ProfileApiClientProvider
    extends
        $FunctionalProvider<
          ProfileApiClient,
          ProfileApiClient,
          ProfileApiClient
        >
    with $Provider<ProfileApiClient> {
  ProfileApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'profileApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$profileApiClientHash();

  @$internal
  @override
  $ProviderElement<ProfileApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ProfileApiClient create(Ref ref) {
    return profileApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ProfileApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ProfileApiClient>(value),
    );
  }
}

String _$profileApiClientHash() => r'8be8303ff48ec74107e61cb46cf2c12d919515ac';
