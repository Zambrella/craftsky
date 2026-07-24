// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'saved_post_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountSavedPostRepository)
final accountSavedPostRepositoryProvider = AccountSavedPostRepositoryFamily._();

final class AccountSavedPostRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<SavedPostRepository>,
          SavedPostRepository,
          FutureOr<SavedPostRepository>
        >
    with
        $FutureModifier<SavedPostRepository>,
        $FutureProvider<SavedPostRepository> {
  AccountSavedPostRepositoryProvider._({
    required AccountSavedPostRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountSavedPostRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountSavedPostRepositoryHash();

  @override
  String toString() {
    return r'accountSavedPostRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<SavedPostRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<SavedPostRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountSavedPostRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountSavedPostRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountSavedPostRepositoryHash() =>
    r'faaa87b4075f7941375e6b54e15e33d2d45256e9';

final class AccountSavedPostRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<SavedPostRepository>, AccountKey> {
  AccountSavedPostRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountSavedPostRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountSavedPostRepositoryProvider call(AccountKey account) =>
      AccountSavedPostRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountSavedPostRepositoryProvider';
}
