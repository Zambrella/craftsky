import 'dart:async';

import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_session_provider.g.dart';

typedef PendingHandoffRecoveryLauncher = Future<void> Function();

final pendingHandoffRecoveryLauncherProvider =
    Provider<PendingHandoffRecoveryLauncher>(
      (ref) => ref.read(authControllerProvider.notifier).resumePendingHandoff,
    );

/// Token-free UI/router projection of the durable session registry.
@Riverpod(keepAlive: true)
class AuthSession extends _$AuthSession {
  AccountSessionLease? _lastValidationLease;
  String? _lastRecoveryReceiptId;

  @override
  Future<AuthState> build() async {
    await ref.read(sessionRegistryProvider.future);
    if (!ref.mounted) throw AccountOperationCanceled();
    final target = ref
        .watch(
          sessionRegistryProvider.select(
            (state) => state.whenData((registry) {
              final lease = registry.activeLease?.session;
              return (
                lease: lease,
                handle: registry.sessions[lease?.account.did]?.handle.value,
                pending: registry.pendingHandoff,
              );
            }),
          ),
        )
        .requireValue;
    final pending = target.pending;
    if (pending == null) {
      _lastRecoveryReceiptId = null;
    } else if (_lastRecoveryReceiptId != pending.receiptId) {
      _lastRecoveryReceiptId = pending.receiptId;
      unawaited(ref.read(pendingHandoffRecoveryLauncherProvider)());
    }
    final activeLease = target.lease;
    if (activeLease == null) return const SignedOut();
    if (_lastValidationLease != activeLease) {
      _lastValidationLease = activeLease;
      unawaited(ref.read(sessionValidationLauncherProvider)(activeLease));
    }
    return SignedIn(
      did: activeLease.account.did.toString(),
      handle: target.handle!,
    );
  }
}
