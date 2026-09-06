// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Single source of truth for a user's profile, keyed by DID.
///
/// Holds both read and write logic so mutation methods can perform
/// optimistic updates against the cached `AsyncData` and roll back on
/// failure. Mutations only succeed against the authenticated user's
/// profile — the AppView rejects writes against any other DID — so
/// callers should only invoke them on the family entry that matches
/// the signed-in user.
///

@ProviderFor(UserProfile)
final userProfileProvider = UserProfileFamily._();

/// Single source of truth for a user's profile, keyed by DID.
///
/// Holds both read and write logic so mutation methods can perform
/// optimistic updates against the cached `AsyncData` and roll back on
/// failure. Mutations only succeed against the authenticated user's
/// profile — the AppView rejects writes against any other DID — so
/// callers should only invoke them on the family entry that matches
/// the signed-in user.
///
final class UserProfileProvider
    extends $AsyncNotifierProvider<UserProfile, Profile> {
  /// Single source of truth for a user's profile, keyed by DID.
  ///
  /// Holds both read and write logic so mutation methods can perform
  /// optimistic updates against the cached `AsyncData` and roll back on
  /// failure. Mutations only succeed against the authenticated user's
  /// profile — the AppView rejects writes against any other DID — so
  /// callers should only invoke them on the family entry that matches
  /// the signed-in user.
  ///
  UserProfileProvider._({
    required UserProfileFamily super.from,
    required Did super.argument,
  }) : super(
         retry: null,
         name: r'userProfileProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$userProfileHash();

  @override
  String toString() {
    return r'userProfileProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  UserProfile create() => UserProfile();

  @override
  bool operator ==(Object other) {
    return other is UserProfileProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$userProfileHash() => r'c26afb60f507054e1f9b14b63d8f4a15ba804c80';

/// Single source of truth for a user's profile, keyed by DID.
///
/// Holds both read and write logic so mutation methods can perform
/// optimistic updates against the cached `AsyncData` and roll back on
/// failure. Mutations only succeed against the authenticated user's
/// profile — the AppView rejects writes against any other DID — so
/// callers should only invoke them on the family entry that matches
/// the signed-in user.
///

final class UserProfileFamily extends $Family
    with
        $ClassFamilyOverride<
          UserProfile,
          AsyncValue<Profile>,
          Profile,
          FutureOr<Profile>,
          Did
        > {
  UserProfileFamily._()
    : super(
        retry: null,
        name: r'userProfileProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Single source of truth for a user's profile, keyed by DID.
  ///
  /// Holds both read and write logic so mutation methods can perform
  /// optimistic updates against the cached `AsyncData` and roll back on
  /// failure. Mutations only succeed against the authenticated user's
  /// profile — the AppView rejects writes against any other DID — so
  /// callers should only invoke them on the family entry that matches
  /// the signed-in user.
  ///

  UserProfileProvider call(Did did) =>
      UserProfileProvider._(argument: did, from: this);

  @override
  String toString() => r'userProfileProvider';
}

/// Single source of truth for a user's profile, keyed by DID.
///
/// Holds both read and write logic so mutation methods can perform
/// optimistic updates against the cached `AsyncData` and roll back on
/// failure. Mutations only succeed against the authenticated user's
/// profile — the AppView rejects writes against any other DID — so
/// callers should only invoke them on the family entry that matches
/// the signed-in user.
///

abstract class _$UserProfile extends $AsyncNotifier<Profile> {
  late final _$args = ref.$arg as Did;
  Did get did => _$args;

  FutureOr<Profile> build(Did did);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Profile>, Profile>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Profile>, Profile>,
              AsyncValue<Profile>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
