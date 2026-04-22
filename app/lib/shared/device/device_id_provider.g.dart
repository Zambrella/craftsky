// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'device_id_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Injection seam: production uses a shared `FlutterSecureStorage`
/// instance; tests override with a fake.

@ProviderFor(deviceIdSecureStorage)
final deviceIdSecureStorageProvider = DeviceIdSecureStorageProvider._();

/// Injection seam: production uses a shared `FlutterSecureStorage`
/// instance; tests override with a fake.

final class DeviceIdSecureStorageProvider
    extends
        $FunctionalProvider<
          FlutterSecureStorage,
          FlutterSecureStorage,
          FlutterSecureStorage
        >
    with $Provider<FlutterSecureStorage> {
  /// Injection seam: production uses a shared `FlutterSecureStorage`
  /// instance; tests override with a fake.
  DeviceIdSecureStorageProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deviceIdSecureStorageProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deviceIdSecureStorageHash();

  @$internal
  @override
  $ProviderElement<FlutterSecureStorage> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  FlutterSecureStorage create(Ref ref) {
    return deviceIdSecureStorage(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(FlutterSecureStorage value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<FlutterSecureStorage>(value),
    );
  }
}

String _$deviceIdSecureStorageHash() =>
    r'3e24dc6b6c68c336fd177d1aa6c3a33684195e98';

/// Returns this install's stable device identifier. On first access,
/// generates a v4 UUID and writes it to secure storage. Subsequent
/// accesses return the persisted value.
///
/// Platform-error tolerant: if secure storage fails, we return a fresh
/// in-memory UUID for this session and attempt to persist it. On a
/// persistence failure, future launches may generate a different ID —
/// acceptable because device-id is correlation data, not a security
/// primitive.

@ProviderFor(deviceId)
final deviceIdProvider = DeviceIdProvider._();

/// Returns this install's stable device identifier. On first access,
/// generates a v4 UUID and writes it to secure storage. Subsequent
/// accesses return the persisted value.
///
/// Platform-error tolerant: if secure storage fails, we return a fresh
/// in-memory UUID for this session and attempt to persist it. On a
/// persistence failure, future launches may generate a different ID —
/// acceptable because device-id is correlation data, not a security
/// primitive.

final class DeviceIdProvider
    extends $FunctionalProvider<AsyncValue<String>, String, FutureOr<String>>
    with $FutureModifier<String>, $FutureProvider<String> {
  /// Returns this install's stable device identifier. On first access,
  /// generates a v4 UUID and writes it to secure storage. Subsequent
  /// accesses return the persisted value.
  ///
  /// Platform-error tolerant: if secure storage fails, we return a fresh
  /// in-memory UUID for this session and attempt to persist it. On a
  /// persistence failure, future launches may generate a different ID —
  /// acceptable because device-id is correlation data, not a security
  /// primitive.
  DeviceIdProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deviceIdProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deviceIdHash();

  @$internal
  @override
  $FutureProviderElement<String> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<String> create(Ref ref) {
    return deviceId(ref);
  }
}

String _$deviceIdHash() => r'93e03fc85c705b8be8b5a586f2e0a8d5271022f3';
