import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/services/notification_routing_storage.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final alice = Did.parse('did:plc:alice');
  final bob = Did.parse('did:plc:bob');
  final aliceBinding = AccountSubscriptionId.parse('alice_binding');
  final replacement = AccountSubscriptionId.parse('alice_replacement');
  final bobBinding = AccountSubscriptionId.parse('bob_binding');

  group('UT-015 DID-keyed routing storage', () {
    test('replaces one DID and removes it without touching another', () async {
      final backend = _MemoryBackend();
      final storage = NotificationRoutingStorage(backend);

      await storage.replace(alice, aliceBinding);
      await storage.replace(bob, bobBinding);
      await storage.replace(alice, replacement);

      expect(await storage.read(alice), replacement);
      expect(await storage.read(bob), bobBinding);

      await storage.remove(alice);

      expect(await storage.read(alice), isNull);
      expect(await storage.read(bob), bobBinding);
    });

    test('clears corrupt data safely', () async {
      final backend = _MemoryBackend()..value = '{not json';
      final storage = NotificationRoutingStorage(backend);

      expect(await storage.read(alice), isNull);
      expect(backend.value, isNull);
    });
  });
}

final class _MemoryBackend implements NotificationRoutingStorageBackend {
  String? value;

  @override
  Future<void> delete() async => value = null;

  @override
  Future<String?> read() async => value;

  @override
  Future<void> write(String value) async => this.value = value;
}
