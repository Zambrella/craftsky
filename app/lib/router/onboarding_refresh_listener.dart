import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Attaches a per-DID `ref.listen` on `onboardingStatusProvider(did)`
/// so the router's `refreshListenable` fires when onboarding flips.
/// Call `update` whenever the `authSessionProvider` transitions; it
/// closes the previous subscription (if any) and reattaches to the
/// new DID, or tears down on sign-out.
///
/// Caller owns the lifecycle: invoke `close` from `ref.onDispose`.
class OnboardingRefreshListener {
  OnboardingRefreshListener({required this.ref, required this.onChange});

  final Ref ref;
  final VoidCallback onChange;

  ProviderSubscription<bool>? _sub;
  Did? _currentDid;

  void update(AuthState? auth) {
    final newDid = switch (auth) {
      SignedIn(:final did) => did,
      _ => null,
    };
    if (newDid == _currentDid) return;

    _sub?.close();
    _sub = null;
    _currentDid = newDid;

    if (newDid != null) {
      _sub = ref.listen<bool>(
        onboardingStatusProvider(newDid),
        (_, _) => onChange(),
      );
    }
  }

  void close() {
    _sub?.close();
    _sub = null;
  }
}
