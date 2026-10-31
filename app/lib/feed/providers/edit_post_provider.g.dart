// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'edit_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(EditPost)
final editPostProvider = EditPostProvider._();

final class EditPostProvider
    extends $AsyncNotifierProvider<EditPost, PostEditResult?> {
  EditPostProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'editPostProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$editPostHash();

  @$internal
  @override
  EditPost create() => EditPost();
}

String _$editPostHash() => r'0bd67fbb32a6c3985b2fa0b840a5ea7d6dc26b0f';

abstract class _$EditPost extends $AsyncNotifier<PostEditResult?> {
  FutureOr<PostEditResult?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<PostEditResult?>, PostEditResult?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<PostEditResult?>, PostEditResult?>,
              AsyncValue<PostEditResult?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
