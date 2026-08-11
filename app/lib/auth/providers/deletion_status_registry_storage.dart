import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';

final _log = Logger('DeletionStatusRegistryStorage');

final class DeletionStatusStorageException implements Exception {
  const DeletionStatusStorageException();

  @override
  String toString() => 'DeletionStatusStorageException(writeFailed)';
}

abstract interface class DeletionStatusStorageBackend {
  Future<String?> read(String key);

  Future<void> write(String key, String value);
}

abstract interface class DeletionStatusRegistryStorage {
  Future<DeletionStatusRegistry> read();

  Future<void> write(DeletionStatusRegistry registry);
}

final class _FlutterDeletionStatusStorageBackend
    implements DeletionStatusStorageBackend {
  const _FlutterDeletionStatusStorageBackend(this._storage);

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);
}

/// Secure status-only state, deliberately isolated from ordinary bearer
/// sessions so accepting deletion can remove all product authority while the
/// device still observes and recovers the deletion job.
final class SecureDeletionStatusRegistryStorage
    implements DeletionStatusRegistryStorage {
  SecureDeletionStatusRegistryStorage(FlutterSecureStorage storage)
    : _backend = _FlutterDeletionStatusStorageBackend(storage);

  SecureDeletionStatusRegistryStorage.withBackend(this._backend);

  static const storageKey = 'craftsky_account_deletion_status_registry_v1';

  final DeletionStatusStorageBackend _backend;

  @override
  Future<DeletionStatusRegistry> read() async {
    try {
      final source = await _backend.read(storageKey);
      return source == null
          ? DeletionStatusRegistry.empty()
          : DeletionStatusRegistry.fromJson(source);
    } on Object catch (error, stackTrace) {
      _log.warning(
        'deletion status snapshot unavailable; treating as empty',
        error,
        stackTrace,
      );
      return DeletionStatusRegistry.empty();
    }
  }

  @override
  Future<void> write(DeletionStatusRegistry registry) async {
    try {
      await _backend.write(storageKey, registry.toJson());
    } on Object {
      throw const DeletionStatusStorageException();
    }
  }
}
