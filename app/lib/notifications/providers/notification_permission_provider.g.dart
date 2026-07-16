// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_permission_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(notificationPermission)
final notificationPermissionProvider = NotificationPermissionProvider._();

final class NotificationPermissionProvider
    extends
        $FunctionalProvider<
          AsyncValue<NotificationPermission>,
          NotificationPermission,
          FutureOr<NotificationPermission>
        >
    with
        $FutureModifier<NotificationPermission>,
        $FutureProvider<NotificationPermission> {
  NotificationPermissionProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationPermissionProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationPermissionHash();

  @$internal
  @override
  $FutureProviderElement<NotificationPermission> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<NotificationPermission> create(Ref ref) {
    return notificationPermission(ref);
  }
}

String _$notificationPermissionHash() =>
    r'2d15c45354b6f3696fb5efd7d2c1f64e2139ab67';
