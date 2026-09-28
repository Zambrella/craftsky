// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'moderation_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountModerationRepository)
final accountModerationRepositoryProvider =
    AccountModerationRepositoryFamily._();

final class AccountModerationRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ModerationRepository>,
          ModerationRepository,
          FutureOr<ModerationRepository>
        >
    with
        $FutureModifier<ModerationRepository>,
        $FutureProvider<ModerationRepository> {
  AccountModerationRepositoryProvider._({
    required AccountModerationRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountModerationRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountModerationRepositoryHash();

  @override
  String toString() {
    return r'accountModerationRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ModerationRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ModerationRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountModerationRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountModerationRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountModerationRepositoryHash() =>
    r'60dc1d97973a5c9a008d88cd19024d816d1d1638';

final class AccountModerationRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ModerationRepository>, AccountKey> {
  AccountModerationRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountModerationRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountModerationRepositoryProvider call(AccountKey account) =>
      AccountModerationRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountModerationRepositoryProvider';
}

@ProviderFor(accountStanding)
final accountStandingProvider = AccountStandingFamily._();

final class AccountStandingProvider
    extends
        $FunctionalProvider<
          AsyncValue<AccountStanding>,
          AccountStanding,
          FutureOr<AccountStanding>
        >
    with $FutureModifier<AccountStanding>, $FutureProvider<AccountStanding> {
  AccountStandingProvider._({
    required AccountStandingFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountStandingProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountStandingHash();

  @override
  String toString() {
    return r'accountStandingProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<AccountStanding> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<AccountStanding> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountStanding(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountStandingProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountStandingHash() => r'3194c279825d2a59acfcbb19ac3b42a9b26fb35a';

final class AccountStandingFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<AccountStanding>, AccountKey> {
  AccountStandingFamily._()
    : super(
        retry: null,
        name: r'accountStandingProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountStandingProvider call(AccountKey account) =>
      AccountStandingProvider._(argument: account, from: this);

  @override
  String toString() => r'accountStandingProvider';
}

@ProviderFor(AccountModerationHistory)
final accountModerationHistoryProvider = AccountModerationHistoryFamily._();

final class AccountModerationHistoryProvider
    extends
        $AsyncNotifierProvider<
          AccountModerationHistory,
          ModerationHistoryState
        > {
  AccountModerationHistoryProvider._({
    required AccountModerationHistoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountModerationHistoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountModerationHistoryHash();

  @override
  String toString() {
    return r'accountModerationHistoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  AccountModerationHistory create() => AccountModerationHistory();

  @override
  bool operator ==(Object other) {
    return other is AccountModerationHistoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountModerationHistoryHash() =>
    r'c94292f15fd39fb9f4670b0c84d225497a7df32b';

final class AccountModerationHistoryFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountModerationHistory,
          AsyncValue<ModerationHistoryState>,
          ModerationHistoryState,
          FutureOr<ModerationHistoryState>,
          AccountKey
        > {
  AccountModerationHistoryFamily._()
    : super(
        retry: null,
        name: r'accountModerationHistoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountModerationHistoryProvider call(AccountKey account) =>
      AccountModerationHistoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountModerationHistoryProvider';
}

abstract class _$AccountModerationHistory
    extends $AsyncNotifier<ModerationHistoryState> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<ModerationHistoryState> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<ModerationHistoryState>, ModerationHistoryState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<ModerationHistoryState>,
                ModerationHistoryState
              >,
              AsyncValue<ModerationHistoryState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}

@ProviderFor(accountModerationHistoryEntry)
final accountModerationHistoryEntryProvider =
    AccountModerationHistoryEntryFamily._();

final class AccountModerationHistoryEntryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ModerationHistoryPage>,
          ModerationHistoryPage,
          FutureOr<ModerationHistoryPage>
        >
    with
        $FutureModifier<ModerationHistoryPage>,
        $FutureProvider<ModerationHistoryPage> {
  AccountModerationHistoryEntryProvider._({
    required AccountModerationHistoryEntryFamily super.from,
    required (AccountKey, ModerationCaseReference) super.argument,
  }) : super(
         retry: null,
         name: r'accountModerationHistoryEntryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountModerationHistoryEntryHash();

  @override
  String toString() {
    return r'accountModerationHistoryEntryProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ModerationHistoryPage> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ModerationHistoryPage> create(Ref ref) {
    final argument = this.argument as (AccountKey, ModerationCaseReference);
    return accountModerationHistoryEntry(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountModerationHistoryEntryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountModerationHistoryEntryHash() =>
    r'684ad4e52cee79348b4bdf004f7abb15dee8815a';

final class AccountModerationHistoryEntryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ModerationHistoryPage>,
          (AccountKey, ModerationCaseReference)
        > {
  AccountModerationHistoryEntryFamily._()
    : super(
        retry: null,
        name: r'accountModerationHistoryEntryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountModerationHistoryEntryProvider call(
    AccountKey account,
    ModerationCaseReference reference,
  ) => AccountModerationHistoryEntryProvider._(
    argument: (account, reference),
    from: this,
  );

  @override
  String toString() => r'accountModerationHistoryEntryProvider';
}
