// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_runtime_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(notificationEffectStream)
final notificationEffectStreamProvider = NotificationEffectStreamProvider._();

final class NotificationEffectStreamProvider
    extends
        $FunctionalProvider<
          Raw<Stream<NotificationEffect>>,
          Raw<Stream<NotificationEffect>>,
          Raw<Stream<NotificationEffect>>
        >
    with $Provider<Raw<Stream<NotificationEffect>>> {
  NotificationEffectStreamProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationEffectStreamProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationEffectStreamHash();

  @$internal
  @override
  $ProviderElement<Raw<Stream<NotificationEffect>>> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  Raw<Stream<NotificationEffect>> create(Ref ref) {
    return notificationEffectStream(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(Raw<Stream<NotificationEffect>> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<Raw<Stream<NotificationEffect>>>(
        value,
      ),
    );
  }
}

String _$notificationEffectStreamHash() =>
    r'a383a20ed2145b23dacb5850534c332710a74dd5';

@ProviderFor(_notificationEffectController)
final _notificationEffectControllerProvider =
    _NotificationEffectControllerProvider._();

final class _NotificationEffectControllerProvider
    extends
        $FunctionalProvider<
          StreamController<NotificationEffect>,
          StreamController<NotificationEffect>,
          StreamController<NotificationEffect>
        >
    with $Provider<StreamController<NotificationEffect>> {
  _NotificationEffectControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'_notificationEffectControllerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$_notificationEffectControllerHash();

  @$internal
  @override
  $ProviderElement<StreamController<NotificationEffect>> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  StreamController<NotificationEffect> create(Ref ref) {
    return _notificationEffectController(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(StreamController<NotificationEffect> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride:
          $SyncValueProvider<StreamController<NotificationEffect>>(value),
    );
  }
}

String _$_notificationEffectControllerHash() =>
    r'7279119876ccedb19c3d75368b930e3874df1904';

@ProviderFor(notificationRuntime)
final notificationRuntimeProvider = NotificationRuntimeProvider._();

final class NotificationRuntimeProvider
    extends
        $FunctionalProvider<
          NotificationRuntime,
          NotificationRuntime,
          NotificationRuntime
        >
    with $Provider<NotificationRuntime> {
  NotificationRuntimeProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'notificationRuntimeProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$notificationRuntimeHash();

  @$internal
  @override
  $ProviderElement<NotificationRuntime> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  NotificationRuntime create(Ref ref) {
    return notificationRuntime(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(NotificationRuntime value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<NotificationRuntime>(value),
    );
  }
}

String _$notificationRuntimeHash() =>
    r'9e70a32db0ccc58cb736c702a8120e5967fc49be';
