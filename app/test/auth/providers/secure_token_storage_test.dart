import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/shared/observability/diagnostic_details.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

class _MemoryBackend implements SessionRegistryStorageBackend {
  final values = <String, String>{};
  bool failWrites = false;
  Object? writeCause;

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    if (writeCause case final Exception cause) throw cause;
    if (writeCause case final Error cause) throw cause;
    if (failWrites) throw StateError('write failed');
    values[key] = value;
  }
}

class _StorageReporter implements ErrorReporter {
  final errors = <Object>[];
  @override
  bool get enabled => true;
  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) {}
  @override
  Future<void> emitLog(
    String message, {
    required ReportContext context,
  }) async {}
  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {}
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    return null;
  }
}

void main() {
  test(
    'SIM-T02 consumed registry read failure has an explicit capture owner',
    () async {
      final backend = _MemoryBackend()
        ..values[SecureSessionRegistryStorage.storageKey] =
            'PRIVATE_BROKEN_SNAPSHOT';
      final reporter = _StorageReporter();
      final storage = SecureSessionRegistryStorage.withBackend(
        backend,
        reporter: reporter,
      );
      expect((await storage.read()).sessions, isEmpty);
      await Future<void>.delayed(Duration.zero);
      expect(reporter.errors, hasLength(1));
      expect(reporter.errors.single, isA<FormatException>());
    },
  );

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

  test(
    'IT-010 storage write retains original cause without snapshot contents',
    () async {
      final records = <LogRecord>[];
      final sub = Logger.root.onRecord.listen(records.add);
      addTearDown(sub.cancel);
      final cause = StateError('opaque private snapshot token=storage-secret');
      final backend = _MemoryBackend()..writeCause = cause;
      final storage = SecureSessionRegistryStorage.withBackend(backend);
      Object? thrown;
      try {
        await storage.write(SessionRegistry.empty());
      } on Object catch (error) {
        thrown = error;
      }
      expect(thrown, isA<SessionRegistryStorageException>());
      expect(selectedCause(thrown!).toString(), contains('StateError'));
      final failure = records.singleWhere(
        (record) => record.loggerName == 'SecureTokenStorage',
      );
      final selected = selectDiagnosticRecord(failure);
      expect(selected['operation'], 'session.storage.write');
      expect(selected['failureStage'], 'storage_write');
      expect(selected.toString(), contains('StateError'));
      expect(selected.toString(), isNot(contains('storage-secret')));
      expect(selected.toString(), isNot(contains('opaque private snapshot')));
    },
  );

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
