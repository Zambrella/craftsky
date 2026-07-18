import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'secure_token_storage.g.dart';

final _log = Logger('SecureTokenStorage');

class SessionRegistryStorageException implements Exception {
  const SessionRegistryStorageException(this.outcome);

  final String outcome;

  @override
  String toString() => 'SessionRegistryStorageException($outcome)';
}

abstract interface class SessionRegistryStorageBackend {
  Future<String?> read(String key);

  Future<void> write(String key, String value);
}

abstract interface class SessionRegistryStorage {
  Future<SessionRegistry> read();

  Future<void> write(SessionRegistry registry);
}

class _FlutterSecureStorageBackend implements SessionRegistryStorageBackend {
  const _FlutterSecureStorageBackend(this._storage);

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);
}

/// Persists the complete account registry as one fail-closed secure snapshot.
class SecureSessionRegistryStorage implements SessionRegistryStorage {
  SecureSessionRegistryStorage(FlutterSecureStorage storage)
    : _backend = _FlutterSecureStorageBackend(storage);

  SecureSessionRegistryStorage.withBackend(this._backend);

  static const storageKey = 'craftsky_session_registry';

  final SessionRegistryStorageBackend _backend;

  @override
  Future<SessionRegistry> read() async {
    try {
      final source = await _backend.read(storageKey);
      if (source == null) return SessionRegistry.empty();
      return SessionRegistry.fromJson(source);
    } on Object catch (error, stackTrace) {
      _log.warning(
        'registry snapshot unavailable; treating as signed out',
        error,
        stackTrace,
      );
      return SessionRegistry.empty();
    }
  }

  @override
  Future<void> write(SessionRegistry registry) async {
    try {
      await _backend.write(storageKey, registry.toJson());
    } on Object {
      throw const SessionRegistryStorageException('writeFailed');
    }
  }
}

@Riverpod(keepAlive: true)
SessionRegistryStorage secureSessionRegistryStorage(Ref ref) =>
    SecureSessionRegistryStorage(const FlutterSecureStorage());
