// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'onboarding_repository_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(onboardingRepository)
final onboardingRepositoryProvider = OnboardingRepositoryFamily._();

final class OnboardingRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<OnboardingRepository>,
          OnboardingRepository,
          FutureOr<OnboardingRepository>
        >
    with
        $FutureModifier<OnboardingRepository>,
        $FutureProvider<OnboardingRepository> {
  OnboardingRepositoryProvider._({
    required OnboardingRepositoryFamily super.from,
    required AccountSessionLease super.argument,
  }) : super(
         retry: null,
         name: r'onboardingRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$onboardingRepositoryHash();

  @override
  String toString() {
    return r'onboardingRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<OnboardingRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<OnboardingRepository> create(Ref ref) {
    final argument = this.argument as AccountSessionLease;
    return onboardingRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is OnboardingRepositoryProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$onboardingRepositoryHash() =>
    r'4cca6071a2c9bb3771f1704319c59654bb3cc379';

final class OnboardingRepositoryFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<OnboardingRepository>,
          AccountSessionLease
        > {
  OnboardingRepositoryFamily._()
    : super(
        retry: null,
        name: r'onboardingRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  OnboardingRepositoryProvider call(AccountSessionLease lease) =>
      OnboardingRepositoryProvider._(argument: lease, from: this);

  @override
  String toString() => r'onboardingRepositoryProvider';
}
