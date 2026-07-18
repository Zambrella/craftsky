import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:flutter_test/flutter_test.dart';

class _MemoryBackend implements SessionRegistryStorageBackend {
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
  test('SIM-UT-001 uses one fail-closed secure snapshot', () async {
    final backend = _MemoryBackend();
    final storage = SecureSessionRegistryStorage.withBackend(backend);
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token-alice',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );

    await storage.write(registry);

    expect(backend.values.keys, [SecureSessionRegistryStorage.storageKey]);
    expect((await storage.read()).toJson(), registry.toJson());

    backend.values[SecureSessionRegistryStorage.storageKey] = 'not-json';
    expect((await storage.read()).sessions, isEmpty);

    backend.failWrites = true;
    await expectLater(
      storage.write(registry),
      throwsA(isA<SessionRegistryStorageException>()),
    );
  });
}
