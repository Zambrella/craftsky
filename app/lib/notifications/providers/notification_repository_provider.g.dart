// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(notificationApiRepository)
final notificationApiRepositoryProvider = NotificationApiRepositoryProvider._();

final class NotificationApiRepositoryProvider
    extends
        $FunctionalProvider<
          ApiNotificationRepository,
          ApiNotificationRepository,
          ApiNotificationRepository
        >
    with $Provider<ApiNotificationRepository> {
  NotificationApiRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationApiRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationApiRepositoryHash();

  @$internal
  @override
  $ProviderElement<ApiNotificationRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  ApiNotificationRepository create(Ref ref) {
    return notificationApiRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ApiNotificationRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ApiNotificationRepository>(value),
    );
  }
}

String _$notificationApiRepositoryHash() =>
    r'710a47bab9b98d16f0bb8ec0a4c6b73eb44ad57c';

@ProviderFor(notificationRepository)
final notificationRepositoryProvider = NotificationRepositoryProvider._();

final class NotificationRepositoryProvider
    extends
        $FunctionalProvider<
          NotificationRepository,
          NotificationRepository,
          NotificationRepository
        >
    with $Provider<NotificationRepository> {
  NotificationRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationRepositoryHash();

  @$internal
  @override
  $ProviderElement<NotificationRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationRepository create(Ref ref) {
    return notificationRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationRepository>(value),
    );
  }
}

String _$notificationRepositoryHash() =>
    r'b7a86543aaeb67853462d59f1fef54bce16141dc';

@ProviderFor(notificationNewnessRepository)
final notificationNewnessRepositoryProvider =
    NotificationNewnessRepositoryProvider._();

final class NotificationNewnessRepositoryProvider
    extends
        $FunctionalProvider<
          NotificationNewnessRepository,
          NotificationNewnessRepository,
          NotificationNewnessRepository
        >
    with $Provider<NotificationNewnessRepository> {
  NotificationNewnessRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationNewnessRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationNewnessRepositoryHash();

  @$internal
  @override
  $ProviderElement<NotificationNewnessRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationNewnessRepository create(Ref ref) {
    return notificationNewnessRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationNewnessRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationNewnessRepository>(
        value,
      ),
    );
  }
}

String _$notificationNewnessRepositoryHash() =>
    r'b9036c8bdbdcc03f55459d72042dc5ccb7225499';

@ProviderFor(notificationPreferencesRepository)
final notificationPreferencesRepositoryProvider =
    NotificationPreferencesRepositoryProvider._();

final class NotificationPreferencesRepositoryProvider
    extends
        $FunctionalProvider<
          NotificationPreferencesRepository,
          NotificationPreferencesRepository,
          NotificationPreferencesRepository
        >
    with $Provider<NotificationPreferencesRepository> {
  NotificationPreferencesRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationPreferencesRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() =>
      _$notificationPreferencesRepositoryHash();

  @$internal
  @override
  $ProviderElement<NotificationPreferencesRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationPreferencesRepository create(Ref ref) {
    return notificationPreferencesRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationPreferencesRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationPreferencesRepository>(
        value,
      ),
    );
  }
}

String _$notificationPreferencesRepositoryHash() =>
    r'ea65783312b2f5be9c3d9f72da3324ff4c6615c8';

@ProviderFor(notificationDeviceRepository)
final notificationDeviceRepositoryProvider =
    NotificationDeviceRepositoryProvider._();

final class NotificationDeviceRepositoryProvider
    extends
        $FunctionalProvider<
          NotificationDeviceRepository,
          NotificationDeviceRepository,
          NotificationDeviceRepository
        >
    with $Provider<NotificationDeviceRepository> {
  NotificationDeviceRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationDeviceRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationDeviceRepositoryHash();

  @$internal
  @override
  $ProviderElement<NotificationDeviceRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationDeviceRepository create(Ref ref) {
    return notificationDeviceRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationDeviceRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationDeviceRepository>(value),
    );
  }
}

String _$notificationDeviceRepositoryHash() =>
    r'1f16c77684102f003a003ab5eb8f1cfbb3c08112';
