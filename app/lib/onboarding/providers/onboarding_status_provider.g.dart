// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'onboarding_status_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Stubbed onboarding completion status. Real implementation will be backed
/// by the user's profile record once onboarding actually persists data.

@ProviderFor(OnboardingStatus)
final onboardingStatusProvider = OnboardingStatusProvider._();

/// Stubbed onboarding completion status. Real implementation will be backed
/// by the user's profile record once onboarding actually persists data.
final class OnboardingStatusProvider
    extends $NotifierProvider<OnboardingStatus, bool> {
  /// Stubbed onboarding completion status. Real implementation will be backed
  /// by the user's profile record once onboarding actually persists data.
  OnboardingStatusProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'onboardingStatusProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$onboardingStatusHash();

  @$internal
  @override
  OnboardingStatus create() => OnboardingStatus();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }
}

String _$onboardingStatusHash() => r'06a4dfa2b0d396252a31c26613f83418fdc10d5d';

/// Stubbed onboarding completion status. Real implementation will be backed
/// by the user's profile record once onboarding actually persists data.

abstract class _$OnboardingStatus extends $Notifier<bool> {
  bool build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<bool, bool>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<bool, bool>,
              bool,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
