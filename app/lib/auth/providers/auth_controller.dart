import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/auth_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/handoff_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:url_launcher/url_launcher.dart' as url_launcher;

part 'auth_controller.g.dart';

class SignOutResult {
  const SignOutResult._({this.activeHandle});

  const SignOutResult.signedOut() : this._();

  const SignOutResult.switchedTo(String handle) : this._(activeHandle: handle);

  final String? activeHandle;

  @override
  String toString() => 'SignOutResult(<redacted>)';
}

final _log = Logger('AuthController');

/// The URL-launch function. Overridable in tests so we don't trigger
/// the real system browser.
typedef AuthUrlLauncher = Future<bool> Function(Uri uri);

final authUrlLauncherProvider = Provider<AuthUrlLauncher>(
  (ref) =>
      (uri) => url_launcher.launchUrl(
        uri,
        mode: url_launcher.LaunchMode.externalApplication,
      ),
);

/// Sign-in / sign-out orchestrator.
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

      final launched = await ref.read(authUrlLauncherProvider)(
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
      final handoff = ref.read(
        handoffApiClientProvider(
          HandoffClientKey(token: token, deviceId: deviceId),
        ),
      );
      try {
        final who = await handoff.whoami();
        if (!ref.mounted) return;

        try {
          final current = await ref.read(sessionRegistryProvider.future);
          await ref
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: token,
                did: who.did,
                handle: who.handle,
                beforePublish: current.sessions.isEmpty
                    ? null
                    : ref.read(accountStateInvalidatorProvider),
              );
        } on SessionRegistryStorageException catch (error) {
          _log.warning('session registry write failed');
          throw StorageFailure(error);
        } on AccountLimitReached {
          rethrow;
        } on Object catch (error) {
          _log.warning('session registry mutation failed');
          throw StorageFailure(error);
        }
        if (!ref.mounted) return;
      } finally {
        if (ref.mounted) {
          ref.read(pendingAuthProvider.notifier).clear();
        }
      }
    });
  }

  Future<SignOutResult?> signOut() async {
    SignOutResult? result;
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final registry = await ref.read(sessionRegistryProvider.future);
      final lease = registry.activeLease?.session;
      if (lease == null) return;
      try {
        final api = await ref.read(
          accountAuthApiClientProvider(lease.account).future,
        );
        await api.logout();
      } on ApiUnauthorized {
        // The server has already made this credential unusable, which is an
        // authoritative confirmation that local removal is safe.
      } on ApiException catch (error, stackTrace) {
        _log.warning(
          'logout was not confirmed; retaining the account for retry',
          error,
          stackTrace,
        );
        rethrow;
      }
      await ref.read(accountStateInvalidatorProvider)();
      await ref.read(accountSessionPrivateStateCleanerProvider)(lease);
      await ref.read(sessionRegistryProvider.notifier).removeConfirmed(lease);
      final next = ref.read(sessionRegistryProvider).requireValue;
      final activeDid = next.activeDid;
      if (activeDid == null) {
        result = const SignOutResult.signedOut();
      } else {
        result = SignOutResult.switchedTo(
          next.sessions[activeDid]!.handle.value,
        );
        await ref.read(accountHomeResetProvider)();
      }
    });
    return state.hasError ? null : result;
  }
}
