// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'auth_session_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Sole source of truth for the app's auth state. Cold start reads
/// secure storage once and emits an optimistic `SignedIn` immediately
/// (if a session exists), then background-validates via `/whoami`.
/// Later updates come through `setSignedIn` / `setSignedOut`, called
/// by `AuthController` and the global 401 interceptor.

@ProviderFor(AuthSession)
final authSessionProvider = AuthSessionProvider._();

/// Sole source of truth for the app's auth state. Cold start reads
/// secure storage once and emits an optimistic `SignedIn` immediately
/// (if a session exists), then background-validates via `/whoami`.
/// Later updates come through `setSignedIn` / `setSignedOut`, called
/// by `AuthController` and the global 401 interceptor.
final class AuthSessionProvider
    extends $AsyncNotifierProvider<AuthSession, AuthState> {
  /// Sole source of truth for the app's auth state. Cold start reads
  /// secure storage once and emits an optimistic `SignedIn` immediately
  /// (if a session exists), then background-validates via `/whoami`.
  /// Later updates come through `setSignedIn` / `setSignedOut`, called
  /// by `AuthController` and the global 401 interceptor.
  AuthSessionProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'authSessionProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$authSessionHash();

  @$internal
  @override
  AuthSession create() => AuthSession();
}

String _$authSessionHash() => r'6742931a12780af5f7193be5dea320d681c0b06c';

/// Sole source of truth for the app's auth state. Cold start reads
/// secure storage once and emits an optimistic `SignedIn` immediately
/// (if a session exists), then background-validates via `/whoami`.
/// Later updates come through `setSignedIn` / `setSignedOut`, called
/// by `AuthController` and the global 401 interceptor.

abstract class _$AuthSession extends $AsyncNotifier<AuthState> {
  FutureOr<AuthState> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<AuthState>, AuthState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<AuthState>, AuthState>,
              AsyncValue<AuthState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
