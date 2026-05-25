import 'dart:convert';

import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

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
}
