import 'dart:async';

import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
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
  final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
  try {
    final existing = await storage.read(key: deviceIdStorageKey);
    if (existing != null && existing.isNotEmpty) return existing;
  } on PlatformException catch (e, st) {
    _log.severe(
      const DiagnosticMessage(
        'device-id read failed; will mint a fresh one',
        context: ReportContext(
          feature: 'DeviceIdProvider',
          operation: 'device.storage.read',
          classification: 'storage.unavailable',
          safeDiagnostics: {'failureStage': 'storage_read'},
        ),
      ),
      e,
      st,
    );
    unawaited(
      reporter.captureException(
        e,
        stackTrace: st,
        context: const ReportContext(
          feature: 'DeviceIdProvider',
          operation: 'device.storage.read',
          classification: 'storage.unavailable',
          safeDiagnostics: {'failureStage': 'storage_read'},
        ),
      ),
    );
  }

  final fresh = const Uuid().v4();
  try {
    await storage.write(key: deviceIdStorageKey, value: fresh);
  } on PlatformException catch (e, st) {
    _log.severe(
      const DiagnosticMessage(
        'device-id write failed; using in-memory only',
        context: ReportContext(
          feature: 'DeviceIdProvider',
          operation: 'device.storage.write',
          classification: 'storage.unavailable',
          safeDiagnostics: {'failureStage': 'storage_write'},
        ),
      ),
      e,
      st,
    );
    unawaited(
      reporter.captureException(
        e,
        stackTrace: st,
        context: const ReportContext(
          feature: 'DeviceIdProvider',
          operation: 'device.storage.write',
          classification: 'storage.unavailable',
          safeDiagnostics: {'failureStage': 'storage_write'},
        ),
      ),
    );
  }
  return fresh;
}
