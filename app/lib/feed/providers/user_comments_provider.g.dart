// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_comments_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Cursor-accumulating authored comments/replies list, keyed by DID.

@ProviderFor(UserComments)
final userCommentsProvider = UserCommentsFamily._();

/// Cursor-accumulating authored comments/replies list, keyed by DID.
final class UserCommentsProvider
    extends $AsyncNotifierProvider<UserComments, UserPostsState> {
  /// Cursor-accumulating authored comments/replies list, keyed by DID.
  UserCommentsProvider._({
    required UserCommentsFamily super.from,
    required (Did, {BrowsingContext? audience}) super.argument,
  }) : super(
         retry: null,
         name: r'userCommentsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$userCommentsHash();

  @override
  String toString() {
    return r'userCommentsProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  UserComments create() => UserComments();

  @override
  bool operator ==(Object other) {
    return other is UserCommentsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$userCommentsHash() => r'9d15ec8429125c3c831f805eb7268e9ae113aeee';

/// Cursor-accumulating authored comments/replies list, keyed by DID.

final class UserCommentsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserComments,
          AsyncValue<UserPostsState>,
          UserPostsState,
          FutureOr<UserPostsState>,
          (Did, {BrowsingContext? audience})
        > {
  UserCommentsFamily._()
    : super(
        retry: null,
        name: r'userCommentsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Cursor-accumulating authored comments/replies list, keyed by DID.

  UserCommentsProvider call(Did did, {BrowsingContext? audience}) =>
      UserCommentsProvider._(argument: (did, audience: audience), from: this);

  @override
  String toString() => r'userCommentsProvider';
}

/// Cursor-accumulating authored comments/replies list, keyed by DID.

abstract class _$UserComments extends $AsyncNotifier<UserPostsState> {
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
