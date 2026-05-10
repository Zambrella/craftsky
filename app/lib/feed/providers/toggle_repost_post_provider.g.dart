// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'toggle_repost_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ToggleRepostPost)
final toggleRepostPostProvider = ToggleRepostPostProvider._();

final class ToggleRepostPostProvider
    extends $AsyncNotifierProvider<ToggleRepostPost, Post?> {
  ToggleRepostPostProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'toggleRepostPostProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$toggleRepostPostHash();

  @$internal
  @override
  ToggleRepostPost create() => ToggleRepostPost();
}

String _$toggleRepostPostHash() => r'ddc5a948ad2171380d3f4bf5fa9ca7f2df5f0c25';

abstract class _$ToggleRepostPost extends $AsyncNotifier<Post?> {
  FutureOr<Post?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Post?>, Post?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Post?>, Post?>,
              AsyncValue<Post?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
