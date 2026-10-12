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

String _$reportProfileHash() => r'602a338f21ae448d97283e508c8fdb1f387b40a7';

abstract class _$ReportProfile extends $AsyncNotifier<ReportResult?> {
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
