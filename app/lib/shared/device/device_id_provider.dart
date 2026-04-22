import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:uuid/uuid.dart';

part 'device_id_provider.g.dart';

final _log = Logger('DeviceIdProvider');

/// Secure-storage key for the per-install device identifier. Separate
/// from `craftsky_session` so sign-out does NOT clear the device ID.
const deviceIdStorageKey = 'craftsky_device_id';

/// Injection seam: production uses a shared `FlutterSecureStorage`
/// instance; tests override with a fake.
@Riverpod(keepAlive: true)
FlutterSecureStorage deviceIdSecureStorage(Ref ref) =>
    const FlutterSecureStorage();

/// Returns this install's stable device identifier. On first access,
/// generates a v4 UUID and writes it to secure storage. Subsequent
/// accesses return the persisted value.
///
/// Platform-error tolerant: if secure storage fails, we return a fresh
/// in-memory UUID for this session and attempt to persist it. On a
/// persistence failure, future launches may generate a different ID —
/// acceptable because device-id is correlation data, not a security
/// primitive.
@Riverpod(keepAlive: true)
Future<String> deviceId(Ref ref) async {
  final storage = ref.watch(deviceIdSecureStorageProvider);
  try {
    final existing = await storage.read(key: deviceIdStorageKey);
    if (existing != null && existing.isNotEmpty) return existing;
  } on PlatformException catch (e, st) {
    _log.warning('device-id read failed; will mint a fresh one', e, st);
  }

  final fresh = const Uuid().v4();
  try {
    await storage.write(key: deviceIdStorageKey, value: fresh);
  } on PlatformException catch (e, st) {
    _log.warning('device-id write failed; using in-memory only', e, st);
  }
  return fresh;
}
