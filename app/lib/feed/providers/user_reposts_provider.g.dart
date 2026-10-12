// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_reposts_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Cursor-accumulating active repost subjects, keyed by the reposter DID.

@ProviderFor(UserReposts)
final userRepostsProvider = UserRepostsFamily._();

/// Cursor-accumulating active repost subjects, keyed by the reposter DID.
final class UserRepostsProvider
    extends $AsyncNotifierProvider<UserReposts, UserPostsState> {
  /// Cursor-accumulating active repost subjects, keyed by the reposter DID.
  UserRepostsProvider._({
    required UserRepostsFamily super.from,
    required Did super.argument,
  }) : super(
         retry: null,
         name: r'userRepostsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$userRepostsHash();

  @override
  String toString() {
    return r'userRepostsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  UserReposts create() => UserReposts();

  @override
  bool operator ==(Object other) {
    return other is UserRepostsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$userRepostsHash() => r'c69c8087f73e185bc76ad4d85f03dcb2b14aab3c';

/// Cursor-accumulating active repost subjects, keyed by the reposter DID.

final class UserRepostsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserReposts,
          AsyncValue<UserPostsState>,
          UserPostsState,
          FutureOr<UserPostsState>,
          Did
        > {
  UserRepostsFamily._()
    : super(
        retry: null,
        name: r'userRepostsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Cursor-accumulating active repost subjects, keyed by the reposter DID.

  UserRepostsProvider call(Did did) =>
      UserRepostsProvider._(argument: did, from: this);

  @override
  String toString() => r'userRepostsProvider';
}

/// Cursor-accumulating active repost subjects, keyed by the reposter DID.

abstract class _$UserReposts extends $AsyncNotifier<UserPostsState> {
  late final _$args = ref.$arg as Did;
  Did get did => _$args;

  FutureOr<UserPostsState> build(Did did);
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
    return element.handleCreate(ref, () => build(_$args));
  }
}
