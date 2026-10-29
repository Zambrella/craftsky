import 'dart:convert';

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/serialized_sentry_transport.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test(
    'deletion staging and rollback failures retain one protected issue owner',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final local = <String>[];
      final emitter = DiagnosticEmitter(platformSink: local.add);
      final previous = Logger.root.level;
      Logger.root.level = Level.INFO;
      final sub = Logger.root.onRecord.listen(emitter.emitLocal);
      addTearDown(sub.cancel);
      addTearDown(() => Logger.root.level = previous);
      const jobId = '10000000-0000-4000-8000-000000000001';
      const requestId = '20000000-0000-4000-8000-000000000002';
      const error = ApiServerError(
        'PRIVATE_RESPONSE_CANARY',
        details: ApiFailureDetails(
          statusCode: 503,
          method: 'POST',
          requestId: requestId,
          appViewError: 'storage_unavailable',
        ),
      );
      final stack = StackTrace.fromString(
        '#0 saveIntent (package:craftsky_app/storage.dart:42:3)',
      );
      final storage = _FailingStorage(error, stack);
      final reporter = _Reporter();
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.invalid'));
      DioAdapter(dio: dio)
        ..onPost(
          '/v1/account-deletion/intents',
          (server) => server.reply(201, {
            'jobId': jobId,
            'authUrl': 'https://pds.invalid/oauth?state=PRIVATE_OAUTH_CANARY',
            'confirmationDid': 'did:plc:alice',
            'expiresAt': DateTime.now()
                .toUtc()
                .add(const Duration(minutes: 10))
                .toIso8601String(),
          }),
        )
        ..onDelete(
          '/v1/account-deletion/intents/$jobId',
          (server) => server.reply(503, {
            'message': 'PRIVATE_ROLLBACK_CANARY',
          }),
        );
      final container = ProviderContainer.test(
        observers: [ProviderLogger(reporter: reporter)],
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(storage),
          accountDioProvider.overrideWith((ref, account) async {
            return dio;
          }),
        ],
      );
      await container.read(sessionRegistryProvider.future);
      await container.read(accountDeletionControllerProvider.future);
      expect(
        await container
            .read(accountDeletionControllerProvider.notifier)
            .startReauthentication(),
        isNull,
      );
      await transport.issueReceived.future.timeout(
        const Duration(seconds: 5),
      );
      await Future.wait(reporter.pending);
      expect(reporter.errors, [same(error)]);
      expect(reporter.stacks, [same(stack)]);
      final events = transport.payloads
          .map(jsonDecode)
          .whereType<Map<String, dynamic>>()
          .where((event) => event.containsKey('exception'))
          .toList();
      expect(events, hasLength(1));
      for (final output in [local.join('\n'), jsonEncode(events)]) {
        for (final positive in [
          'account.delete',
          'stage_intent',
          'did:plc:alice',
          requestId,
          '503',
          'storage_unavailable',
          'ApiServerError',
          'storage.dart',
          jobId,
        ]) {
          expect(output, contains(positive));
        }
        for (final canary in [
          'PRIVATE_RESPONSE_CANARY',
          'PRIVATE_OAUTH_CANARY',
          'PRIVATE_ROLLBACK_CANARY',
          'PRIVATE_TOKEN_CANARY',
        ]) {
          expect(output, isNot(contains(canary)));
        }
      }
      expect(local.join(), contains('rollback_intent'));
      expect(storage.registry.pendingAccountDeletion, isNull);
    },
  );
}

final class _FailingStorage implements SessionRegistryStorage {
  new(this.error, this.stack);
  final Object error;
  final StackTrace stack;
  final SessionRegistry registry = SessionRegistry.empty().upsertAndActivate(
    token: 'PRIVATE_TOKEN_CANARY',
    did: 'did:plc:alice',
    handle: 'alice.test',
  );
  @override
  Future<SessionRegistry> read() async => registry;
  @override
  Future<void> write(SessionRegistry registry) async =>
      Error.throwWithStackTrace(error, stack);
}

final class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final pending = <Future<String?>>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) {
    errors.add(error);
    stacks.add(stackTrace);
    final result = const SentryErrorReporter().captureException(
      error,
      context: context,
      stackTrace: stackTrace,
    );
    pending.add(result);
    return result;
  }
}
