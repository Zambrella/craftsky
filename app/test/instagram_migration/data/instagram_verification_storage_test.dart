import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_verification_storage.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-022 secure verification snapshots are isolated by account',
    () async {
      final backend = _MemoryBackend();
      final storage = SecureInstagramVerificationStorage.withBackend(backend);
      final alice = AccountKey('did:plc:alice');
      final bob = AccountKey('did:plc:bob');
      final aliceSnapshot = InstagramVerificationSnapshot(
        verificationId: 'verification-alice',
        challenge: 'CSKY-ALICE-PRIVATE',
        dmUrl: Uri.parse('https://instagram.example/alice'),
        expiresAt: DateTime.utc(2026, 7, 22, 16, 10),
      );
      final bobSnapshot = InstagramVerificationSnapshot(
        verificationId: 'verification-bob',
        challenge: 'CSKY-BOB-PRIVATE',
        dmUrl: Uri.parse('https://instagram.example/bob'),
        expiresAt: DateTime.utc(2026, 7, 22, 16, 11),
      );

      await storage.write(alice, aliceSnapshot);
      await storage.write(bob, bobSnapshot);

      expect((await storage.read(alice))?.verificationId, 'verification-alice');
      expect((await storage.read(alice))?.challenge, 'CSKY-ALICE-PRIVATE');
      expect((await storage.read(bob))?.verificationId, 'verification-bob');
      expect(aliceSnapshot.toString(), isNot(contains('verification-alice')));
      expect(aliceSnapshot.toString(), isNot(contains('CSKY-ALICE-PRIVATE')));

      await storage.delete(alice, verificationId: 'different-verification');
      expect((await storage.read(alice))?.verificationId, 'verification-alice');
      await storage.delete(alice, verificationId: 'verification-alice');
      expect(await storage.read(alice), isNull);
      expect((await storage.read(bob))?.verificationId, 'verification-bob');
    },
  );

  test('IT-022 malformed snapshots fail closed and are removed', () async {
    final backend = _MemoryBackend();
    final storage = SecureInstagramVerificationStorage.withBackend(backend);
    final account = AccountKey('did:plc:alice');
    await storage.write(
      account,
      InstagramVerificationSnapshot(
        verificationId: 'verification-alice',
        challenge: 'CSKY-ALICE-PRIVATE',
        dmUrl: Uri.parse('https://instagram.example/alice'),
        expiresAt: DateTime.utc(2026, 7, 22, 16, 10),
      ),
    );
    backend.values[backend.values.keys.single] = '{not-json';

    expect(await storage.read(account), isNull);
    expect(backend.values, isEmpty);
  });
}

final class _MemoryBackend implements InstagramVerificationStorageBackend {
  final values = <String, String>{};

  @override
  Future<void> delete(String key) async => values.remove(key);

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;
}
