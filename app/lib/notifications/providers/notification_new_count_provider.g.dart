// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'notification_new_count_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

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
    r'5c93dd5b73133810a42cf8d47eb6bc7a85f38aa3';

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
