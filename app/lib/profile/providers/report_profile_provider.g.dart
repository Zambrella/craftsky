// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'report_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ReportProfile)
final reportProfileProvider = ReportProfileProvider._();

final class ReportProfileProvider
    extends $AsyncNotifierProvider<ReportProfile, ReportResult?> {
  ReportProfileProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'reportProfileProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$reportProfileHash();

  @$internal
  @override
  ReportProfile create() => ReportProfile();
}

String _$reportProfileHash() => r'12bcc64a5d4c7d3c0c135ba67e4dc0326078e8ca';

abstract class _$ReportProfile extends $AsyncNotifier<ReportResult?> {
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
