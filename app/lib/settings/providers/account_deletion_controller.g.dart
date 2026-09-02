// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'account_deletion_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountDeletionController)
final accountDeletionControllerProvider = AccountDeletionControllerProvider._();

final class AccountDeletionControllerProvider
    extends $AsyncNotifierProvider<AccountDeletionController, void> {
  AccountDeletionControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'accountDeletionControllerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$accountDeletionControllerHash();

  @$internal
  @override
  AccountDeletionController create() => AccountDeletionController();
}

String _$accountDeletionControllerHash() =>
    r'3ea7582fe4d388d21e3c90868a1fb770f2d9750f';

abstract class _$AccountDeletionController extends $AsyncNotifier<void> {
  FutureOr<void> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
