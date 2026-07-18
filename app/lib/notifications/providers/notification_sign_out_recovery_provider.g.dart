// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_sign_out_recovery_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(notificationSignOutRecovery)
final notificationSignOutRecoveryProvider =
    NotificationSignOutRecoveryProvider._();

final class NotificationSignOutRecoveryProvider
    extends
        $FunctionalProvider<
          NotificationSignOutRecovery,
          NotificationSignOutRecovery,
          NotificationSignOutRecovery
        >
    with $Provider<NotificationSignOutRecovery> {
  NotificationSignOutRecoveryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationSignOutRecoveryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationSignOutRecoveryHash();

  @$internal
  @override
  $ProviderElement<NotificationSignOutRecovery> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationSignOutRecovery create(Ref ref) {
    return notificationSignOutRecovery(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationSignOutRecovery value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationSignOutRecovery>(value),
    );
  }
}

String _$notificationSignOutRecoveryHash() =>
    r'4411ae38fce8c57257936baf94a6315f366c68d1';
