import 'dart:async';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'secure_token_storage.g.dart';

final _log = Logger('SecureTokenStorage');

class SessionRegistryStorageException
    implements Exception, DiagnosticFailureCause {
  const SessionRegistryStorageException(
    this.outcome, {
    this.diagnosticCause,
    this.diagnosticStack,
  });
  @override
  final Object? diagnosticCause;
  @override
  final StackTrace? diagnosticStack;

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
  SecureSessionRegistryStorage(
    FlutterSecureStorage storage, {
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _backend = _FlutterSecureStorageBackend(storage),
       _reporter = GuardedErrorReporter(reporter);

  SecureSessionRegistryStorage.withBackend(
    this._backend, {
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _reporter = GuardedErrorReporter(reporter);

  static const storageKey = 'craftsky_session_registry';

  final SessionRegistryStorageBackend _backend;
  final ErrorReporter _reporter;

  @override
  Future<SessionRegistry> read() async {
    try {
      final source = await _backend.read(storageKey);
      if (source == null) return SessionRegistry.empty();
      return SessionRegistry.fromJson(source);
    } on Object catch (error, stackTrace) {
      _log.severe(
        const DiagnosticMessage(
          'registry snapshot unavailable; treating as signed out',
          context: ReportContext(
            feature: 'SecureTokenStorage',
            operation: 'session.storage.read',
            classification: 'storage.unavailable',
            safeDiagnostics: {'failureStage': 'storage_read'},
          ),
        ),
        error,
        stackTrace,
      );
      // This boundary consumes the failure, so it owns issue capture.
      unawaited(
        _reporter.captureException(
          error,
          stackTrace: stackTrace,
          context: const ReportContext(
            feature: 'SecureTokenStorage',
            operation: 'session.storage.read',
            classification: 'storage.unavailable',
            safeDiagnostics: {'failureStage': 'storage_read'},
          ),
        ),
      );
      return SessionRegistry.empty();
    }
  }

  @override
  Future<void> write(SessionRegistry registry) async {
    try {
      await _backend.write(storageKey, registry.toJson());
    } on Object catch (error, stackTrace) {
      _log.severe(
        const DiagnosticMessage(
          'registry snapshot write failed',
          context: ReportContext(
            feature: 'SecureTokenStorage',
            operation: 'session.storage.write',
            classification: 'storage.unavailable',
            safeDiagnostics: {'failureStage': 'storage_write'},
          ),
        ),
        error,
        stackTrace,
      );
      throw SessionRegistryStorageException(
        'writeFailed',
        diagnosticCause: error,
        diagnosticStack: stackTrace,
      );
    }
  }
}

@Riverpod(keepAlive: true)
SessionRegistryStorage secureSessionRegistryStorage(Ref ref) =>
    SecureSessionRegistryStorage(
      const FlutterSecureStorage(),
      reporter: ref.read(errorReporterProvider),
    );
