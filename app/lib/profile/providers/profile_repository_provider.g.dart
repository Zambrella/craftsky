// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(profileRepository)
final profileRepositoryProvider = ProfileRepositoryProvider._();

final class ProfileRepositoryProvider
    extends
        $FunctionalProvider<
          ProfileRepository,
          ProfileRepository,
          ProfileRepository
        >
    with $Provider<ProfileRepository> {
  ProfileRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'profileRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$profileRepositoryHash();

  @$internal
  @override
  $ProviderElement<ProfileRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  ProfileRepository create(Ref ref) {
    return profileRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ProfileRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ProfileRepository>(value),
    );
  }
}

String _$profileRepositoryHash() => r'f3023fe6a10025f168a18ff29fd374e7cd79527f';

@ProviderFor(accountRelationshipRepository)
final accountRelationshipRepositoryProvider =
    AccountRelationshipRepositoryFamily._();

final class AccountRelationshipRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ProfileRepository>,
          ProfileRepository,
          FutureOr<ProfileRepository>
        >
    with
        $FutureModifier<ProfileRepository>,
        $FutureProvider<ProfileRepository> {
  AccountRelationshipRepositoryProvider._({
    required AccountRelationshipRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountRelationshipRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountRelationshipRepositoryHash();

  @override
  String toString() {
    return r'accountRelationshipRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ProfileRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ProfileRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountRelationshipRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountRelationshipRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountRelationshipRepositoryHash() =>
    r'b889f53a866a81ef83498571e291f875c2481020';

final class AccountRelationshipRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ProfileRepository>, AccountKey> {
  AccountRelationshipRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountRelationshipRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountRelationshipRepositoryProvider call(AccountKey account) =>
      AccountRelationshipRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountRelationshipRepositoryProvider';
}

@ProviderFor(accountFollowerGrowthRepository)
final accountFollowerGrowthRepositoryProvider =
    AccountFollowerGrowthRepositoryFamily._();

final class AccountFollowerGrowthRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ProfileRepository>,
          ProfileRepository,
          FutureOr<ProfileRepository>
        >
    with
        $FutureModifier<ProfileRepository>,
        $FutureProvider<ProfileRepository> {
  AccountFollowerGrowthRepositoryProvider._({
    required AccountFollowerGrowthRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountFollowerGrowthRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountFollowerGrowthRepositoryHash();

  @override
  String toString() {
    return r'accountFollowerGrowthRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ProfileRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ProfileRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountFollowerGrowthRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountFollowerGrowthRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountFollowerGrowthRepositoryHash() =>
    r'9fe4ea45e4fd9040d169c144d716279393b2dd9d';

final class AccountFollowerGrowthRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ProfileRepository>, AccountKey> {
  AccountFollowerGrowthRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountFollowerGrowthRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountFollowerGrowthRepositoryProvider call(AccountKey account) =>
      AccountFollowerGrowthRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountFollowerGrowthRepositoryProvider';
}

@ProviderFor(accountProfileRepository)
final accountProfileRepositoryProvider = AccountProfileRepositoryFamily._();

final class AccountProfileRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ProfileRepository>,
          ProfileRepository,
          FutureOr<ProfileRepository>
        >
    with
        $FutureModifier<ProfileRepository>,
        $FutureProvider<ProfileRepository> {
  AccountProfileRepositoryProvider._({
    required AccountProfileRepositoryFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'accountProfileRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountProfileRepositoryHash();

  @override
  String toString() {
    return r'accountProfileRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ProfileRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ProfileRepository> create(Ref ref) {
    final argument = this.argument as ActiveAccountLease;
    return accountProfileRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountProfileRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountProfileRepositoryHash() =>
    r'ad0d45b414e8a8f23308a7e12a7b62380f200574';

final class AccountProfileRepositoryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ProfileRepository>,
          ActiveAccountLease
        > {
  AccountProfileRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountProfileRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountProfileRepositoryProvider call(ActiveAccountLease lease) =>
      AccountProfileRepositoryProvider._(argument: lease, from: this);

  @override
  String toString() => r'accountProfileRepositoryProvider';
}
