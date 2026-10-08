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
    r'9b80922e80a5ad7c6e9f2f15262923bd5b7783a4';

abstract class _$ToggleBlockProfile extends $AsyncNotifier<bool?> {
  FutureOr<bool?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<bool?>, bool?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<bool?>, bool?>,
              AsyncValue<bool?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
