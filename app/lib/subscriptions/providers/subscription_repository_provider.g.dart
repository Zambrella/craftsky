// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subscription_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(subscriptionRepository)
final subscriptionRepositoryProvider = SubscriptionRepositoryFamily._();

final class SubscriptionRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<SubscriptionApi>,
          SubscriptionApi,
          FutureOr<SubscriptionApi>
        >
    with $FutureModifier<SubscriptionApi>, $FutureProvider<SubscriptionApi> {
  SubscriptionRepositoryProvider._({
    required SubscriptionRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'subscriptionRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$subscriptionRepositoryHash();

  @override
  String toString() {
    return r'subscriptionRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<SubscriptionApi> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<SubscriptionApi> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return subscriptionRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is SubscriptionRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$subscriptionRepositoryHash() =>
    r'67fb13b3dbdee266912fd60d381b210081f36bfb';

final class SubscriptionRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<SubscriptionApi>, AccountKey> {
  SubscriptionRepositoryFamily._()
    : super(
        retry: null,
        name: r'subscriptionRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SubscriptionRepositoryProvider call(AccountKey account) =>
      SubscriptionRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'subscriptionRepositoryProvider';
}
