// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'secure_token_storage.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(secureSessionRegistryStorage)
final secureSessionRegistryStorageProvider =
    SecureSessionRegistryStorageProvider._();

final class SecureSessionRegistryStorageProvider
    extends
        $FunctionalProvider<
          SessionRegistryStorage,
          SessionRegistryStorage,
          SessionRegistryStorage
        >
    with $Provider<SessionRegistryStorage> {
  SecureSessionRegistryStorageProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'secureSessionRegistryStorageProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$secureSessionRegistryStorageHash();

  @$internal
  @override
  $ProviderElement<SessionRegistryStorage> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SessionRegistryStorage create(Ref ref) {
    return secureSessionRegistryStorage(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SessionRegistryStorage value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SessionRegistryStorage>(value),
    );
  }
}

String _$secureSessionRegistryStorageHash() =>
    r'167cb5c3e8eec87468de062182c52dd891c6f319';

@ProviderFor(secureTokenStorage)
final secureTokenStorageProvider = SecureTokenStorageProvider._();

final class SecureTokenStorageProvider
    extends
        $FunctionalProvider<
          SecureTokenStorage,
          SecureTokenStorage,
          SecureTokenStorage
        >
    with $Provider<SecureTokenStorage> {
  SecureTokenStorageProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'secureTokenStorageProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$secureTokenStorageHash();

  @$internal
  @override
  $ProviderElement<SecureTokenStorage> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SecureTokenStorage create(Ref ref) {
    return secureTokenStorage(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SecureTokenStorage value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SecureTokenStorage>(value),
    );
  }
}

String _$secureTokenStorageHash() =>
    r'7b870de02678b1310d7982ffe1fe6970db192fa2';
