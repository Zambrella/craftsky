import 'dart:convert';

import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_details.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/serialized_sentry_transport.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test(
    'SDK keeps sampled native thread evidence and removes protected data',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      await Sentry.captureEvent(
        SentryEvent(
          threads: [
            SentryThread(
              id: 1,
              current: true,
              name: 'PRIVATE_THREAD_NAME',
              unknown: {'payload': 'PRIVATE_THREAD_PAYLOAD'},
              stacktrace: SentryStackTrace(
                unknown: {'private': 'PRIVATE_STACK'},
                registers: {'x0': 'PRIVATE_REGISTER'},
                frames: [
                  SentryStackFrame(
                    function: 'decodeImage',
                    fileName: 'native.cc',
                    absPath: '/private/local/source.cc',
                    instructionAddr: '0x1234',
                    vars: {'image': 'PRIVATE_IMAGE'},
                    contextLine: 'PRIVATE_SOURCE',
                  ),
                ],
              ),
            ),
            SentryThread(
              id: 2,
              name: 'PRIVATE_OTHER_THREAD',
              stacktrace: SentryStackTrace(frames: []),
            ),
          ],
          exceptions: [
            SentryException(type: 'NativeHang', value: 'PRIVATE_MESSAGE'),
          ],
        ),
      );
      await Sentry.close();
      final event = transport.payloads
          .map(jsonDecode)
          .cast<Map<String, dynamic>>()
          .firstWhere((p) => p.containsKey('exception'));
      final operation = (event['contexts'] as Map)['operation'] as Map;
      expect(operation['schedulerPhase'], 'idle');
      expect(operation['semanticsEnabled'], isA<bool>());
      expect(operation['accessibilityAnimationsDisabled'], isA<bool>());
      final threads = (event['threads'] as Map)['values'] as List;
      expect(threads, hasLength(1));
      expect(threads.single.toString(), contains('0x1234'));
      expect(threads.single.toString(), contains('decodeImage'));
      expect(transport.payloads.join(), isNot(contains('PRIVATE_')));
      expect(transport.payloads.join(), isNot(contains('/private/local')));
    },
  );

  test(
    'provider causes and picker codes survive serialized filtering',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final failed = Provider<int>(
        (ref) => throw PlatformException(
          code: 'invalid_image',
          message: 'PRIVATE_IMAGE',
          details: {'payload': 'PRIVATE_BYTES'},
        ),
      );
      final dependency = Provider<int>((ref) => ref.watch(failed));
      final container = ProviderContainer.test();
      try {
        container.read(dependency);
      } on Object catch (error, stack) {
        await Sentry.captureException(error, stackTrace: stack);
      }
      await Sentry.close();
      final serialized = transport.payloads.join();
      expect(serialized, contains('invalid_image'));
      expect(serialized, contains('PlatformException'));
      expect(serialized, isNot(contains('PRIVATE_')));
    },
  );

  test(
    'draft validation distinguishes limits from stale media without payloads',
    () {
      final limit = selectedCause(
        const DraftRepositoryException(
          DraftRepositoryFailureReason.invalidRequest,
          validationFailure: DraftValidationFailure.imageLimit,
        ),
      );
      final stale = selectedCause(
        const DraftRepositoryException(
          DraftRepositoryFailureReason.invalidRequest,
          validationFailure: DraftValidationFailure.staleMedia,
        ),
      );
      expect(limit['message'], contains('imageLimit'));
      expect(stale['message'], contains('staleMedia'));
      expect(
        selectedCause(
          PlatformException(
            code: 'PRIVATE_UNKNOWN_CODE',
            message: 'PRIVATE_MESSAGE',
          ),
        ).toString(),
        isNot(contains('PRIVATE_')),
      );
    },
  );

  test('mutation cause evidence preserves existing UI and terminal policy', () {
    const error = PdsMutationUnresolvedException(
      diagnosticCause: PdsMutationAmbiguousException(
        retryAfterSeconds: 1,
        diagnosticCause: ApiNetworkError('PRIVATE_OFFLINE'),
      ),
    );
    final mapped = AppErrorMapper.map(
      error,
      fallbackKind: AppErrorKind.backgroundLoadFailed,
      fallbackClassification: 'provider.failed',
    );
    expect(mapped.kind, AppErrorKind.backgroundLoadFailed);
    expect(mapped.sentryClassification, 'provider.failed');
    expect(
      isExpectedDiagnostic(
        error,
        const ReportContext(
          feature: 'like',
          operation: 'provider',
          classification: 'provider.failed',
        ),
      ),
      isFalse,
    );
  });

  test('SDK retains correlation through exhausted mutation wrappers', () async {
    final transport = SerializedTransport();
    await Sentry.init((options) {
      options
        ..dsn = 'https://public@example.invalid/1'
        ..transport = transport;
      configureDiagnosticOptions(options);
    });
    addTearDown(Sentry.close);
    const requestId = '10000000-0000-4000-8000-000000000001';
    const error = PdsMutationUnresolvedException(
      diagnosticCause: PdsMutationAmbiguousException(
        retryAfterSeconds: 1,
        diagnosticCause: ApiServerError(
          'PRIVATE_MESSAGE',
          details: ApiFailureDetails(
            statusCode: 502,
            method: 'POST',
            requestId: requestId,
            appViewError: 'upstream_unavailable',
            responseKind: 'appViewError',
          ),
        ),
      ),
    );
    await Sentry.captureException(error, stackTrace: StackTrace.current);
    await transport.issueReceived.future.timeout(const Duration(seconds: 5));
    final event = transport.payloads
        .map(jsonDecode)
        .whereType<Map<String, dynamic>>()
        .firstWhere((payload) => payload.containsKey('exception'));
    expect((event['exception'] as Map)['values'], hasLength(3));
    expect(
      (event['contexts'] as Map)['operation'],
      containsPair('appViewRequestId', requestId),
    );
    expect(
      (event['contexts'] as Map)['operation'],
      containsPair('httpStatus', 502),
    );
    expect(transport.payloads.join(), contains('upstream_unavailable'));
    expect(transport.payloads.join(), isNot(contains('PRIVATE_MESSAGE')));
  });

  test(
    'SDK serializes the preserved native cause chain without private prose',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final error = DiagnosticFormatException(
        'Search hashtag postCount missing',
        diagnosticCause: const FormatException('PRIVATE_RESPONSE'),
        diagnosticStack: StackTrace.current,
      );
      await Sentry.captureException(error, stackTrace: StackTrace.current);
      await transport.issueReceived.future.timeout(const Duration(seconds: 5));
      final serialized = transport.payloads.join();
      expect(serialized, contains('Search hashtag postCount missing'));
      final event = transport.payloads
          .map(jsonDecode)
          .whereType<Map<String, dynamic>>()
          .firstWhere((payload) => payload.containsKey('exception'));
      expect((event['exception'] as Map)['values'], hasLength(2));
      expect(serialized, contains('FormatException'));
      expect(serialized, isNot(contains('PRIVATE_RESPONSE')));
    },
  );

  test(
    'static format explanation retains original cause without its prose',
    () async {
      const cause = FormatException('PRIVATE_RESPONSE');
      final error = DiagnosticFormatException(
        'Search hashtag postCount missing',
        diagnosticCause: cause,
        diagnosticStack: StackTrace.current,
      );
      final selected = jsonEncode(selectedCauses(error));
      expect(selected, contains('Search hashtag postCount missing'));
      expect(selected, contains('FormatException'));
      expect(selected, isNot(contains('PRIVATE_RESPONSE')));
      final options = SentryOptions();
      configureDiagnosticOptions(options);
      final event = await options.beforeSend!(
        SentryEvent(
          throwable: error,
          exceptions: [
            SentryException(
              type: 'DiagnosticFormatException',
              value: 'PRIVATE_RESPONSE',
              throwable: error,
            ),
          ],
        ),
        Hint(),
      );
      expect(
        jsonEncode(event!.toJson()),
        contains('Search hashtag postCount missing'),
      );
      expect(jsonEncode(event.toJson()), isNot(contains('PRIVATE_RESPONSE')));
    },
  );

  test('exhausted mutation retains original server evidence', () async {
    const cause = ApiServerError(
      'PRIVATE_MESSAGE',
      details: ApiFailureDetails(
        statusCode: 502,
        appViewError: 'upstream_unavailable',
      ),
    );
    try {
      await unwrapPdsMutationApi<void>(() async => throw cause);
      fail('Expected ambiguous failure');
    } on PdsMutationAmbiguousException catch (ambiguous, stack) {
      expect(ambiguous.diagnosticCause, same(cause));
      expect(ambiguous.diagnosticStack, isNotNull);
      final error = PdsMutationUnresolvedException(
        diagnosticCause: ambiguous,
        diagnosticStack: stack,
      );
      final selected = jsonEncode(selectedCauses(error));
      expect(selected, contains('PDS mutation retry budget exhausted'));
      expect(selected, contains('ApiServerError'));
      expect(selected, isNot(contains('PRIVATE_MESSAGE')));
    }
  });
}
