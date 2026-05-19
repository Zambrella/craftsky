// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_posts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.

@ProviderFor(UserPosts)
final userPostsProvider = UserPostsFamily._();

/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.
final class UserPostsProvider
    extends $AsyncNotifierProvider<UserPosts, UserPostsState> {
  /// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.
  UserPostsProvider._({
    required UserPostsFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'userPostsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$userPostsHash();

  @override
  String toString() {
    return r'userPostsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  UserPosts create() => UserPosts();

  @override
  bool operator ==(Object other) {
    return other is UserPostsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$userPostsHash() => r'fcee975de4d938d5232c38da72d025e176bd1dcf';

/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.

final class UserPostsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserPosts,
          AsyncValue<UserPostsState>,
          UserPostsState,
          FutureOr<UserPostsState>,
          String
        > {
  UserPostsFamily._()
    : super(
        retry: null,
        name: r'userPostsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.

  UserPostsProvider call(String handleOrDid) =>
      UserPostsProvider._(argument: handleOrDid, from: this);

  @override
  String toString() => r'userPostsProvider';
}

/// Cursor-accumulating list-by-author provider, keyed by `handleOrDid`.

abstract class _$UserPosts extends $AsyncNotifier<UserPostsState> {
  late final _$args = ref.$arg as String;
  String get handleOrDid => _$args;

  FutureOr<UserPostsState> build(String handleOrDid);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<UserPostsState>, UserPostsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<UserPostsState>, UserPostsState>,
              AsyncValue<UserPostsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
