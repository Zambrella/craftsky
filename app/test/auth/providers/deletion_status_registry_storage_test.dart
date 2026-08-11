import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_storage.dart';
import 'package:flutter_test/flutter_test.dart';

class _MemoryBackend implements DeletionStatusStorageBackend {
  final values = <String, String>{};
  bool failWrites = false;

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    if (failWrites) throw StateError('write failed');
    values[key] = value;
  }
}

void main() {
  test('uses a separate fail-closed secure snapshot', () async {
    final backend = _MemoryBackend();
    final storage = SecureDeletionStatusRegistryStorage.withBackend(backend);
    final registry = DeletionStatusRegistry.empty().upsert(
      DeletionStatusEntry.pending(
        jobId: '10000000-0000-0000-0000-000000000001',
        did: 'did:plc:alice',
        handle: 'alice.test',
        statusToken: 'status-token',
      ),
    );

    await storage.write(registry);

    expect(backend.values.keys, [
      SecureDeletionStatusRegistryStorage.storageKey,
    ]);
    expect((await storage.read()).toJson(), registry.toJson());
    expect(
      SecureDeletionStatusRegistryStorage.storageKey,
      isNot('craftsky_session_registry'),
    );

    backend.values[SecureDeletionStatusRegistryStorage.storageKey] = 'bad';
    expect((await storage.read()).entries, isEmpty);

    backend.failWrites = true;
    await expectLater(
      storage.write(registry),
      throwsA(isA<DeletionStatusStorageException>()),
    );
  });
}
