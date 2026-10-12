// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'owner_business_events_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(OwnerBusinessEvents)
final ownerBusinessEventsProvider = OwnerBusinessEventsFamily._();

final class OwnerBusinessEventsProvider
    extends
        $AsyncNotifierProvider<OwnerBusinessEvents, BusinessEventListState> {
  OwnerBusinessEventsProvider._({
    required OwnerBusinessEventsFamily super.from,
    required OwnerEventFilter super.argument,
  }) : super(
         retry: null,
         name: r'ownerBusinessEventsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$ownerBusinessEventsHash();

  @override
  String toString() {
    return r'ownerBusinessEventsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  OwnerBusinessEvents create() => OwnerBusinessEvents();

  @override
  bool operator ==(Object other) {
    return other is OwnerBusinessEventsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$ownerBusinessEventsHash() =>
    r'23eb8f6f1202f00e9751cafab0a9625df26a6027';

final class OwnerBusinessEventsFamily extends $Family
    with
        $ClassFamilyOverride<
          OwnerBusinessEvents,
          AsyncValue<BusinessEventListState>,
          BusinessEventListState,
          FutureOr<BusinessEventListState>,
          OwnerEventFilter
        > {
  OwnerBusinessEventsFamily._()
    : super(
        retry: null,
        name: r'ownerBusinessEventsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  OwnerBusinessEventsProvider call(OwnerEventFilter filter) =>
      OwnerBusinessEventsProvider._(argument: filter, from: this);

  @override
  String toString() => r'ownerBusinessEventsProvider';
}

abstract class _$OwnerBusinessEvents
    extends $AsyncNotifier<BusinessEventListState> {
  late final _$args = ref.$arg as OwnerEventFilter;
  OwnerEventFilter get filter => _$args;

  FutureOr<BusinessEventListState> build(OwnerEventFilter filter);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<BusinessEventListState>, BusinessEventListState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<BusinessEventListState>,
                BusinessEventListState
              >,
              AsyncValue<BusinessEventListState>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}
