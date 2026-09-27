// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'delete_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Standalone delete-a-post mutation notifier. The caller supplies the [Post]
/// so the command can retain its exact URI and expected CID.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).

@ProviderFor(DeletePost)
final deletePostProvider = DeletePostProvider._();

/// Standalone delete-a-post mutation notifier. The caller supplies the [Post]
/// so the command can retain its exact URI and expected CID.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
final class DeletePostProvider
    extends $AsyncNotifierProvider<DeletePost, Post?> {
  /// Standalone delete-a-post mutation notifier. The caller supplies the [Post]
  /// so the command can retain its exact URI and expected CID.
  ///
  /// `build()` returns `Post?` so the `AsyncData(post)` transition
  /// carries the deleted post for `ref.listen` consumers (e.g. an
  /// "undo delete" snackbar).
  DeletePostProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deletePostProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deletePostHash();

  @$internal
  @override
  DeletePost create() => DeletePost();
}

String _$deletePostHash() => r'b168e6c52c661675e3c8d8f7a6588991f15d14c7';

/// Standalone delete-a-post mutation notifier. The caller supplies the [Post]
/// so the command can retain its exact URI and expected CID.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).

abstract class _$DeletePost extends $AsyncNotifier<Post?> {
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
