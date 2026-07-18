// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'toggle_follow_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ToggleFollowProfile)
final toggleFollowProfileProvider = ToggleFollowProfileProvider._();

final class ToggleFollowProfileProvider
    extends $AsyncNotifierProvider<ToggleFollowProfile, Profile?> {
  ToggleFollowProfileProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'toggleFollowProfileProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$toggleFollowProfileHash();

  @$internal
  @override
  ToggleFollowProfile create() => ToggleFollowProfile();
}

String _$toggleFollowProfileHash() =>
    r'59c4c3e64ea591e2225e0f823fe42d3f26f5b871';

abstract class _$ToggleFollowProfile extends $AsyncNotifier<Profile?> {
  FutureOr<Profile?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Profile?>, Profile?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Profile?>, Profile?>,
              AsyncValue<Profile?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
