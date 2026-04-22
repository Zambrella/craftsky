import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/api_client_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_session_provider.g.dart';

/// Sole source of truth for the app's auth state. Cold start reads
/// secure storage once and emits an optimistic `SignedIn` immediately
/// (if a session exists), then background-validates via `/whoami`.
/// Later updates come through `setSignedIn` / `setSignedOut`, called
/// by `AuthController` and the global 401 interceptor.
@Riverpod(keepAlive: true)
class AuthSession extends _$AuthSession {
  @override
  Future<AuthState> build() async {
    final storage = ref.watch(secureTokenStorageProvider);
    final stored = await storage.read();
    if (stored == null) return const SignedOut();

    // Unawaited — we return SignedIn now and validate in parallel.
    unawaited(_validateInBackground(stored));

    return SignedIn(
      did: stored.did,
      handle: stored.handle,
      token: stored.token,
    );
  }

  Future<void> _validateInBackground(StoredSession stored) async {
    try {
      final api = ref.read(craftskyApiClientProvider);
      final who = await api.whoami();
      if (!ref.mounted) return;

      if (who.did != stored.did) {
        await _clearLocalState();
        return;
      }
      if (who.handle != stored.handle) {
        final updated = StoredSession(
          token: stored.token,
          did: who.did,
          handle: who.handle,
        );
        await ref.read(secureTokenStorageProvider).write(updated);
        if (!ref.mounted) return;
        state = AsyncData(
          SignedIn(did: who.did, handle: who.handle, token: stored.token),
        );
      }
      // else: handles match; nothing to do.
    } on ApiUnauthorized {
      await _clearLocalState();
    } on ApiNetworkError {
      // Offline; keep cached SignedIn. Next cold start revalidates.
    }
  }

  Future<void> _clearLocalState() async {
    await ref.read(secureTokenStorageProvider).clear();
    if (!ref.mounted) return;
    state = const AsyncData(SignedOut());
  }

  void setSignedIn(SignedIn signedIn) => state = AsyncData(signedIn);

  void setSignedOut() => state = const AsyncData(SignedOut());
}
