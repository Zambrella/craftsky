// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'scheduled_post_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountScheduledPostRepository)
final accountScheduledPostRepositoryProvider =
    AccountScheduledPostRepositoryFamily._();

final class AccountScheduledPostRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ScheduledPostRepository>,
          ScheduledPostRepository,
          FutureOr<ScheduledPostRepository>
        >
    with
        $FutureModifier<ScheduledPostRepository>,
        $FutureProvider<ScheduledPostRepository> {
  AccountScheduledPostRepositoryProvider._({
    required AccountScheduledPostRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountScheduledPostRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountScheduledPostRepositoryHash();

  @override
  String toString() {
    return r'accountScheduledPostRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ScheduledPostRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ScheduledPostRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountScheduledPostRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountScheduledPostRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountScheduledPostRepositoryHash() =>
    r'88e93e20e827290561646e6617793516c2f5f1b3';

final class AccountScheduledPostRepositoryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ScheduledPostRepository>,
          AccountKey
        > {
  AccountScheduledPostRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountScheduledPostRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountScheduledPostRepositoryProvider call(AccountKey account) =>
      AccountScheduledPostRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountScheduledPostRepositoryProvider';
}
