import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_handoff.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'session_registry_provider.g.dart';

/// The sole mutable source for retained CraftSky account sessions.
@Riverpod(keepAlive: true)
class SessionRegistry extends _$SessionRegistry {
  Future<void> _pendingMutation = Future.value();

  @override
  Future<registry.SessionRegistry> build() =>
      ref.watch(secureSessionRegistryStorageProvider).read();

  Future<void> upsertAndActivate({
    required String token,
    required String did,
    required String handle,
    String? cachedDisplayName,
    String? cachedAvatarUrl,
    ProfileCustomisation cachedCustomisation = ProfileCustomisation.defaults,
    Future<void> Function()? beforePublish,
  }) => _mutate(
    (current) => current.upsertAndActivate(
      token: token,
      did: did,
      handle: handle,
      cachedDisplayName: cachedDisplayName,
      cachedAvatarUrl: cachedAvatarUrl,
      cachedCustomisation: cachedCustomisation,
    ),
    beforePublish: beforePublish,
  );

  Future<void> stageHandoff(PendingHandoff handoff) =>
      _mutate((current) => current.stageHandoff(handoff));

  Future<void> confirmHandoff(
    String receiptId, {
    Future<void> Function()? beforePublish,
  }) => _mutate(
    (current) => current.confirmHandoff(receiptId),
    beforePublish: beforePublish,
  );

  Future<void> discardHandoff(String receiptId) =>
      _mutate((current) => current.discardHandoff(receiptId));

  Future<void> invalidate(AccountSessionLease lease) {
    return removeConfirmed(lease);
  }

  Future<void> removeConfirmed(AccountSessionLease lease) => _mutate((current) {
    final stored = current.sessions[lease.account.did];
    if (stored == null || stored.sessionGeneration != lease.sessionGeneration) {
      return current;
    }
    return current.remove(lease.account.did.value);
  });

  Future<void> activate(AccountSessionLease lease) =>
      _mutate((current) => current.activate(lease));

  Future<void> saveRoutingBinding(
    AccountSessionLease lease,
    AccountSubscriptionId binding,
  ) => _mutate(
    (current) => current.saveRoutingBinding(lease, binding.wireValue),
  );

  Future<void> removeRoutingBinding(AccountSessionLease lease) =>
      _mutate((current) => current.removeRoutingBinding(lease));

  Future<void> updateCachedIdentity(
    AccountSessionLease lease, {
    required String? displayName,
    required String? avatarUrl,
    required ProfileCustomisation customisation,
  }) => _mutate(
    (current) => current.updateCachedIdentity(
      lease,
      displayName: displayName,
      avatarUrl: avatarUrl,
      customisation: customisation,
    ),
  );

  Future<void> updateCachedCustomisation(
    AccountSessionLease lease,
    ProfileCustomisation customisation,
  ) => _mutate(
    (current) => current.updateCachedCustomisation(lease, customisation),
  );

  Future<void> _mutate(
    registry.SessionRegistry Function(registry.SessionRegistry current)
    transform, {
    Future<void> Function()? beforePublish,
  }) {
    final operation = _pendingMutation.then((_) async {
      final current = state.requireValue;
      final next = transform(current);
      if (identical(next, current)) return;
      await ref.read(secureSessionRegistryStorageProvider).write(next);
      if (!ref.mounted) return;
      await beforePublish?.call();
      if (!ref.mounted) return;
      state = AsyncData(next);
    });
    _pendingMutation = operation.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    return operation;
  }
}
