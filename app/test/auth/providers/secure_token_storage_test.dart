import 'dart:convert';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

class _InterruptingBackend implements SessionRegistryStorageBackend {
  final values = <String, String>{};
  bool corruptNextWrite = false;

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    values[key] = corruptNextWrite
        ? value.substring(0, value.length ~/ 2)
        : value;
    corruptNextWrite = false;
  }
}

void main() {
  setUpAll(initializeMappers);

  setUp(() {
    FlutterSecureStorage.setMockInitialValues({});
  });

  test('read returns null when storage is empty', () async {
    final storage = SecureTokenStorage(const FlutterSecureStorage());
    expect(await storage.read(), isNull);
  });

  test('write then read round-trips a session', () async {
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    await storage.write(
      StoredSession(
        token: 'tok',
        did: 'did:plc:a',
        handle: 'a.bsky.social',
      ),
    );

    final session = await storage.read();
    expect(session, isNotNull);
    expect(session!.token, 'tok');
    expect(session.did, 'did:plc:a');
  });

  test('clear removes the stored session', () async {
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    await storage.write(
      StoredSession(token: 't', did: 'did:plc:test', handle: 'h.test'),
    );
    await storage.clear();

    expect(await storage.read(), isNull);
  });

  test('read returns null on corrupt blob and logs a warning', () async {
    FlutterSecureStorage.setMockInitialValues(
      {'craftsky_session': 'not-valid-json'},
    );
    final storage = SecureTokenStorage(const FlutterSecureStorage());

    expect(await storage.read(), isNull);
  });

  test(
    'read gives back well-formed JSON that matches the blob shape',
    () async {
      FlutterSecureStorage.setMockInitialValues({
        'craftsky_session': jsonEncode(
          {'token': 't', 'did': 'did:plc:test', 'handle': 'h.test'},
        ),
      });
      final storage = SecureTokenStorage(const FlutterSecureStorage());

      final session = await storage.read();
      expect(session?.token, 't');
    },
  );

  test(
    'alternates complete registry snapshots and restores the newest',
    () async {
      const secureStorage = FlutterSecureStorage();
      final storage = SecureSessionRegistryStorage(secureStorage);
      final first = SessionRegistry.empty().upsertAndActivate(
        token: 'token-alice',
        did: 'did:plc:alice',
        handle: 'alice.test',
      );
      final second = first.upsertAndActivate(
        token: 'token-bob',
        did: 'did:plc:bob',
        handle: 'bob.test',
      );

      await storage.write(first);
      await storage.write(second);

      final restored = await storage.read();
      expect(restored.revision, second.revision);
      expect(restored.activeDid, 'did:plc:bob');
      expect(restored.sessions.keys, {'did:plc:alice', 'did:plc:bob'});
      expect(
        await secureStorage.read(key: SecureSessionRegistryStorage.slotAKey),
        isNotNull,
      );
      expect(
        await secureStorage.read(key: SecureSessionRegistryStorage.slotBKey),
        isNotNull,
      );
    },
  );

  test('keeps the previous winner when target read-back is corrupt', () async {
    final backend = _InterruptingBackend();
    final storage = SecureSessionRegistryStorage.withBackend(backend);
    final first = SessionRegistry.empty().upsertAndActivate(
      token: 'token-alice',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );
    final second = first.upsertAndActivate(
      token: 'token-bob',
      did: 'did:plc:bob',
      handle: 'bob.test',
    );
    await storage.write(first);

    backend.corruptNextWrite = true;
    await expectLater(
      storage.write(second),
      throwsA(isA<SessionRegistryStorageException>()),
    );

    final restored = await storage.read();
    expect(restored.revision, first.revision);
    expect(restored.sessions.keys, {'did:plc:alice'});
    expect(restored.activeDid, 'did:plc:alice');
  });
}
