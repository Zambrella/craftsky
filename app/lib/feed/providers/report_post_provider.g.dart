// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'report_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ReportPost)
final reportPostProvider = ReportPostProvider._();

final class ReportPostProvider
    extends $AsyncNotifierProvider<ReportPost, ReportResult?> {
  ReportPostProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'reportPostProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$reportPostHash();

  @$internal
  @override
  ReportPost create() => ReportPost();
}

String _$reportPostHash() => r'ba50e0ae1673f5a0e6148704816994d65f2d86fa';

abstract class _$ReportPost extends $AsyncNotifier<ReportResult?> {
  FutureOr<ReportResult?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<ReportResult?>, ReportResult?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ReportResult?>, ReportResult?>,
              AsyncValue<ReportResult?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
