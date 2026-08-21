import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/pending_handoff.dart';
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
  String? _confirmingReceiptId;
  Future<void>? _confirmationOperation;

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

  Future<void> completeFromDeepLink(String code) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final registry = await ref.read(sessionRegistryProvider.future);
      final storedHandoff = registry.pendingHandoff;
      if (storedHandoff != null) {
        await _confirmStoredHandoff(storedHandoff);
        if (ref.mounted) ref.read(pendingAuthProvider.notifier).clear();
        return;
      }

      // Resolve the stable installation ID before redemption. The anonymous
      // handoff Dio attaches the same value to exchange and confirmation.
      await ref.read(deviceIdProvider.future);
      if (!ref.mounted) return;

      final api = ref.read(handoffApiClientProvider);
      final PendingHandoff handoff;
      try {
        handoff = await api.exchange(code: code);
      } on ApiException catch (error) {
        switch (error) {
          case ApiBadRequest(code: 'invalid_handoff'):
            ref.read(pendingAuthProvider.notifier).clear();
            throw const SignInTimedOut();
          case ApiNetworkError() || ApiServerError() || ApiCanceled():
            throw const ServerUnavailable();
          default:
            rethrow;
        }
      }

      if (!ref.mounted) return;
      try {
        await ref.read(sessionRegistryProvider.notifier).stageHandoff(handoff);
      } on SessionRegistryStorageException catch (error) {
        _log.warning('pending handoff storage failed');
        throw StorageFailure(error);
      } on AccountLimitReached {
        rethrow;
      } on Object catch (error) {
        _log.warning('pending handoff mutation failed');
        throw StorageFailure(error);
      }
      if (!ref.mounted) return;
      ref.read(pendingAuthProvider.notifier).clear();
      await _confirmStoredHandoff(handoff);
    });
  }

  /// Retries confirmation of an already durable receipt. A lost confirmation
  /// response or process restart therefore cannot strand an active server
  /// session outside the local registry.
  Future<void> resumePendingHandoff() async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final registry = await ref.read(sessionRegistryProvider.future);
      final pending = registry.pendingHandoff;
      if (pending == null) return;
      await ref.read(deviceIdProvider.future);
      if (!ref.mounted) return;
      await _confirmStoredHandoff(pending);
    });
  }

  Future<void> _confirmStoredHandoff(PendingHandoff handoff) {
    final inFlight = _confirmationOperation;
    if (inFlight != null && _confirmingReceiptId == handoff.receiptId) {
      return inFlight;
    }

    final operation = _runHandoffConfirmation(handoff);
    _confirmingReceiptId = handoff.receiptId;
    _confirmationOperation = operation;
    unawaited(
      operation.then<void>(
        (_) => _clearConfirmationOperation(operation),
        onError: (Object _, StackTrace _) =>
            _clearConfirmationOperation(operation),
      ),
    );
    return operation;
  }

  void _clearConfirmationOperation(Future<void> operation) {
    if (!identical(_confirmationOperation, operation)) return;
    _confirmationOperation = null;
    _confirmingReceiptId = null;
  }

  Future<void> _runHandoffConfirmation(PendingHandoff handoff) async {
    final api = ref.read(handoffApiClientProvider);
    try {
      await api.confirm(token: handoff.token, receiptId: handoff.receiptId);
    } on ApiException catch (error) {
      switch (error) {
        case ApiBadRequest(code: 'invalid_handoff') || ApiUnauthorized():
          await ref
              .read(sessionRegistryProvider.notifier)
              .discardHandoff(handoff.receiptId);
          throw const SignInTimedOut();
        case ApiNetworkError() || ApiServerError() || ApiCanceled():
          throw const ServerUnavailable();
        default:
          rethrow;
      }
    }
    if (!ref.mounted) return;

    try {
      final current = ref.read(sessionRegistryProvider).requireValue;
      await ref
          .read(sessionRegistryProvider.notifier)
          .confirmHandoff(
            handoff.receiptId,
            beforePublish: current.sessions.isEmpty
                ? null
                : ref.read(accountStateInvalidatorProvider),
          );
    } on SessionRegistryStorageException catch (error) {
      _log.warning('confirmed handoff storage failed');
      throw StorageFailure(error);
    } on AccountLimitReached {
      rethrow;
    } on Object catch (error) {
      _log.warning('confirmed handoff mutation failed');
      throw StorageFailure(error);
    }
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
