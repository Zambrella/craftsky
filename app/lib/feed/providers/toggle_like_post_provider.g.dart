// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'toggle_like_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ToggleLikePost)
final toggleLikePostProvider = ToggleLikePostProvider._();

final class ToggleLikePostProvider
    extends $AsyncNotifierProvider<ToggleLikePost, Post?> {
  ToggleLikePostProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'toggleLikePostProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$toggleLikePostHash();

  @$internal
  @override
  ToggleLikePost create() => ToggleLikePost();
}

String _$toggleLikePostHash() => r'e9f1831c9a4b0285787a2fb120d9c8374a41e363';

abstract class _$ToggleLikePost extends $AsyncNotifier<Post?> {
  FutureOr<Post?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Post?>, Post?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Post?>, Post?>,
              AsyncValue<Post?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
