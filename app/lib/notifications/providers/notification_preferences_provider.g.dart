// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_preferences_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

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
