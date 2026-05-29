// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'timeline_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Cursor-accumulating authenticated home timeline provider.

@ProviderFor(Timeline)
final timelineProvider = TimelineProvider._();

/// Cursor-accumulating authenticated home timeline provider.
final class TimelineProvider
    extends $AsyncNotifierProvider<Timeline, TimelineState> {
  /// Cursor-accumulating authenticated home timeline provider.
  TimelineProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'timelineProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$timelineHash();

  @$internal
  @override
  Timeline create() => Timeline();
}

String _$timelineHash() => r'9bedb37719986b3c7742eb79839005e0a01663a7';

/// Cursor-accumulating authenticated home timeline provider.

abstract class _$Timeline extends $AsyncNotifier<TimelineState> {
  FutureOr<TimelineState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<TimelineState>, TimelineState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<TimelineState>, TimelineState>,
              AsyncValue<TimelineState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
