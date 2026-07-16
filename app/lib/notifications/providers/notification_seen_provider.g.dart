// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_seen_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(notificationSeen)
final notificationSeenProvider = NotificationSeenProvider._();

final class NotificationSeenProvider
    extends
        $FunctionalProvider<
          NotificationSeenCoordinator,
          NotificationSeenCoordinator,
          NotificationSeenCoordinator
        >
    with $Provider<NotificationSeenCoordinator> {
  NotificationSeenProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationSeenProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationSeenHash();

  @$internal
  @override
  $ProviderElement<NotificationSeenCoordinator> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationSeenCoordinator create(Ref ref) {
    return notificationSeen(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationSeenCoordinator value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationSeenCoordinator>(value),
    );
  }
}

String _$notificationSeenHash() => r'8c369605100e2b46d62687f155e8d4eba5e666e8';
