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
    required (ProfileBusinessEventsTarget, {BrowsingContext? audience})
    super.argument,
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
        '$argument';
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
    r'a495e1dbf077107cb426d6a66212196fb6f4a26b';

final class ProfileBusinessEventsFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileBusinessEvents,
          AsyncValue<BusinessEventListState>,
          BusinessEventListState,
          FutureOr<BusinessEventListState>,
          (ProfileBusinessEventsTarget, {BrowsingContext? audience})
        > {
  ProfileBusinessEventsFamily._()
    : super(
        retry: null,
        name: r'profileBusinessEventsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileBusinessEventsProvider call(
    ProfileBusinessEventsTarget target, {
    BrowsingContext? audience,
  }) => ProfileBusinessEventsProvider._(
    argument: (target, audience: audience),
    from: this,
  );

  @override
  String toString() => r'profileBusinessEventsProvider';
}

abstract class _$ProfileBusinessEvents
    extends $AsyncNotifier<BusinessEventListState> {
  late final _$args =
      ref.$arg as (ProfileBusinessEventsTarget, {BrowsingContext? audience});
  ProfileBusinessEventsTarget get target => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<BusinessEventListState> build(
    ProfileBusinessEventsTarget target, {
    BrowsingContext? audience,
  });
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
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
