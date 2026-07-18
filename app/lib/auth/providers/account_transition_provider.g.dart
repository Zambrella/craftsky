// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'account_transition_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountTransitionState)
final accountTransitionStateProvider = AccountTransitionStateProvider._();

final class AccountTransitionStateProvider
    extends $NotifierProvider<AccountTransitionState, AccountTransition?> {
  AccountTransitionStateProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'accountTransitionStateProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$accountTransitionStateHash();

  @$internal
  @override
  AccountTransitionState create() => AccountTransitionState();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AccountTransition? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AccountTransition?>(value),
    );
  }
}

String _$accountTransitionStateHash() =>
    r'b87195cbbf8d4d46d7e8cba9b0cc003912031464';

abstract class _$AccountTransitionState extends $Notifier<AccountTransition?> {
  AccountTransition? build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AccountTransition?, AccountTransition?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AccountTransition?, AccountTransition?>,
              AccountTransition?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
