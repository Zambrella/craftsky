import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_provider.dart';
import 'package:craftsky_app/notifications/services/notification_delivery_dedupe_store.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-PUSH-018 account removal clears its notification partition',
    () async {
      var registry = model.SessionRegistry.empty().upsertAndActivate(
        token: 'token',
        did: 'did:plc:recipient',
        handle: 'recipient.test',
      );
      final lease = registry.activeLease!.session;
      registry = registry.saveRoutingBinding(lease, 'routing-account-one');
      final dedupe = _Dedupe();
      final container = ProviderContainer.test(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(registry),
          ),
          instagramVerificationStorageProvider.overrideWithValue(
            _InstagramStorage(),
          ),
          notificationDeliveryDedupeStoreProvider.overrideWithValue(dedupe),
        ],
      );
      await container.read(sessionRegistryProvider.future);

      await container.read(accountSessionPrivateStateCleanerProvider)(lease);

      expect(dedupe.cleared, ['routing-account-one']);
    },
  );
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.registry);

  model.SessionRegistry registry;

  @override
  Future<model.SessionRegistry> read() async => registry;

  @override
  Future<void> write(model.SessionRegistry registry) async {
    this.registry = registry;
  }
}

final class _Dedupe implements NotificationDeliveryDedupeStore {
  final cleared = <String>[];

  @override
  Future<bool> claim({
    required String notificationId,
    required String accountPartition,
    required NotificationDeliveryStage stage,
  }) async => true;

  @override
  Future<void> clearAccountPartition(String accountPartition) async {
    cleared.add(accountPartition);
  }
}

final class _InstagramStorage implements InstagramVerificationStorage {
  @override
  Future<void> delete(AccountKey account, {String? verificationId}) async {}

  @override
  Future<InstagramVerificationSnapshot?> read(AccountKey account) async => null;

  @override
  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  ) async {}
}
