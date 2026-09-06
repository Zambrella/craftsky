// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_posts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Cursor-accumulating list-by-author provider, keyed by DID.

@ProviderFor(UserPosts)
final userPostsProvider = UserPostsFamily._();

/// Cursor-accumulating list-by-author provider, keyed by DID.
final class UserPostsProvider
    extends $AsyncNotifierProvider<UserPosts, UserPostsState> {
  /// Cursor-accumulating list-by-author provider, keyed by DID.
  UserPostsProvider._({
    required UserPostsFamily super.from,
    required Did super.argument,
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

String _$userPostsHash() => r'6584df0973f108f1fae2bc8f8b68a4707267317e';

/// Cursor-accumulating list-by-author provider, keyed by DID.

final class UserPostsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserPosts,
          AsyncValue<UserPostsState>,
          UserPostsState,
          FutureOr<UserPostsState>,
          Did
        > {
  UserPostsFamily._()
    : super(
        retry: null,
        name: r'userPostsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Cursor-accumulating list-by-author provider, keyed by DID.

  UserPostsProvider call(Did did) =>
      UserPostsProvider._(argument: did, from: this);

  @override
  String toString() => r'userPostsProvider';
}

/// Cursor-accumulating list-by-author provider, keyed by DID.

abstract class _$UserPosts extends $AsyncNotifier<UserPostsState> {
  late final _$args = ref.$arg as Did;
  Did get did => _$args;

  FutureOr<UserPostsState> build(Did did);
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
