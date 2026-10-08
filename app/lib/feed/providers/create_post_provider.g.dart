// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.

@ProviderFor(CreatePost)
final createPostProvider = CreatePostProvider._();

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.
final class CreatePostProvider
    extends $AsyncNotifierProvider<CreatePost, Post?> {
  /// Standalone create-a-post mutation notifier. Idle until [create] runs,
  /// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
  /// `AsyncError` on failure.
  ///
  /// Callers should bind via `ref.listen(createPostProvider, ...)` and
  /// call [reset] after consuming a transition so a re-entry to the
  /// compose page doesn't see the previous result.
  CreatePostProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'createPostProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$createPostHash();

  @$internal
  @override
  CreatePost create() => CreatePost();
}

String _$createPostHash() => r'6891144aa078e6e4f37c5d30eab173a235e0c415';

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.

abstract class _$CreatePost extends $AsyncNotifier<Post?> {
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
