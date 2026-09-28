// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'subscription_access_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(subscriptionAccess)
final subscriptionAccessProvider = SubscriptionAccessFamily._();

final class SubscriptionAccessProvider
    extends
        $FunctionalProvider<
          AsyncValue<SubscriptionAccess>,
          SubscriptionAccess,
          FutureOr<SubscriptionAccess>
        >
    with
        $FutureModifier<SubscriptionAccess>,
        $FutureProvider<SubscriptionAccess> {
  SubscriptionAccessProvider._({
    required SubscriptionAccessFamily super.from,
    required AccountSessionLease super.argument,
  }) : super(
         retry: null,
         name: r'subscriptionAccessProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$subscriptionAccessHash();

  @override
  String toString() {
    return r'subscriptionAccessProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<SubscriptionAccess> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<SubscriptionAccess> create(Ref ref) {
    final argument = this.argument as AccountSessionLease;
    return subscriptionAccess(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is SubscriptionAccessProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$subscriptionAccessHash() =>
    r'04ff4ca306d7e8f4d08e42136da43eca39345605';

final class SubscriptionAccessFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<SubscriptionAccess>,
          AccountSessionLease
        > {
  SubscriptionAccessFamily._()
    : super(
        retry: null,
        name: r'subscriptionAccessProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SubscriptionAccessProvider call(AccountSessionLease lease) =>
      SubscriptionAccessProvider._(argument: lease, from: this);

  @override
  String toString() => r'subscriptionAccessProvider';
}
