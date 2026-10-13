// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'business_event_detail_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(BusinessEventDetail)
final businessEventDetailProvider = BusinessEventDetailFamily._();

final class BusinessEventDetailProvider
    extends
        $AsyncNotifierProvider<BusinessEventDetail, BusinessEventDetailState> {
  BusinessEventDetailProvider._({
    required BusinessEventDetailFamily super.from,
    required BusinessEventDetailTarget super.argument,
  }) : super(
         retry: null,
         name: r'businessEventDetailProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$businessEventDetailHash();

  @override
  String toString() {
    return r'businessEventDetailProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  BusinessEventDetail create() => BusinessEventDetail();

  @override
  bool operator ==(Object other) {
    return other is BusinessEventDetailProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$businessEventDetailHash() =>
    r'f923c765c244b0a10ecc8220f248ef0801f97596';

final class BusinessEventDetailFamily extends $Family
    with
        $ClassFamilyOverride<
          BusinessEventDetail,
          AsyncValue<BusinessEventDetailState>,
          BusinessEventDetailState,
          FutureOr<BusinessEventDetailState>,
          BusinessEventDetailTarget
        > {
  BusinessEventDetailFamily._()
    : super(
        retry: null,
        name: r'businessEventDetailProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  BusinessEventDetailProvider call(BusinessEventDetailTarget target) =>
      BusinessEventDetailProvider._(argument: target, from: this);

  @override
  String toString() => r'businessEventDetailProvider';
}

abstract class _$BusinessEventDetail
    extends $AsyncNotifier<BusinessEventDetailState> {
  late final _$args = ref.$arg as BusinessEventDetailTarget;
  BusinessEventDetailTarget get target => _$args;

  FutureOr<BusinessEventDetailState> build(BusinessEventDetailTarget target);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<BusinessEventDetailState>,
              BusinessEventDetailState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<BusinessEventDetailState>,
                BusinessEventDetailState
              >,
              AsyncValue<BusinessEventDetailState>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
