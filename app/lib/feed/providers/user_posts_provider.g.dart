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
    required (Did, {BrowsingContext? audience}) super.argument,
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
        '$argument';
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

String _$userPostsHash() => r'361b0fa18a700c6ee92811234d4aea1407c55837';

/// Cursor-accumulating list-by-author provider, keyed by DID.

final class UserPostsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserPosts,
          AsyncValue<UserPostsState>,
          UserPostsState,
          FutureOr<UserPostsState>,
          (Did, {BrowsingContext? audience})
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

  UserPostsProvider call(Did did, {BrowsingContext? audience}) =>
      UserPostsProvider._(argument: (did, audience: audience), from: this);

  @override
  String toString() => r'userPostsProvider';
}

/// Cursor-accumulating list-by-author provider, keyed by DID.

abstract class _$UserPosts extends $AsyncNotifier<UserPostsState> {
  late final _$args = ref.$arg as (Did, {BrowsingContext? audience});
  Did get did => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<UserPostsState> build(Did did, {BrowsingContext? audience});
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<UserPostsState>, UserPostsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<UserPostsState>, UserPostsState>,
              AsyncValue<UserPostsState>,
              Object?,
              Object?
            >;
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
