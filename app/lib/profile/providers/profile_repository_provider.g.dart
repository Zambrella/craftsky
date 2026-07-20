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
