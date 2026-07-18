// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_lifecycle_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(notificationRoutingStorage)
final notificationRoutingStorageProvider =
    NotificationRoutingStorageProvider._();

final class NotificationRoutingStorageProvider
    extends
        $FunctionalProvider<
          NotificationRoutingStorage,
          NotificationRoutingStorage,
          NotificationRoutingStorage
        >
    with $Provider<NotificationRoutingStorage> {
  NotificationRoutingStorageProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationRoutingStorageProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationRoutingStorageHash();

  @$internal
  @override
  $ProviderElement<NotificationRoutingStorage> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationRoutingStorage create(Ref ref) {
    return notificationRoutingStorage(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationRoutingStorage value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationRoutingStorage>(value),
    );
  }
}

String _$notificationRoutingStorageHash() =>
    r'bbd3146f4b65a2355721c7e08022d4d7b634494d';

@ProviderFor(notificationSignOutCleanup)
final notificationSignOutCleanupProvider =
    NotificationSignOutCleanupProvider._();

final class NotificationSignOutCleanupProvider
    extends
        $FunctionalProvider<
          NotificationSignOutCleanup,
          NotificationSignOutCleanup,
          NotificationSignOutCleanup
        >
    with $Provider<NotificationSignOutCleanup> {
  NotificationSignOutCleanupProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationSignOutCleanupProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationSignOutCleanupHash();

  @$internal
  @override
  $ProviderElement<NotificationSignOutCleanup> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationSignOutCleanup create(Ref ref) {
    return notificationSignOutCleanup(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationSignOutCleanup value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationSignOutCleanup>(value),
    );
  }
}

String _$notificationSignOutCleanupHash() =>
    r'bdd1838cea332c53c4e832fa58356d11652057ba';
