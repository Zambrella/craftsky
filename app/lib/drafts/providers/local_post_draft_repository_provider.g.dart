// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'local_post_draft_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountLocalPostDraftRepository)
final accountLocalPostDraftRepositoryProvider =
    AccountLocalPostDraftRepositoryFamily._();

final class AccountLocalPostDraftRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<LocalPostDraftRepository>,
          LocalPostDraftRepository,
          FutureOr<LocalPostDraftRepository>
        >
    with
        $FutureModifier<LocalPostDraftRepository>,
        $FutureProvider<LocalPostDraftRepository> {
  AccountLocalPostDraftRepositoryProvider._({
    required AccountLocalPostDraftRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountLocalPostDraftRepositoryProvider',
         isAutoDispose: false,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountLocalPostDraftRepositoryHash();

  @override
  String toString() {
    return r'accountLocalPostDraftRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<LocalPostDraftRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<LocalPostDraftRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountLocalPostDraftRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountLocalPostDraftRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountLocalPostDraftRepositoryHash() =>
    r'd5c01eb552622055eb7bec5400273c0f2f4a4517';

final class AccountLocalPostDraftRepositoryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<LocalPostDraftRepository>,
          AccountKey
        > {
  AccountLocalPostDraftRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountLocalPostDraftRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: false,
      );

  AccountLocalPostDraftRepositoryProvider call(AccountKey account) =>
      AccountLocalPostDraftRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountLocalPostDraftRepositoryProvider';
}
