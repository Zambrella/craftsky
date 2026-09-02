// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'onboarding_status_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(OnboardingStatus)
final onboardingStatusProvider = OnboardingStatusFamily._();

final class OnboardingStatusProvider
    extends $AsyncNotifierProvider<OnboardingStatus, OnboardingCompletion> {
  OnboardingStatusProvider._({
    required OnboardingStatusFamily super.from,
    required AccountSessionLease super.argument,
  }) : super(
         retry: null,
         name: r'onboardingStatusProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$onboardingStatusHash();

  @override
  String toString() {
    return r'onboardingStatusProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  OnboardingStatus create() => OnboardingStatus();

  @override
  bool operator ==(Object other) {
    return other is OnboardingStatusProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$onboardingStatusHash() => r'87c381dc0ed23e4f0db0388a019fe0942d72357d';

final class OnboardingStatusFamily extends $Family
    with
        $ClassFamilyOverride<
          OnboardingStatus,
          AsyncValue<OnboardingCompletion>,
          OnboardingCompletion,
          FutureOr<OnboardingCompletion>,
          AccountSessionLease
        > {
  OnboardingStatusFamily._()
    : super(
        retry: null,
        name: r'onboardingStatusProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  OnboardingStatusProvider call(AccountSessionLease lease) =>
      OnboardingStatusProvider._(argument: lease, from: this);

  @override
  String toString() => r'onboardingStatusProvider';
}

abstract class _$OnboardingStatus extends $AsyncNotifier<OnboardingCompletion> {
  late final _$args = ref.$arg as AccountSessionLease;
  AccountSessionLease get lease => _$args;

  FutureOr<OnboardingCompletion> build(AccountSessionLease lease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<OnboardingCompletion>, OnboardingCompletion>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<OnboardingCompletion>,
                OnboardingCompletion
              >,
              AsyncValue<OnboardingCompletion>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
