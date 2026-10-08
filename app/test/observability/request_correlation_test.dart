import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/diagnostic_evidence.dart';
import '../test_support/serialized_sentry_transport.dart';

class CorrelationHandler extends ErrorInterceptorHandler {
  ApiException? error;
  @override
  void next(DioException err) {
    error = err.error! as ApiException;
  }
}

void main() {
  final fixturePath = Platform.environment['CRAFTSKY_CORRELATION_FIXTURE'];
  test(
    'IT-004 actual Go envelope links Flutter local Logs and issue',
    () async {
      final fixture =
          jsonDecode(await File(fixturePath!).readAsString())
              as Map<String, dynamic>;
      final envelope = fixture['envelope'] as Map<String, dynamic>;
      final id = envelope['requestId'] as String;
      expect(fixture['local'], contains(id));
      expect(jsonEncode(fixture['events']), contains(id));
      final req = RequestOptions(
        path: '/v1/posts/did:plc:target/record',
        method: 'GET',
      );
      final handler = CorrelationHandler();
      const ErrorMappingInterceptor().onError(
        DioException(
          requestOptions: req,
          type: DioExceptionType.badResponse,
          response: Response(
            requestOptions: req,
            statusCode: 500,
            data: envelope,
          ),
        ),
        handler,
      );
      final error = handler.error!;
      expect(error.details.requestId, id);
      final mapped = AppErrorMapper.map(error);
      expect(mapped.safeDiagnostics['appViewRequestId'], id);
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport
          ..enableLogs = true;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final local = <String>[];
      final subscription = configureRootLogForwarding(
        platformSink: local.add,
      );
      addTearDown(subscription.cancel);
      final context = ReportContext(
        feature: 'Post',
        operation: 'read',
        classification: mapped.sentryClassification,
        safeDiagnostics: mapped.safeDiagnostics,
      );
      Logger('Post').severe(
        DiagnosticMessage(
          'Operation failed',
          context: context,
        ),
        error,
        StackTrace.fromString(
          '#0 loadPublicRecord (package:craftsky_app/post.dart:12:3)',
        ),
      );
      await const SentryErrorReporter().captureException(
        error,
        context: context,
      );
      await Future<void>.delayed(Duration.zero);
      await Sentry.close();
      expect(local.join(), contains(id));
      final payloads = transport.payloads
          .map(jsonDecode)
          .cast<Map<String, dynamic>>()
          .toList();
      final logs = payloads
          .expand((payload) => (payload['items'] as List?) ?? const [])
          .cast<Map<String, dynamic>>()
          .toList();
      expect(logs.length, 1);
      final attributes = logs.single['attributes'] as Map<String, dynamic>;
      final requestId = attributes['appViewRequestId'] as Map<String, dynamic>;
      expect(requestId['value'], id);
      final events = payloads
          .where((payload) => payload.containsKey('exception'))
          .toList();
      expect(events.length, 1);
      final contexts = events.single['contexts'] as Map<String, dynamic>;
      final correlation = contexts['operation'] as Map<String, dynamic>;
      expect(correlation['appViewRequestId'], id);
      writeDiagnosticEvidence(
        'paired-correlation-flutter',
        local: local,
        exported: transport.payloads,
      );
    },
    skip: fixturePath == null
        ? 'Generate paired fixture with '
              'TestLoggingErrorReportingRequestCorrelation and '
              'CRAFTSKY_CORRELATION_FIXTURE.'
        : false,
  );
}
