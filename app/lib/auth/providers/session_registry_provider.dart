import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/pending_session_cleanup.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
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
    Future<void> Function()? beforePublish,
  }) {
    final operation = _pendingMutation.then((_) async {
      final current = state.requireValue;
      final next = current.upsertAndActivate(
        token: token,
        did: did,
        handle: handle,
        cachedDisplayName: cachedDisplayName,
        cachedAvatarUrl: cachedAvatarUrl,
      );
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

  Future<void> invalidate(AccountSessionLease lease) {
    return removeConfirmed(lease);
  }

  Future<void> removeConfirmed(AccountSessionLease lease) {
    final operation = _pendingMutation.then((_) async {
      final current = state.requireValue;
      final stored = current.sessions[lease.account.did];
      if (stored == null ||
          stored.sessionGeneration != lease.sessionGeneration) {
        return;
      }
      final next = current.remove(lease.account.did.value);
      await ref.read(secureSessionRegistryStorageProvider).write(next);
      if (!ref.mounted) return;
      state = AsyncData(next);
    });
    _pendingMutation = operation.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    return operation;
  }

  Future<void> activate(AccountSessionLease lease) {
    final operation = _pendingMutation.then((_) async {
      final current = state.requireValue;
      final next = current.activate(lease);
      if (identical(next, current)) return;
      await ref.read(secureSessionRegistryStorageProvider).write(next);
      if (!ref.mounted) return;
      state = AsyncData(next);
    });
    _pendingMutation = operation.then<void>(
      (_) {},
      onError: (Object _, StackTrace _) {},
    );
    return operation;
  }

  Future<void> saveRoutingBinding(
    AccountSessionLease lease,
    AccountSubscriptionId binding,
  ) => _mutate(
    (current) => current.saveRoutingBinding(lease, binding.wireValue),
  );

  Future<void> removeRoutingBinding(AccountSessionLease lease) =>
      _mutate((current) => current.removeRoutingBinding(lease));

  Future<void> quarantineAndRemove(AccountSessionLease lease) =>
      _mutate((current) => current.quarantineAndRemove(lease));

  Future<void> deletePendingCleanup(PendingSessionCleanup cleanup) =>
      _mutate((current) => current.removePendingCleanup(cleanup));

  Future<void> updateCachedIdentity(
    AccountSessionLease lease, {
    required String? displayName,
    required String? avatarUrl,
  }) => _mutate(
    (current) => current.updateCachedIdentity(
      lease,
      displayName: displayName,
      avatarUrl: avatarUrl,
    ),
  );

  Future<void> _mutate(
    registry.SessionRegistry Function(registry.SessionRegistry current)
    transform,
  ) {
    final operation = _pendingMutation.then((_) async {
      final current = state.requireValue;
      final next = transform(current);
      if (identical(next, current)) return;
      await ref.read(secureSessionRegistryStorageProvider).write(next);
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
