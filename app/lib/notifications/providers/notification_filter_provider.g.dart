// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_filter_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// In-memory selection survives navigation, but resets when the owner changes.

@ProviderFor(NotificationFilter)
final notificationFilterProvider = NotificationFilterProvider._();

/// In-memory selection survives navigation, but resets when the owner changes.
final class NotificationFilterProvider
    extends $NotifierProvider<NotificationFilter, NotificationCategory?> {
  /// In-memory selection survives navigation, but resets when the owner changes.
  NotificationFilterProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationFilterProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationFilterHash();

  @$internal
  @override
  NotificationFilter create() => NotificationFilter();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationCategory? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationCategory?>(value),
    );
  }
}

String _$notificationFilterHash() =>
    r'c3345d144166a61eb3be5c44a79b437ce00682b4';

/// In-memory selection survives navigation, but resets when the owner changes.

abstract class _$NotificationFilter extends $Notifier<NotificationCategory?> {
  NotificationCategory? build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<NotificationCategory?, NotificationCategory?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<NotificationCategory?, NotificationCategory?>,
              NotificationCategory?,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
