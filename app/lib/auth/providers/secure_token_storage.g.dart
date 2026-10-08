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
    r'1305b8c635a3f07ea0dc7d2c23de5bcbe55b5d5d';
