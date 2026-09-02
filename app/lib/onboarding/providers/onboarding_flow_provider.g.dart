// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'onboarding_flow_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(OnboardingFlow)
final onboardingFlowProvider = OnboardingFlowFamily._();

final class OnboardingFlowProvider
    extends $AsyncNotifierProvider<OnboardingFlow, OnboardingFlowState> {
  OnboardingFlowProvider._({
    required OnboardingFlowFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'onboardingFlowProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$onboardingFlowHash();

  @override
  String toString() {
    return r'onboardingFlowProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  OnboardingFlow create() => OnboardingFlow();

  @override
  bool operator ==(Object other) {
    return other is OnboardingFlowProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$onboardingFlowHash() => r'847f18cb6ecf4942fa60f26cae566814a641a58c';

final class OnboardingFlowFamily extends $Family
    with
        $ClassFamilyOverride<
          OnboardingFlow,
          AsyncValue<OnboardingFlowState>,
          OnboardingFlowState,
          FutureOr<OnboardingFlowState>,
          ActiveAccountLease
        > {
  OnboardingFlowFamily._()
    : super(
        retry: null,
        name: r'onboardingFlowProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  OnboardingFlowProvider call(ActiveAccountLease lease) =>
      OnboardingFlowProvider._(argument: lease, from: this);

  @override
  String toString() => r'onboardingFlowProvider';
}

abstract class _$OnboardingFlow extends $AsyncNotifier<OnboardingFlowState> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get lease => _$args;

  FutureOr<OnboardingFlowState> build(ActiveAccountLease lease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<OnboardingFlowState>, OnboardingFlowState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<OnboardingFlowState>, OnboardingFlowState>,
              AsyncValue<OnboardingFlowState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
