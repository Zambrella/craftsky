// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Single source of truth for a user's profile, keyed by DID.
///
/// Accepted writes are composed through the shared record overlay while the
/// AppView projection catches up; this provider remains read-only.

@ProviderFor(UserProfile)
final userProfileProvider = UserProfileFamily._();

/// Single source of truth for a user's profile, keyed by DID.
///
/// Accepted writes are composed through the shared record overlay while the
/// AppView projection catches up; this provider remains read-only.
final class UserProfileProvider
    extends $AsyncNotifierProvider<UserProfile, Profile> {
  /// Single source of truth for a user's profile, keyed by DID.
  ///
  /// Accepted writes are composed through the shared record overlay while the
  /// AppView projection catches up; this provider remains read-only.
  UserProfileProvider._({
    required UserProfileFamily super.from,
    required (Did, {BrowsingContext? audience}) super.argument,
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
        '$argument';
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

String _$userProfileHash() => r'8f0f46e552971ddc6a738590076a7201bec56920';

/// Single source of truth for a user's profile, keyed by DID.
///
/// Accepted writes are composed through the shared record overlay while the
/// AppView projection catches up; this provider remains read-only.

final class UserProfileFamily extends $Family
    with
        $ClassFamilyOverride<
          UserProfile,
          AsyncValue<Profile>,
          Profile,
          FutureOr<Profile>,
          (Did, {BrowsingContext? audience})
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
  /// Accepted writes are composed through the shared record overlay while the
  /// AppView projection catches up; this provider remains read-only.

  UserProfileProvider call(Did did, {BrowsingContext? audience}) =>
      UserProfileProvider._(argument: (did, audience: audience), from: this);

  @override
  String toString() => r'userProfileProvider';
}

/// Single source of truth for a user's profile, keyed by DID.
///
/// Accepted writes are composed through the shared record overlay while the
/// AppView projection catches up; this provider remains read-only.

abstract class _$UserProfile extends $AsyncNotifier<Profile> {
  late final _$args = ref.$arg as (Did, {BrowsingContext? audience});
  Did get did => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<Profile> build(Did did, {BrowsingContext? audience});
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Profile>, Profile>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Profile>, Profile>,
              AsyncValue<Profile>,
              Object?,
              Object?
            >;
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
