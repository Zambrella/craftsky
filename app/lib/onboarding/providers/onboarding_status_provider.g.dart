// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'onboarding_status_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
/// survives relaunch but not reinstall on Android (clear-app-data
/// semantics). First-run for a new DID defaults to `false`.
///
/// `@riverpod` codegen exposes the family arg as an instance field
/// (`did`) on the generated notifier base class, so both `build` and
/// `finish` reference `did` directly.

@ProviderFor(OnboardingStatus)
final onboardingStatusProvider = OnboardingStatusFamily._();

/// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
/// survives relaunch but not reinstall on Android (clear-app-data
/// semantics). First-run for a new DID defaults to `false`.
///
/// `@riverpod` codegen exposes the family arg as an instance field
/// (`did`) on the generated notifier base class, so both `build` and
/// `finish` reference `did` directly.
final class OnboardingStatusProvider
    extends $NotifierProvider<OnboardingStatus, bool> {
  /// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
  /// survives relaunch but not reinstall on Android (clear-app-data
  /// semantics). First-run for a new DID defaults to `false`.
  ///
  /// `@riverpod` codegen exposes the family arg as an instance field
  /// (`did`) on the generated notifier base class, so both `build` and
  /// `finish` reference `did` directly.
  OnboardingStatusProvider._({
    required OnboardingStatusFamily super.from,
    required Did super.argument,
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

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is OnboardingStatusProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$onboardingStatusHash() => r'30a86886e3dcb13cf4fdecfac271b23206cd797e';

/// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
/// survives relaunch but not reinstall on Android (clear-app-data
/// semantics). First-run for a new DID defaults to `false`.
///
/// `@riverpod` codegen exposes the family arg as an instance field
/// (`did`) on the generated notifier base class, so both `build` and
/// `finish` reference `did` directly.

final class OnboardingStatusFamily extends $Family
    with $ClassFamilyOverride<OnboardingStatus, bool, bool, bool, Did> {
  OnboardingStatusFamily._()
    : super(
        retry: null,
        name: r'onboardingStatusProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
  /// survives relaunch but not reinstall on Android (clear-app-data
  /// semantics). First-run for a new DID defaults to `false`.
  ///
  /// `@riverpod` codegen exposes the family arg as an instance field
  /// (`did`) on the generated notifier base class, so both `build` and
  /// `finish` reference `did` directly.

  OnboardingStatusProvider call(Did did) =>
      OnboardingStatusProvider._(argument: did, from: this);

  @override
  String toString() => r'onboardingStatusProvider';
}

/// Per-DID onboarding completion flag. Backed by `SharedPreferences`;
/// survives relaunch but not reinstall on Android (clear-app-data
/// semantics). First-run for a new DID defaults to `false`.
///
/// `@riverpod` codegen exposes the family arg as an instance field
/// (`did`) on the generated notifier base class, so both `build` and
/// `finish` reference `did` directly.

abstract class _$OnboardingStatus extends $Notifier<bool> {
  late final _$args = ref.$arg as Did;
  Did get did => _$args;

  bool build(Did did);
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
    element.handleCreate(ref, () => build(_$args));
  }
}
