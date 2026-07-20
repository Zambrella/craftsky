// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_new_count_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountNotificationNewCount)
final accountNotificationNewCountProvider =
    AccountNotificationNewCountFamily._();

final class AccountNotificationNewCountProvider
    extends $AsyncNotifierProvider<AccountNotificationNewCount, int> {
  AccountNotificationNewCountProvider._({
    required AccountNotificationNewCountFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountNotificationNewCountProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountNotificationNewCountHash();

  @override
  String toString() {
    return r'accountNotificationNewCountProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  AccountNotificationNewCount create() => AccountNotificationNewCount();

  @override
  bool operator ==(Object other) {
    return other is AccountNotificationNewCountProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountNotificationNewCountHash() =>
    r'd6a71bdf36c58bea0957358b882b8511a04bd9a6';

final class AccountNotificationNewCountFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountNotificationNewCount,
          AsyncValue<int>,
          int,
          FutureOr<int>,
          AccountKey
        > {
  AccountNotificationNewCountFamily._()
    : super(
        retry: null,
        name: r'accountNotificationNewCountProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  AccountNotificationNewCountProvider call(AccountKey account) =>
      AccountNotificationNewCountProvider._(argument: account, from: this);

  @override
  String toString() => r'accountNotificationNewCountProvider';
}

abstract class _$AccountNotificationNewCount extends $AsyncNotifier<int> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<int> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<int>, int>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<int>, int>,
              AsyncValue<int>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

@ProviderFor(NotificationNewCount)
final notificationNewCountProvider = NotificationNewCountProvider._();

final class NotificationNewCountProvider
    extends $AsyncNotifierProvider<NotificationNewCount, int> {
  NotificationNewCountProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationNewCountProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationNewCountHash();

  @$internal
  @override
  NotificationNewCount create() => NotificationNewCount();
}

String _$notificationNewCountHash() =>
    r'351a23ce08d86b49ea97647d6b43d0fb442ef968';

abstract class _$NotificationNewCount extends $AsyncNotifier<int> {
  FutureOr<int> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<int>, int>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<int>, int>,
              AsyncValue<int>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
