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

String _$reportPostHash() => r'036dca5a1beb1c862b7c1b2c3cee793cf8611f04';

abstract class _$ReportPost extends $AsyncNotifier<ReportResult?> {
  FutureOr<ReportResult?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<ReportResult?>, ReportResult?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ReportResult?>, ReportResult?>,
              AsyncValue<ReportResult?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
