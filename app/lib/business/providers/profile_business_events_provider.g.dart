// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_business_events_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileBusinessEvents)
final profileBusinessEventsProvider = ProfileBusinessEventsFamily._();

final class ProfileBusinessEventsProvider
    extends
        $AsyncNotifierProvider<ProfileBusinessEvents, BusinessEventListState> {
  ProfileBusinessEventsProvider._({
    required ProfileBusinessEventsFamily super.from,
    required ProfileBusinessEventsTarget super.argument,
  }) : super(
         retry: null,
         name: r'profileBusinessEventsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileBusinessEventsHash();

  @override
  String toString() {
    return r'profileBusinessEventsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfileBusinessEvents create() => ProfileBusinessEvents();

  @override
  bool operator ==(Object other) {
    return other is ProfileBusinessEventsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileBusinessEventsHash() =>
    r'd51267fa1632acf9fe8d5d28bbc5e0b35610759f';

final class ProfileBusinessEventsFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileBusinessEvents,
          AsyncValue<BusinessEventListState>,
          BusinessEventListState,
          FutureOr<BusinessEventListState>,
          ProfileBusinessEventsTarget
        > {
  ProfileBusinessEventsFamily._()
    : super(
        retry: null,
        name: r'profileBusinessEventsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileBusinessEventsProvider call(ProfileBusinessEventsTarget target) =>
      ProfileBusinessEventsProvider._(argument: target, from: this);

  @override
  String toString() => r'profileBusinessEventsProvider';
}

abstract class _$ProfileBusinessEvents
    extends $AsyncNotifier<BusinessEventListState> {
  late final _$args = ref.$arg as ProfileBusinessEventsTarget;
  ProfileBusinessEventsTarget get target => _$args;

  FutureOr<BusinessEventListState> build(ProfileBusinessEventsTarget target);
  @$mustCallSuper
  @override
  void runBuild() {
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
    element.handleCreate(ref, () => build(_$args));
  }
}
