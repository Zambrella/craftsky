// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_preferences_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountNotificationPreferencesNotifier)
final accountNotificationPreferencesProvider =
    AccountNotificationPreferencesNotifierFamily._();

final class AccountNotificationPreferencesNotifierProvider
    extends
        $AsyncNotifierProvider<
          AccountNotificationPreferencesNotifier,
          NotificationPreferences
        > {
  AccountNotificationPreferencesNotifierProvider._({
    required AccountNotificationPreferencesNotifierFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountNotificationPreferencesProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() =>
      _$accountNotificationPreferencesNotifierHash();

  @override
  String toString() {
    return r'accountNotificationPreferencesProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  AccountNotificationPreferencesNotifier create() =>
      AccountNotificationPreferencesNotifier();

  @override
  bool operator ==(Object other) {
    return other is AccountNotificationPreferencesNotifierProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountNotificationPreferencesNotifierHash() =>
    r'2bdfdc5d2cbbbf9959b5d9071fc4efb2cf4cee4f';

final class AccountNotificationPreferencesNotifierFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountNotificationPreferencesNotifier,
          AsyncValue<NotificationPreferences>,
          NotificationPreferences,
          FutureOr<NotificationPreferences>,
          AccountKey
        > {
  AccountNotificationPreferencesNotifierFamily._()
    : super(
        retry: null,
        name: r'accountNotificationPreferencesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  AccountNotificationPreferencesNotifierProvider call(AccountKey account) =>
      AccountNotificationPreferencesNotifierProvider._(
        argument: account,
        from: this,
      );

  @override
  String toString() => r'accountNotificationPreferencesProvider';
}

abstract class _$AccountNotificationPreferencesNotifier
    extends $AsyncNotifier<NotificationPreferences> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<NotificationPreferences> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<NotificationPreferences>,
              NotificationPreferences
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<NotificationPreferences>,
                NotificationPreferences
              >,
              AsyncValue<NotificationPreferences>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

@ProviderFor(NotificationPreferencesNotifier)
final notificationPreferencesProvider =
    NotificationPreferencesNotifierProvider._();

final class NotificationPreferencesNotifierProvider
    extends
        $AsyncNotifierProvider<
          NotificationPreferencesNotifier,
          NotificationPreferences
        > {
  NotificationPreferencesNotifierProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationPreferencesProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationPreferencesNotifierHash();

  @$internal
  @override
  NotificationPreferencesNotifier create() => NotificationPreferencesNotifier();
}

String _$notificationPreferencesNotifierHash() =>
    r'c5567087f667340ffe0646aea67d2e6c3a32af3d';

abstract class _$NotificationPreferencesNotifier
    extends $AsyncNotifier<NotificationPreferences> {
  FutureOr<NotificationPreferences> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<NotificationPreferences>,
              NotificationPreferences
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<NotificationPreferences>,
                NotificationPreferences
              >,
              AsyncValue<NotificationPreferences>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
