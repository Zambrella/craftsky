// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'saved_posts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SavedPosts)
final savedPostsProvider = SavedPostsFamily._();

final class SavedPostsProvider
    extends $AsyncNotifierProvider<SavedPosts, SavedPostListState> {
  SavedPostsProvider._({
    required SavedPostsFamily super.from,
    required SavedPostListKey super.argument,
  }) : super(
         retry: null,
         name: r'savedPostsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$savedPostsHash();

  @override
  String toString() {
    return r'savedPostsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  SavedPosts create() => SavedPosts();

  @override
  bool operator ==(Object other) {
    return other is SavedPostsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$savedPostsHash() => r'6620720139f0758e501d492bbd424443ed0c5171';

final class SavedPostsFamily extends $Family
    with
        $ClassFamilyOverride<
          SavedPosts,
          AsyncValue<SavedPostListState>,
          SavedPostListState,
          FutureOr<SavedPostListState>,
          SavedPostListKey
        > {
  SavedPostsFamily._()
    : super(
        retry: null,
        name: r'savedPostsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SavedPostsProvider call(SavedPostListKey key) =>
      SavedPostsProvider._(argument: key, from: this);

  @override
  String toString() => r'savedPostsProvider';
}

abstract class _$SavedPosts extends $AsyncNotifier<SavedPostListState> {
  late final _$args = ref.$arg as SavedPostListKey;
  SavedPostListKey get key => _$args;

  FutureOr<SavedPostListState> build(SavedPostListKey key);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<SavedPostListState>, SavedPostListState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<SavedPostListState>, SavedPostListState>,
              AsyncValue<SavedPostListState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
