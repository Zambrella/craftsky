// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'delete_post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Standalone delete-a-post mutation notifier. Takes the full [Post]
/// because the cache update needs `did`, `handle`, and `rkey` to splice
/// the post out of any live family entries (lists may be keyed by
/// either form). The caller — UI deleting a post it's already
/// rendering — has the [Post] in hand.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
///
/// On success, removes the post from any live `userPostsProvider`
/// family entries keyed by either the author's handle or DID,
/// sidestepping the AppView's read-after-delete window (where a
/// refetch could still include the just-deleted row until the firehose
/// tombstone arrives).

@ProviderFor(DeletePost)
final deletePostProvider = DeletePostProvider._();

/// Standalone delete-a-post mutation notifier. Takes the full [Post]
/// because the cache update needs `did`, `handle`, and `rkey` to splice
/// the post out of any live family entries (lists may be keyed by
/// either form). The caller — UI deleting a post it's already
/// rendering — has the [Post] in hand.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
///
/// On success, removes the post from any live `userPostsProvider`
/// family entries keyed by either the author's handle or DID,
/// sidestepping the AppView's read-after-delete window (where a
/// refetch could still include the just-deleted row until the firehose
/// tombstone arrives).
final class DeletePostProvider
    extends $AsyncNotifierProvider<DeletePost, Post?> {
  /// Standalone delete-a-post mutation notifier. Takes the full [Post]
  /// because the cache update needs `did`, `handle`, and `rkey` to splice
  /// the post out of any live family entries (lists may be keyed by
  /// either form). The caller — UI deleting a post it's already
  /// rendering — has the [Post] in hand.
  ///
  /// `build()` returns `Post?` so the `AsyncData(post)` transition
  /// carries the deleted post for `ref.listen` consumers (e.g. an
  /// "undo delete" snackbar).
  ///
  /// On success, removes the post from any live `userPostsProvider`
  /// family entries keyed by either the author's handle or DID,
  /// sidestepping the AppView's read-after-delete window (where a
  /// refetch could still include the just-deleted row until the firehose
  /// tombstone arrives).
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

String _$deletePostHash() => r'c700c6f30facba70ec4c3e18b8e48cbd5e8eb200';

/// Standalone delete-a-post mutation notifier. Takes the full [Post]
/// because the cache update needs `did`, `handle`, and `rkey` to splice
/// the post out of any live family entries (lists may be keyed by
/// either form). The caller — UI deleting a post it's already
/// rendering — has the [Post] in hand.
///
/// `build()` returns `Post?` so the `AsyncData(post)` transition
/// carries the deleted post for `ref.listen` consumers (e.g. an
/// "undo delete" snackbar).
///
/// On success, removes the post from any live `userPostsProvider`
/// family entries keyed by either the author's handle or DID,
/// sidestepping the AppView's read-after-delete window (where a
/// refetch could still include the just-deleted row until the firehose
/// tombstone arrives).

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
