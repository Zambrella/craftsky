// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'report_business_event_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ReportBusinessEvent)
final reportBusinessEventProvider = ReportBusinessEventFamily._();

final class ReportBusinessEventProvider
    extends $AsyncNotifierProvider<ReportBusinessEvent, ReportResult?> {
  ReportBusinessEventProvider._({
    required ReportBusinessEventFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'reportBusinessEventProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$reportBusinessEventHash();

  @override
  String toString() {
    return r'reportBusinessEventProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ReportBusinessEvent create() => ReportBusinessEvent();

  @override
  bool operator ==(Object other) {
    return other is ReportBusinessEventProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$reportBusinessEventHash() =>
    r'30ad8bcb6339d80542e8695615b1d3590f999f98';

final class ReportBusinessEventFamily extends $Family
    with
        $ClassFamilyOverride<
          ReportBusinessEvent,
          AsyncValue<ReportResult?>,
          ReportResult?,
          FutureOr<ReportResult?>,
          AccountKey
        > {
  ReportBusinessEventFamily._()
    : super(
        retry: null,
        name: r'reportBusinessEventProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ReportBusinessEventProvider call(AccountKey account) =>
      ReportBusinessEventProvider._(argument: account, from: this);

  @override
  String toString() => r'reportBusinessEventProvider';
}

abstract class _$ReportBusinessEvent extends $AsyncNotifier<ReportResult?> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<ReportResult?> build(AccountKey account);
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
    element.handleCreate(ref, () => build(_$args));
  }
}
