import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_session_provider.g.dart';

/// Token-free UI/router projection of the durable session registry.
@Riverpod(keepAlive: true)
class AuthSession extends _$AuthSession {
  AccountSessionLease? _lastValidationLease;

  @override
  Future<AuthState> build() async {
    final registry = await ref.watch(sessionRegistryProvider.future);
    final activeLease = registry.activeLease?.session;
    if (activeLease == null) return const SignedOut();
    final active = registry.sessions[activeLease.account.did]!;
    if (_lastValidationLease != activeLease) {
      _lastValidationLease = activeLease;
      unawaited(ref.read(sessionValidationLauncherProvider)(activeLease));
    }
    return SignedIn(did: active.did.value, handle: active.handle.value);
  }
}
