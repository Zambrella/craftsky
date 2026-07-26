// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'account_saved_post_state_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountSavedPostState)
final accountSavedPostStateProvider = AccountSavedPostStateFamily._();

final class AccountSavedPostStateProvider
    extends
        $AsyncNotifierProvider<
          AccountSavedPostState,
          AccountSavedPostStateMap
        > {
  AccountSavedPostStateProvider._({
    required AccountSavedPostStateFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountSavedPostStateProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountSavedPostStateHash();

  @override
  String toString() {
    return r'accountSavedPostStateProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  AccountSavedPostState create() => AccountSavedPostState();

  @override
  bool operator ==(Object other) {
    return other is AccountSavedPostStateProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountSavedPostStateHash() =>
    r'b1aec61dbfdbe4756f7b0e1d206f8bdf17ca6093';

final class AccountSavedPostStateFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountSavedPostState,
          AsyncValue<AccountSavedPostStateMap>,
          AccountSavedPostStateMap,
          FutureOr<AccountSavedPostStateMap>,
          AccountKey
        > {
  AccountSavedPostStateFamily._()
    : super(
        retry: null,
        name: r'accountSavedPostStateProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  AccountSavedPostStateProvider call(AccountKey account) =>
      AccountSavedPostStateProvider._(argument: account, from: this);

  @override
  String toString() => r'accountSavedPostStateProvider';
}

abstract class _$AccountSavedPostState
    extends $AsyncNotifier<AccountSavedPostStateMap> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<AccountSavedPostStateMap> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<AccountSavedPostStateMap>,
              AccountSavedPostStateMap
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<AccountSavedPostStateMap>,
                AccountSavedPostStateMap
              >,
              AsyncValue<AccountSavedPostStateMap>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

@ProviderFor(savedPostPresentation)
final savedPostPresentationProvider = SavedPostPresentationFamily._();

final class SavedPostPresentationProvider
    extends
        $FunctionalProvider<
          AsyncValue<SavedPostPresentation>,
          AsyncValue<SavedPostPresentation>,
          AsyncValue<SavedPostPresentation>
        >
    with $Provider<AsyncValue<SavedPostPresentation>> {
  SavedPostPresentationProvider._({
    required SavedPostPresentationFamily super.from,
    required SavedPostKey super.argument,
  }) : super(
         retry: null,
         name: r'savedPostPresentationProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$savedPostPresentationHash();

  @override
  String toString() {
    return r'savedPostPresentationProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<AsyncValue<SavedPostPresentation>> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  AsyncValue<SavedPostPresentation> create(Ref ref) {
    final argument = this.argument as SavedPostKey;
    return savedPostPresentation(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AsyncValue<SavedPostPresentation> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AsyncValue<SavedPostPresentation>>(
        value,
      ),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is SavedPostPresentationProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$savedPostPresentationHash() =>
    r'e7c1c63c73fa8611a55eb508d6aac169e54ae9b3';

final class SavedPostPresentationFamily extends $Family
    with
        $FunctionalFamilyOverride<
          AsyncValue<SavedPostPresentation>,
          SavedPostKey
        > {
  SavedPostPresentationFamily._()
    : super(
        retry: null,
        name: r'savedPostPresentationProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SavedPostPresentationProvider call(SavedPostKey key) =>
      SavedPostPresentationProvider._(argument: key, from: this);

  @override
  String toString() => r'savedPostPresentationProvider';
}
