// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'toggle_block_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ToggleBlockProfile)
final toggleBlockProfileProvider = ToggleBlockProfileProvider._();

final class ToggleBlockProfileProvider
    extends $AsyncNotifierProvider<ToggleBlockProfile, bool?> {
  ToggleBlockProfileProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'toggleBlockProfileProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$toggleBlockProfileHash();

  @$internal
  @override
  ToggleBlockProfile create() => ToggleBlockProfile();
}

String _$toggleBlockProfileHash() =>
    r'f78d6b0a7988b69a881eab656e31e2013a8f34fa';

abstract class _$ToggleBlockProfile extends $AsyncNotifier<bool?> {
  FutureOr<bool?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<bool?>, bool?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<bool?>, bool?>,
              AsyncValue<bool?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
