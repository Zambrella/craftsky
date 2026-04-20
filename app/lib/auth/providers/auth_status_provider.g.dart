// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_status_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Stubbed auth status. Real implementation will be backed by the app-view
/// session token once atproto OAuth is wired up.
///
/// Exposes explicit `signIn` / `signOut` methods rather than a generic
/// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).

@ProviderFor(AuthStatus)
final authStatusProvider = AuthStatusProvider._();

/// Stubbed auth status. Real implementation will be backed by the app-view
/// session token once atproto OAuth is wired up.
///
/// Exposes explicit `signIn` / `signOut` methods rather than a generic
/// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).
final class AuthStatusProvider extends $NotifierProvider<AuthStatus, bool> {
  /// Stubbed auth status. Real implementation will be backed by the app-view
  /// session token once atproto OAuth is wired up.
  ///
  /// Exposes explicit `signIn` / `signOut` methods rather than a generic
  /// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).
  AuthStatusProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authStatusProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authStatusHash();

  @$internal
  @override
  AuthStatus create() => AuthStatus();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(bool value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<bool>(value),
    );
  }
}

String _$authStatusHash() => r'3c0cec2f0bcf0def3fd6b3a83f79161acf89b945';

/// Stubbed auth status. Real implementation will be backed by the app-view
/// session token once atproto OAuth is wired up.
///
/// Exposes explicit `signIn` / `signOut` methods rather than a generic
/// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).

abstract class _$AuthStatus extends $Notifier<bool> {
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
