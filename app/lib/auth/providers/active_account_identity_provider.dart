import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef _ActiveIdentityTarget = ({
  AccountSessionLease lease,
  String handle,
});

final class ActiveAccountIdentity {
  const ActiveAccountIdentity({required this.lease, required this.profile});

  final AccountSessionLease lease;
  final Profile profile;

  @override
  String toString() => 'ActiveAccountIdentity(<redacted>)';
}

/// Loads the active account's own profile as soon as the signed-in shell is
/// mounted and persists its switcher/navigation identity against that lease.
// The builder's concrete auto-dispose provider type is intentionally inferred.
// ignore: specify_nonobvious_property_types
final activeAccountIdentityProvider =
    FutureProvider.autoDispose<ActiveAccountIdentity?>(
      (ref) async {
        final target = ref.watch(
          sessionRegistryProvider.select(_activeIdentityTarget),
        );
        if (target == null) return null;

        final profile = await ref.watch(
          userProfileProvider(target.handle).future,
        );
        final current = ref.read(sessionRegistryProvider).value;
        if (current?.activeLease?.session != target.lease) return null;

        await ref
            .read(sessionRegistryProvider.notifier)
            .updateCachedIdentity(
              target.lease,
              displayName: profile.displayName,
              avatarUrl: profile.avatar,
            );
        final afterUpdate = ref.read(sessionRegistryProvider).value;
        return ref.mounted && afterUpdate?.activeLease?.session == target.lease
            ? ActiveAccountIdentity(lease: target.lease, profile: profile)
            : null;
      },
    );

_ActiveIdentityTarget? _activeIdentityTarget(
  AsyncValue<registry.SessionRegistry> state,
) {
  final value = state.value;
  final lease = value?.activeLease?.session;
  if (value == null || lease == null) return null;
  final session = value.sessions[lease.account.did];
  if (session == null) return null;
  return (lease: lease, handle: session.handle.value);
}
