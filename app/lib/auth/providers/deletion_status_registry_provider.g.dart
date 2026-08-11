// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'deletion_status_registry_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(deletionStatusRegistryStorage)
final deletionStatusRegistryStorageProvider =
    DeletionStatusRegistryStorageProvider._();

final class DeletionStatusRegistryStorageProvider
    extends
        $FunctionalProvider<
          DeletionStatusRegistryStorage,
          DeletionStatusRegistryStorage,
          DeletionStatusRegistryStorage
        >
    with $Provider<DeletionStatusRegistryStorage> {
  DeletionStatusRegistryStorageProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deletionStatusRegistryStorageProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deletionStatusRegistryStorageHash();

  @$internal
  @override
  $ProviderElement<DeletionStatusRegistryStorage> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  DeletionStatusRegistryStorage create(Ref ref) {
    return deletionStatusRegistryStorage(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(DeletionStatusRegistryStorage value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<DeletionStatusRegistryStorage>(
        value,
      ),
    );
  }
}

String _$deletionStatusRegistryStorageHash() =>
    r'cc4663670830509587653f45ecb27c0d1cb60dda';

@ProviderFor(DeletionStatusRegistry)
final deletionStatusRegistryProvider = DeletionStatusRegistryProvider._();

final class DeletionStatusRegistryProvider
    extends
        $AsyncNotifierProvider<
          DeletionStatusRegistry,
          model.DeletionStatusRegistry
        > {
  DeletionStatusRegistryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deletionStatusRegistryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deletionStatusRegistryHash();

  @$internal
  @override
  DeletionStatusRegistry create() => DeletionStatusRegistry();
}

String _$deletionStatusRegistryHash() =>
    r'4b2f2933766e0b0a5b6f8e5c5bb606ae7460095f';

abstract class _$DeletionStatusRegistry
    extends $AsyncNotifier<model.DeletionStatusRegistry> {
  FutureOr<model.DeletionStatusRegistry> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<model.DeletionStatusRegistry>,
              model.DeletionStatusRegistry
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<model.DeletionStatusRegistry>,
                model.DeletionStatusRegistry
              >,
              AsyncValue<model.DeletionStatusRegistry>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
