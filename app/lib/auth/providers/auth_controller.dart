import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/auth_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/handoff_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:url_launcher/url_launcher.dart' as url_launcher;

part 'auth_controller.g.dart';

final _log = Logger('AuthController');

/// The URL-launch function. Overridable in tests so we don't trigger
/// the real system browser.
typedef AuthUrlLauncher = Future<bool> Function(Uri uri);

@Riverpod(keepAlive: true)
AuthUrlLauncher launchAuthUrl(Ref ref) {
  return (Uri uri) => url_launcher.launchUrl(
    uri,
    mode: url_launcher.LaunchMode.externalApplication,
  );
}

/// Sign-in / sign-out orchestrator. Exposes `AsyncValue<void>`; pages
/// listen for `AsyncError(AuthError)` transitions via `ref.listen`.
///
/// Tests that need to simulate a stale `PendingAuth` do so via
/// `pendingAuthProvider.notifier.debugSet(...)` (defined on the
/// `PendingAuth` notifier in Task 13), not through this controller.
@Riverpod(keepAlive: true)
class AuthController extends _$AuthController {
  @override
  FutureOr<void> build() => null;

  Future<void> signIn({required String handle}) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final trimmed = handle.trim().replaceFirst(RegExp('^@'), '');
      if (trimmed.isEmpty) throw const HandleRequired();

      final api = ref.read(authApiClientProvider);
      final LoginResponse response;
      try {
        response = await api.login(handle: trimmed);
      } on ApiException catch (e) {
        throw switch (e) {
          ApiBadRequest(code: 'handle_required') => const HandleRequired(),
          ApiBadRequest() => const InvalidHandle(),
          ApiNetworkError() ||
          ApiServerError() ||
          ApiUnauthorized() ||
          ApiCanceled() => const ServerUnavailable(),
        };
      }

      if (!ref.mounted) return;
      ref.read(pendingAuthProvider.notifier).start(trimmed);

      final launched = await ref.read(launchAuthUrlProvider)(
        Uri.parse(response.authUrl),
      );
      if (!launched) {
        if (ref.mounted) ref.read(pendingAuthProvider.notifier).clear();
        throw const BrowserLaunchFailed();
      }
    });
  }

  Future<void> completeFromDeepLink(String token) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final pending = ref.read(pendingAuthProvider);
      if (pending == null) throw const NoPendingSignIn();
      if (DateTime.now().difference(pending.startedAt) >
          const Duration(minutes: 10)) {
        ref.read(pendingAuthProvider.notifier).clear();
        throw const SignInTimedOut();
      }

      // Pre-resolve the device-id so the handoff provider stays sync.
      // The server requires X-Craftsky-Device-Id on every authenticated
      // call; the handoff Dio bakes it into BaseOptions alongside the
      // bearer token.
      final deviceId = await ref.read(deviceIdProvider.future);
      if (!ref.mounted) return;

      // One-shot client; the token + deviceId are in its
      // BaseOptions.headers. No global provider state, no need to clear
      // anything on exit beyond pending-auth.
      final handoff = ref.read(handoffApiClientProvider(token, deviceId));
      try {
        final who = await handoff.whoami();
        if (!ref.mounted) return;

        final storage = ref.read(secureTokenStorageProvider);
        try {
          await storage.write(
            StoredSession(token: token, did: who.did, handle: who.handle),
          );
        } on Object catch (e, st) {
          _log.warning('SecureTokenStorage.write failed', e, st);
          throw StorageFailure(e);
        }
        if (!ref.mounted) return;

        ref
            .read(authSessionProvider.notifier)
            .setSignedIn(
              SignedIn(did: who.did, handle: who.handle, token: token),
            );
      } finally {
        if (ref.mounted) {
          ref.read(pendingAuthProvider.notifier).clear();
        }
      }
    });
  }

  Future<void> signOut() async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      try {
        await ref.read(authApiClientProvider).logout();
      } on ApiException catch (e, st) {
        _log.warning('logout network/server error; clearing locally', e, st);
      }
      if (!ref.mounted) return;
      await ref.read(secureTokenStorageProvider).clear();
      if (!ref.mounted) return;
      ref.read(authSessionProvider.notifier).setSignedOut();
    });
  }
}
