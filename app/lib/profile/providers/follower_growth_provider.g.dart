// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'follower_growth_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(followerGrowth)
final followerGrowthProvider = FollowerGrowthFamily._();

final class FollowerGrowthProvider
    extends
        $FunctionalProvider<
          AsyncValue<FollowerGrowth>,
          FollowerGrowth,
          FutureOr<FollowerGrowth>
        >
    with $FutureModifier<FollowerGrowth>, $FutureProvider<FollowerGrowth> {
  FollowerGrowthProvider._({
    required FollowerGrowthFamily super.from,
    required (AccountKey, FollowerGrowthPeriod) super.argument,
  }) : super(
         retry: null,
         name: r'followerGrowthProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$followerGrowthHash();

  @override
  String toString() {
    return r'followerGrowthProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<FollowerGrowth> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<FollowerGrowth> create(Ref ref) {
    final argument = this.argument as (AccountKey, FollowerGrowthPeriod);
    return followerGrowth(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is FollowerGrowthProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$followerGrowthHash() => r'13e051e7ca998805204da9b32f134386bdd18bff';

final class FollowerGrowthFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<FollowerGrowth>,
          (AccountKey, FollowerGrowthPeriod)
        > {
  FollowerGrowthFamily._()
    : super(
        retry: null,
        name: r'followerGrowthProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  FollowerGrowthProvider call(
    AccountKey account,
    FollowerGrowthPeriod period,
  ) => FollowerGrowthProvider._(argument: (account, period), from: this);

  @override
  String toString() => r'followerGrowthProvider';
}
