// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'app_dependencies.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(appDependencies)
final appDependenciesProvider = AppDependenciesProvider._();

final class AppDependenciesProvider
    extends
        $FunctionalProvider<
          AsyncValue<AppDependencies>,
          AppDependencies,
          FutureOr<AppDependencies>
        >
    with $FutureModifier<AppDependencies>, $FutureProvider<AppDependencies> {
  AppDependenciesProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'appDependenciesProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$appDependenciesHash();

  @$internal
  @override
  $FutureProviderElement<AppDependencies> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<AppDependencies> create(Ref ref) {
    return appDependencies(ref);
  }
}

String _$appDependenciesHash() => r'fb0ab3446fb744480fff9e191a8f5f78a5fc380d';

@ProviderFor(sharedPreferences)
final sharedPreferencesProvider = SharedPreferencesProvider._();

final class SharedPreferencesProvider
    extends
        $FunctionalProvider<
          SharedPreferences,
          SharedPreferences,
          SharedPreferences
        >
    with $Provider<SharedPreferences> {
  SharedPreferencesProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'sharedPreferencesProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sharedPreferencesHash();

  @$internal
  @override
  $ProviderElement<SharedPreferences> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  SharedPreferences create(Ref ref) {
    return sharedPreferences(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SharedPreferences value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SharedPreferences>(value),
    );
  }
}

String _$sharedPreferencesHash() => r'e78a5cc39f313e59e0a492b7f1ad06226c61c73e';

@ProviderFor(packageInfo)
final packageInfoProvider = PackageInfoProvider._();

final class PackageInfoProvider
    extends $FunctionalProvider<PackageInfo, PackageInfo, PackageInfo>
    with $Provider<PackageInfo> {
  PackageInfoProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'packageInfoProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$packageInfoHash();

  @$internal
  @override
  $ProviderElement<PackageInfo> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  PackageInfo create(Ref ref) {
    return packageInfo(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(PackageInfo value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<PackageInfo>(value),
    );
  }
}

String _$packageInfoHash() => r'e7be0723cbc2345aefdaac1ec66d5dfbaed9dd9a';

@ProviderFor(deviceInfo)
final deviceInfoProvider = DeviceInfoProvider._();

final class DeviceInfoProvider
    extends
        $FunctionalProvider<
          CraftskyDeviceInfo,
          CraftskyDeviceInfo,
          CraftskyDeviceInfo
        >
    with $Provider<CraftskyDeviceInfo> {
  DeviceInfoProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deviceInfoProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deviceInfoHash();

  @$internal
  @override
  $ProviderElement<CraftskyDeviceInfo> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  CraftskyDeviceInfo create(Ref ref) {
    return deviceInfo(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(CraftskyDeviceInfo value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<CraftskyDeviceInfo>(value),
    );
  }
}

String _$deviceInfoHash() => r'9a7f4d8e3b92769d076298f3d39c51fdec5e8c03';

@ProviderFor(appVersion)
final appVersionProvider = AppVersionProvider._();

final class AppVersionProvider
    extends $FunctionalProvider<Version, Version, Version>
    with $Provider<Version> {
  AppVersionProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'appVersionProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$appVersionHash();

  @$internal
  @override
  $ProviderElement<Version> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  Version create(Ref ref) {
    return appVersion(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Version value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Version>(value),
    );
  }
}

String _$appVersionHash() => r'0d9fc092c80524b14247a7f1c1f7ba2736a6708a';
