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

String _$reportPostHash() => r'80141658589634a8dcf14e598fc1a0b1a901620c';

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
