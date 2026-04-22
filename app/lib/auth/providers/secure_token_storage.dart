import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'secure_token_storage.g.dart';

final _log = Logger('SecureTokenStorage');

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
