import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:flutter/services.dart';
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

/// Persists complete registry snapshots in alternating secure-storage slots.
/// The previous winner remains untouched until the target slot has been read
/// back through the production decoder and verified.
class SecureSessionRegistryStorage implements SessionRegistryStorage {
  SecureSessionRegistryStorage(FlutterSecureStorage storage)
    : _backend = _FlutterSecureStorageBackend(storage);

  SecureSessionRegistryStorage.withBackend(this._backend);

  static const slotAKey = 'craftsky_session_registry_a';
  static const slotBKey = 'craftsky_session_registry_b';

  final SessionRegistryStorageBackend _backend;

  @override
  Future<SessionRegistry> read() async {
    final slots = await _readSlots();
    return SessionRegistry.recover(slotA: slots.$1, slotB: slots.$2);
  }

  @override
  Future<void> write(SessionRegistry registry) async {
    final slots = await _readSlots();
    final decodedA = _decode(slots.$1);
    final decodedB = _decode(slots.$2);
    final targetKey =
        decodedA != null &&
            (decodedB == null || decodedA.revision >= decodedB.revision)
        ? slotBKey
        : slotAKey;

    try {
      final encoded = registry.toJson();
      await _backend.write(targetKey, encoded);
      final readBack = await _backend.read(targetKey);
      if (readBack == null) {
        throw const SessionRegistryStorageException('missingReadBack');
      }
      final verified = SessionRegistry.fromJson(readBack);
      if (verified.revision != registry.revision ||
          verified.toJson() != encoded) {
        throw const SessionRegistryStorageException('readBackMismatch');
      }
    } on SessionRegistryStorageException {
      rethrow;
    } on Object {
      throw const SessionRegistryStorageException('writeFailed');
    }
  }

  Future<(String?, String?)> _readSlots() async {
    String? slotA;
    String? slotB;
    try {
      slotA = await _backend.read(slotAKey);
    } on Object {
      _log.warning('registry slot read failed: slotA');
    }
    try {
      slotB = await _backend.read(slotBKey);
    } on Object {
      _log.warning('registry slot read failed: slotB');
    }
    return (slotA, slotB);
  }

  SessionRegistry? _decode(String? source) {
    if (source == null) return null;
    try {
      return SessionRegistry.fromJson(source);
    } on Object {
      return null;
    }
  }
}

@Riverpod(keepAlive: true)
SessionRegistryStorage secureSessionRegistryStorage(Ref ref) =>
    SecureSessionRegistryStorage(const FlutterSecureStorage());

/// Thin async wrapper around [FlutterSecureStorage] for the Craftsky
/// session blob. All platform errors are swallowed and logged so the
/// app can always fall back to a `SignedOut` state rather than
/// crashing on startup.
class SecureTokenStorage {
  SecureTokenStorage(this._fss);

  final FlutterSecureStorage _fss;

  static const _key = 'craftsky_session';

  Future<StoredSession?> read() async {
    try {
      final raw = await _fss.read(key: _key);
      if (raw == null) return null;
      return StoredSessionMapper.fromJson(raw);
    } on PlatformException catch (e, st) {
      _log.warning('read failed; treating as unsigned-in', e, st);
      return null;
    } on Object catch (e, st) {
      // FormatException (malformed JSON) or MapperException (missing
      // required fields from dart_mappable) — both mean the blob on
      // disk is garbage we can't use. Delete it so subsequent writes
      // aren't fighting a corrupt value.
      _log.warning('corrupt blob; clearing', e, st);
      try {
        await _fss.delete(key: _key);
      } on Object catch (deleteErr, deleteSt) {
        _log.warning('delete-after-corrupt also failed', deleteErr, deleteSt);
      }
      return null;
    }
  }

  Future<void> write(StoredSession session) =>
      _fss.write(key: _key, value: session.toJson());

  Future<void> clear() => _fss.delete(key: _key);
}

@Riverpod(keepAlive: true)
SecureTokenStorage secureTokenStorage(Ref ref) =>
    SecureTokenStorage(const FlutterSecureStorage());
