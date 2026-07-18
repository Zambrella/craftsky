// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_seen_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountNotificationSeen)
final accountNotificationSeenProvider = AccountNotificationSeenFamily._();

final class AccountNotificationSeenProvider
    extends
        $FunctionalProvider<
          AsyncValue<NotificationSeenCoordinator>,
          NotificationSeenCoordinator,
          FutureOr<NotificationSeenCoordinator>
        >
    with
        $FutureModifier<NotificationSeenCoordinator>,
        $FutureProvider<NotificationSeenCoordinator> {
  AccountNotificationSeenProvider._({
    required AccountNotificationSeenFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountNotificationSeenProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountNotificationSeenHash();

  @override
  String toString() {
    return r'accountNotificationSeenProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<NotificationSeenCoordinator> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<NotificationSeenCoordinator> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountNotificationSeen(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountNotificationSeenProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountNotificationSeenHash() =>
    r'33e354c6d7e9c8c5246d67a260fe58d20531d507';

final class AccountNotificationSeenFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<NotificationSeenCoordinator>,
          AccountKey
        > {
  AccountNotificationSeenFamily._()
    : super(
        retry: null,
        name: r'accountNotificationSeenProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  AccountNotificationSeenProvider call(AccountKey account) =>
      AccountNotificationSeenProvider._(argument: account, from: this);

  @override
  String toString() => r'accountNotificationSeenProvider';
}

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
