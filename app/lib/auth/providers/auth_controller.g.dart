// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(launchAuthUrl)
final launchAuthUrlProvider = LaunchAuthUrlProvider._();

final class LaunchAuthUrlProvider
    extends
        $FunctionalProvider<AuthUrlLauncher, AuthUrlLauncher, AuthUrlLauncher>
    with $Provider<AuthUrlLauncher> {
  LaunchAuthUrlProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'launchAuthUrlProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$launchAuthUrlHash();

  @$internal
  @override
  $ProviderElement<AuthUrlLauncher> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  AuthUrlLauncher create(Ref ref) {
    return launchAuthUrl(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(AuthUrlLauncher value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<AuthUrlLauncher>(value),
    );
  }
}

String _$launchAuthUrlHash() => r'a1311cb31922e425e20601c8c71983c5a16025e7';

/// Sign-in / sign-out orchestrator. Exposes `AsyncValue<void>`; pages
/// listen for `AsyncError(AuthError)` transitions via `ref.listen`.
///
/// Tests that need to simulate a stale `PendingAuth` do so via
/// `pendingAuthProvider.notifier.debugSet(...)` (defined on the
/// `PendingAuth` notifier in Task 13), not through this controller.

@ProviderFor(AuthController)
final authControllerProvider = AuthControllerProvider._();

/// Sign-in / sign-out orchestrator. Exposes `AsyncValue<void>`; pages
/// listen for `AsyncError(AuthError)` transitions via `ref.listen`.
///
/// Tests that need to simulate a stale `PendingAuth` do so via
/// `pendingAuthProvider.notifier.debugSet(...)` (defined on the
/// `PendingAuth` notifier in Task 13), not through this controller.
final class AuthControllerProvider
    extends $AsyncNotifierProvider<AuthController, void> {
  /// Sign-in / sign-out orchestrator. Exposes `AsyncValue<void>`; pages
  /// listen for `AsyncError(AuthError)` transitions via `ref.listen`.
  ///
  /// Tests that need to simulate a stale `PendingAuth` do so via
  /// `pendingAuthProvider.notifier.debugSet(...)` (defined on the
  /// `PendingAuth` notifier in Task 13), not through this controller.
  AuthControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authControllerProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authControllerHash();

  @$internal
  @override
  AuthController create() => AuthController();
}

String _$authControllerHash() => r'd939de56867ee96a70dbcb09eab738e5ee097238';

/// Sign-in / sign-out orchestrator. Exposes `AsyncValue<void>`; pages
/// listen for `AsyncError(AuthError)` transitions via `ref.listen`.
///
/// Tests that need to simulate a stale `PendingAuth` do so via
/// `pendingAuthProvider.notifier.debugSet(...)` (defined on the
/// `PendingAuth` notifier in Task 13), not through this controller.

abstract class _$AuthController extends $AsyncNotifier<void> {
  FutureOr<void> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
