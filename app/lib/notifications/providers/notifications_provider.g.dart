// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notifications_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountNotifications)
final accountNotificationsProvider = AccountNotificationsFamily._();

final class AccountNotificationsProvider
    extends $AsyncNotifierProvider<AccountNotifications, NotificationsState> {
  AccountNotificationsProvider._({
    required AccountNotificationsFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountNotificationsProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountNotificationsHash();

  @override
  String toString() {
    return r'accountNotificationsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  AccountNotifications create() => AccountNotifications();

  @override
  bool operator ==(Object other) {
    return other is AccountNotificationsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountNotificationsHash() =>
    r'2f026da14098c9af6c06fd91446e5fda3e7e3cd1';

final class AccountNotificationsFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountNotifications,
          AsyncValue<NotificationsState>,
          NotificationsState,
          FutureOr<NotificationsState>,
          AccountKey
        > {
  AccountNotificationsFamily._()
    : super(
        retry: null,
        name: r'accountNotificationsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  AccountNotificationsProvider call(AccountKey account) =>
      AccountNotificationsProvider._(argument: account, from: this);

  @override
  String toString() => r'accountNotificationsProvider';
}

abstract class _$AccountNotifications
    extends $AsyncNotifier<NotificationsState> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<NotificationsState> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<NotificationsState>, NotificationsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<NotificationsState>, NotificationsState>,
              AsyncValue<NotificationsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

@ProviderFor(Notifications)
final notificationsProvider = NotificationsProvider._();

final class NotificationsProvider
    extends $AsyncNotifierProvider<Notifications, NotificationsState> {
  NotificationsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationsProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationsHash();

  @$internal
  @override
  Notifications create() => Notifications();
}

String _$notificationsHash() => r'924434df127dc12191f3a79b8d2ff05ba8507ac7';

abstract class _$Notifications extends $AsyncNotifier<NotificationsState> {
  FutureOr<NotificationsState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<NotificationsState>, NotificationsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<NotificationsState>, NotificationsState>,
              AsyncValue<NotificationsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
