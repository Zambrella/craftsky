// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_relationship_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Account-owned relationship overlay for one target.
///
/// Successful mutations remain authoritative over stale AppView reads until
/// Tap exposes the same policy state. A different account gets a different
/// provider instance and therefore cannot see or complete this overlay.

@ProviderFor(ProfileRelationshipController)
final profileRelationshipControllerProvider =
    ProfileRelationshipControllerFamily._();

/// Account-owned relationship overlay for one target.
///
/// Successful mutations remain authoritative over stale AppView reads until
/// Tap exposes the same policy state. A different account gets a different
/// provider instance and therefore cannot see or complete this overlay.
final class ProfileRelationshipControllerProvider
    extends
        $NotifierProvider<ProfileRelationshipController, ProfileRelationship> {
  /// Account-owned relationship overlay for one target.
  ///
  /// Successful mutations remain authoritative over stale AppView reads until
  /// Tap exposes the same policy state. A different account gets a different
  /// provider instance and therefore cannot see or complete this overlay.
  ProfileRelationshipControllerProvider._({
    required ProfileRelationshipControllerFamily super.from,
    required (AccountKey, String) super.argument,
  }) : super(
         retry: null,
         name: r'profileRelationshipControllerProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileRelationshipControllerHash();

  @override
  String toString() {
    return r'profileRelationshipControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  ProfileRelationshipController create() => ProfileRelationshipController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ProfileRelationship value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ProfileRelationship>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is ProfileRelationshipControllerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileRelationshipControllerHash() =>
    r'dbd08ba6b0f5c627c03ec9bbfd46515f58a76165';

/// Account-owned relationship overlay for one target.
///
/// Successful mutations remain authoritative over stale AppView reads until
/// Tap exposes the same policy state. A different account gets a different
/// provider instance and therefore cannot see or complete this overlay.

final class ProfileRelationshipControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileRelationshipController,
          ProfileRelationship,
          ProfileRelationship,
          ProfileRelationship,
          (AccountKey, String)
        > {
  ProfileRelationshipControllerFamily._()
    : super(
        retry: null,
        name: r'profileRelationshipControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  /// Account-owned relationship overlay for one target.
  ///
  /// Successful mutations remain authoritative over stale AppView reads until
  /// Tap exposes the same policy state. A different account gets a different
  /// provider instance and therefore cannot see or complete this overlay.

  ProfileRelationshipControllerProvider call(
    AccountKey account,
    String handleOrDid,
  ) => ProfileRelationshipControllerProvider._(
    argument: (account, handleOrDid),
    from: this,
  );

  @override
  String toString() => r'profileRelationshipControllerProvider';
}

/// Account-owned relationship overlay for one target.
///
/// Successful mutations remain authoritative over stale AppView reads until
/// Tap exposes the same policy state. A different account gets a different
/// provider instance and therefore cannot see or complete this overlay.

abstract class _$ProfileRelationshipController
    extends $Notifier<ProfileRelationship> {
  late final _$args = ref.$arg as (AccountKey, String);
  AccountKey get account => _$args.$1;
  String get handleOrDid => _$args.$2;

  ProfileRelationship build(AccountKey account, String handleOrDid);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<ProfileRelationship, ProfileRelationship>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<ProfileRelationship, ProfileRelationship>,
              ProfileRelationship,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args.$1, _$args.$2));
  }
}
