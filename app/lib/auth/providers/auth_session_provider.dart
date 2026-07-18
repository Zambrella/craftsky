import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_session_provider.g.dart';

/// Token-free UI/router projection of the durable session registry.
@Riverpod(keepAlive: true)
class AuthSession extends _$AuthSession {
  @override
  Future<AuthState> build() async {
    final registry = await ref.watch(sessionRegistryProvider.future);
    final activeDid = registry.activeDid;
    final active = activeDid == null ? null : registry.sessions[activeDid];
    if (active == null) return const SignedOut();
    if (ref.read(sessionValidationLaunchGuardProvider).shouldLaunch(registry)) {
      unawaited(ref.read(sessionValidationLauncherProvider)(registry));
    }
    return SignedIn(did: active.did.value, handle: active.handle.value);
  }

  /// Temporary compatibility for callers migrated in the sign-out loop.
  void setSignedOut() => state = const AsyncData(SignedOut());
}
