// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_comments_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.

@ProviderFor(UserComments)
final userCommentsProvider = UserCommentsFamily._();

/// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.
final class UserCommentsProvider
    extends $AsyncNotifierProvider<UserComments, UserPostsState> {
  /// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.
  UserCommentsProvider._({
    required UserCommentsFamily super.from,
    required String super.argument,
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
        '($argument)';
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

String _$userCommentsHash() => r'46bd50bf2c6a1a80cf7cf44881676b0db5a82420';

/// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.

final class UserCommentsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserComments,
          AsyncValue<UserPostsState>,
          UserPostsState,
          FutureOr<UserPostsState>,
          String
        > {
  UserCommentsFamily._()
    : super(
        retry: null,
        name: r'userCommentsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.

  UserCommentsProvider call(String handleOrDid) =>
      UserCommentsProvider._(argument: handleOrDid, from: this);

  @override
  String toString() => r'userCommentsProvider';
}

/// Cursor-accumulating authored comments/replies list, keyed by `handleOrDid`.

abstract class _$UserComments extends $AsyncNotifier<UserPostsState> {
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
