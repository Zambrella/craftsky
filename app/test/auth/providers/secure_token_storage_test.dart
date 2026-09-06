import 'dart:convert';
import 'dart:io';

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

  test('UT-017 rejects PDS credentials and routing state in storage', () async {
    const forbiddenValues = {
      'pds-access-token-canary',
      'pds-refresh-token-canary',
      'dpop-private-key-canary',
      'https://obsolete-pds.example',
    };
    final backend = _MemoryBackend();
    final storage = SecureSessionRegistryStorage.withBackend(backend);
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'craftsky-session-canary',
      did: 'did:plc:alice',
      handle: 'alice.test',
    );

    await storage.write(registry);

    final stored = backend.values.values.single;
    expect(stored, contains('craftsky-session-canary'));
    for (final forbidden in forbiddenValues) {
      expect(stored, isNot(contains(forbidden)));
    }

    final attempted = jsonDecode(stored) as Map<String, dynamic>;
    final session =
        Map<String, dynamic>.from(
          (attempted['sessions'] as Map<String, dynamic>)['did:plc:alice']
              as Map,
        )..addAll(const {
          'accessToken': 'pds-access-token-canary',
          'refreshToken': 'pds-refresh-token-canary',
          'dpopPrivateKey': 'dpop-private-key-canary',
          'pdsEndpoint': 'https://obsolete-pds.example',
        });
    (attempted['sessions'] as Map<String, dynamic>)['did:plc:alice'] = session;
    backend.values[SecureSessionRegistryStorage.storageKey] = jsonEncode(
      attempted,
    );

    expect((await storage.read()).sessions, isEmpty);

    final cleanStored = jsonDecode(stored) as Map<String, dynamic>;
    attempted
      ..['sessions'] = cleanStored['sessions']
      ..['pdsEndpoint'] = 'https://obsolete-pds.example';
    backend.values[SecureSessionRegistryStorage.storageKey] = jsonEncode(
      attempted,
    );

    expect((await storage.read()).sessions, isEmpty);
  });

  test('UT-017 Flutter auth models contain no PDS credential state', () {
    final violations = <String>[];
    final forbidden = RegExp(
      r'\b(?:accessToken|refreshToken|dpopPrivateKey|dpopProof|pdsEndpoint|pdsHost|pdsUrl|serviceEndpoint|authorizationServer|issuer)\b',
      caseSensitive: false,
    );

    for (final entity in Directory('lib/auth/models').listSync()) {
      if (entity is! File || !entity.path.endsWith('.dart')) continue;
      if (forbidden.hasMatch(entity.readAsStringSync())) {
        violations.add(entity.path.replaceAll(Platform.pathSeparator, '/'));
      }
    }

    expect(
      violations,
      isEmpty,
      reason:
          'Flutter auth state may hold only opaque Craftsky session tokens; '
          'PDS/OAuth/DPoP authority remains in AppView.',
    );
  });
}
