// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'account_eligibility_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountEligibilityRepository)
final accountEligibilityRepositoryProvider =
    AccountEligibilityRepositoryFamily._();

final class AccountEligibilityRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<AccountEligibilityRepository>,
          AccountEligibilityRepository,
          FutureOr<AccountEligibilityRepository>
        >
    with
        $FutureModifier<AccountEligibilityRepository>,
        $FutureProvider<AccountEligibilityRepository> {
  AccountEligibilityRepositoryProvider._({
    required AccountEligibilityRepositoryFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'accountEligibilityRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountEligibilityRepositoryHash();

  @override
  String toString() {
    return r'accountEligibilityRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<AccountEligibilityRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<AccountEligibilityRepository> create(Ref ref) {
    final argument = this.argument as ActiveAccountLease;
    return accountEligibilityRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountEligibilityRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountEligibilityRepositoryHash() =>
    r'6131b56f6e33d5212fa252032232eccbd26ef53f';

final class AccountEligibilityRepositoryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<AccountEligibilityRepository>,
          ActiveAccountLease
        > {
  AccountEligibilityRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountEligibilityRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountEligibilityRepositoryProvider call(ActiveAccountLease lease) =>
      AccountEligibilityRepositoryProvider._(argument: lease, from: this);

  @override
  String toString() => r'accountEligibilityRepositoryProvider';
}

@ProviderFor(accountEligibility)
final accountEligibilityProvider = AccountEligibilityFamily._();

final class AccountEligibilityProvider
    extends
        $FunctionalProvider<
          AsyncValue<AccountEligibilityStatus>,
          AccountEligibilityStatus,
          FutureOr<AccountEligibilityStatus>
        >
    with
        $FutureModifier<AccountEligibilityStatus>,
        $FutureProvider<AccountEligibilityStatus> {
  AccountEligibilityProvider._({
    required AccountEligibilityFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'accountEligibilityProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountEligibilityHash();

  @override
  String toString() {
    return r'accountEligibilityProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<AccountEligibilityStatus> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<AccountEligibilityStatus> create(Ref ref) {
    final argument = this.argument as ActiveAccountLease;
    return accountEligibility(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountEligibilityProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountEligibilityHash() =>
    r'e416695a53412aa1d3d5320df3dd281471ca52fe';

final class AccountEligibilityFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<AccountEligibilityStatus>,
          ActiveAccountLease
        > {
  AccountEligibilityFamily._()
    : super(
        retry: null,
        name: r'accountEligibilityProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountEligibilityProvider call(ActiveAccountLease lease) =>
      AccountEligibilityProvider._(argument: lease, from: this);

  @override
  String toString() => r'accountEligibilityProvider';
}
