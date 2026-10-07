import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

import '../../test_support/recording_error_reporter.dart';

class _FakeSecureStorage implements FlutterSecureStorage {
  _FakeSecureStorage();
  bool failRead = false;
  final Map<String, String> _map = <String, String>{};

  @override
  Future<String?> read({
    required String key,
    AppleOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WindowsOptions? wOptions,
    WebOptions? webOptions,
    AppleOptions? mOptions,
  }) async {
    if (failRead) {
      throw PlatformException(
        code: 'read_error',
        message: 'private device-id canary',
        details: {'token': 'platform-secret'},
      );
    }
    return _map[key];
  }

  @override
  Future<void> write({
    required String key,
    required String? value,
    AppleOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WindowsOptions? wOptions,
    WebOptions? webOptions,
    AppleOptions? mOptions,
  }) async {
    if (value == null) {
      _map.remove(key);
    } else {
      _map[key] = value;
    }
  }

  // Other methods throw — we exercise only read/write in the provider.
  @override
  dynamic noSuchMethod(Invocation inv) => super.noSuchMethod(inv);
}

void main() {
  test(
    'IT-010 device storage failure retains code/stage without device value',
    () async {
      final records = <LogRecord>[];
      final sub = Logger.root.onRecord.listen(records.add);
      addTearDown(sub.cancel);
      final storage = _FakeSecureStorage()..failRead = true;
      final reporter = RecordingErrorReporter();
      final container = ProviderContainer.test(
        overrides: [
          deviceIdSecureStorageProvider.overrideWithValue(storage),
          errorReporterProvider.overrideWithValue(reporter),
        ],
      );
      final id = await container.read(deviceIdProvider.future);
      expect(id, isNotEmpty);
      final selected = selectDiagnosticRecord(
        records.singleWhere(
          (record) => record.loggerName == 'DeviceIdProvider',
        ),
      );
      expect(reporter.errors, hasLength(1));
      expect(selected['operation'], 'device.storage.read');
      expect(selected['failureStage'], 'storage_read');
      expect(selected.toString(), contains('PlatformException'));
      expect(selected.toString(), contains('read_error'));
      expect(selected.toString(), isNot(contains(id)));
      expect(selected.toString(), isNot(contains('private device-id canary')));
      expect(selected.toString(), isNot(contains('platform-secret')));
    },
  );

  test('generates and persists a UUID when storage is empty', () async {
    final storage = _FakeSecureStorage();
    final container = ProviderContainer.test(
      overrides: [deviceIdSecureStorageProvider.overrideWithValue(storage)],
    );

    final id = await container.read(deviceIdProvider.future);
    expect(id, isNotEmpty);
    expect(id.length, greaterThanOrEqualTo(32));
    expect(await storage.read(key: 'craftsky_device_id'), id);
  });

  test('returns the existing ID when storage already has one', () async {
    final storage = _FakeSecureStorage();
    await storage.write(key: 'craftsky_device_id', value: 'pre-existing-id');
    final container = ProviderContainer.test(
      overrides: [deviceIdSecureStorageProvider.overrideWithValue(storage)],
    );

    final id = await container.read(deviceIdProvider.future);
    expect(id, 'pre-existing-id');
  });

  test('two reads return the same ID (keep-alive cached)', () async {
    final storage = _FakeSecureStorage();
    final container = ProviderContainer.test(
      overrides: [deviceIdSecureStorageProvider.overrideWithValue(storage)],
    );

    final first = await container.read(deviceIdProvider.future);
    final second = await container.read(deviceIdProvider.future);
    expect(first, second);
  });
}
