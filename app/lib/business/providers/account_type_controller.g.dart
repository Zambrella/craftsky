// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'account_type_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountTypeProfileReconciler)
final accountTypeProfileReconcilerProvider =
    AccountTypeProfileReconcilerProvider._();

final class AccountTypeProfileReconcilerProvider
    extends
        $FunctionalProvider<
          AccountTypeReconciler,
          AccountTypeReconciler,
          AccountTypeReconciler
        >
    with $Provider<AccountTypeReconciler> {
  AccountTypeProfileReconcilerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'accountTypeProfileReconcilerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$accountTypeProfileReconcilerHash();

  @$internal
  @override
  $ProviderElement<AccountTypeReconciler> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  AccountTypeReconciler create(Ref ref) {
    return accountTypeProfileReconciler(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AccountTypeReconciler value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AccountTypeReconciler>(value),
    );
  }
}

String _$accountTypeProfileReconcilerHash() =>
    r'4755dc3a1795647d382213f57d89d4aedd77f3f0';

@ProviderFor(accountTypeStateInvalidator)
final accountTypeStateInvalidatorProvider =
    AccountTypeStateInvalidatorProvider._();

final class AccountTypeStateInvalidatorProvider
    extends
        $FunctionalProvider<
          AccountTypeReconciler,
          AccountTypeReconciler,
          AccountTypeReconciler
        >
    with $Provider<AccountTypeReconciler> {
  AccountTypeStateInvalidatorProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'accountTypeStateInvalidatorProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$accountTypeStateInvalidatorHash();

  @$internal
  @override
  $ProviderElement<AccountTypeReconciler> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  AccountTypeReconciler create(Ref ref) {
    return accountTypeStateInvalidator(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AccountTypeReconciler value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AccountTypeReconciler>(value),
    );
  }
}

String _$accountTypeStateInvalidatorHash() =>
    r'd95531ad5b78d3a98233af41366cef7dc16cc435';

@ProviderFor(AccountTypeController)
final accountTypeControllerProvider = AccountTypeControllerProvider._();

final class AccountTypeControllerProvider
    extends $AsyncNotifierProvider<AccountTypeController, AccountType?> {
  AccountTypeControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'accountTypeControllerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$accountTypeControllerHash();

  @$internal
  @override
  AccountTypeController create() => AccountTypeController();
}

String _$accountTypeControllerHash() =>
    r'51ae932c51e8511b1c98e12db608e260aacfb2f1';

abstract class _$AccountTypeController extends $AsyncNotifier<AccountType?> {
  FutureOr<AccountType?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<AccountType?>, AccountType?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<AccountType?>, AccountType?>,
              AsyncValue<AccountType?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
