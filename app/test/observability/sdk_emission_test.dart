import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/diagnostic_summary.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/diagnostic_evidence.dart';
import '../test_support/serialized_sentry_transport.dart';

void main() {
  test(
    'SIM-T04 IR-006 scope and hint attachments never reach SDK envelopes',
    () async {
      final transport = SerializedTransport();
      var attachmentLoads = 0;
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport
          ..release = 'review-release'
          ..tracesSampleRate = 1;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      await Sentry.configureScope((scope) {
        scope.addAttachment(
          SentryAttachment.fromLoader(
            loader: () {
              attachmentLoads++;
              return Uint8List.fromList(
                utf8.encode('private-scope-attachment-canary'),
              );
            },
            filename: 'private-scope-filename-canary.txt',
            addToTransactions: true,
          ),
        );
      });
      final hint =
          Hint.withAttachment(
              SentryAttachment.fromIntList(
                utf8.encode('private-hint-attachment-canary'),
                'private-hint.txt',
              ),
            )
            ..screenshot = SentryAttachment.fromIntList(
              utf8.encode('private-screenshot-canary'),
              'screenshot.png',
            )
            ..viewHierarchy = SentryAttachment.fromIntList(
              utf8.encode('private-hierarchy-canary'),
              'hierarchy.json',
            );
      await Sentry.captureException(
        StateError('private-exception-canary'),
        stackTrace: StackTrace.fromString(
          '#0 readRecord (package:craftsky_app/post.dart:12:3)',
        ),
        hint: hint,
      );
      await Sentry.startTransaction('post.read', 'post.read').finish();
      await Sentry.close();
      expect(transport.payloads, hasLength(2));
      final issue =
          jsonDecode(transport.payloads.first) as Map<String, dynamic>;
      final transaction =
          jsonDecode(transport.payloads.last) as Map<String, dynamic>;
      expect(issue['exception'].toString(), contains('StateError'));
      expect(issue['exception'].toString(), contains('readRecord'));
      expect(issue['release'], 'review-release');
      expect(transaction['transaction'], 'post.read');
      expect(transaction['release'], 'review-release');
      for (final canary in [
        'private-scope-attachment-canary',
        'private-scope-filename-canary',
        'private-hint-attachment-canary',
        'private-screenshot-canary',
        'private-hierarchy-canary',
        'private-exception-canary',
      ]) {
        expect(transport.payloads.join(), isNot(contains(canary)));
      }
      expect(attachmentLoads, 0);
    },
  );

  test(
    'UT-014 final SDK summary retains only explicitly public references',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final identity = SignedIn(did: 'did:plc:actor', handle: 'actor.test');
      for (final public in [true, false]) {
        await const SentryErrorReporter().captureException(
          StateError('PRIVATE_MODEL_PROSE'),
          stackTrace: StackTrace.fromString(
            '#0 loadIdentity (package:craftsky_app/provider.dart:12:3)',
          ),
          context: ReportContext(
            feature: 'Profile',
            operation: 'read',
            classification: 'profile.read',
            workflow: identity.diagnosticSummary(publicWorkflow: public),
          ),
        );
      }
      await Sentry.close();
      final events = transport.payloads
          .map((payload) => jsonDecode(payload) as Map<String, dynamic>)
          .where((event) => event.containsKey('exception'))
          .toList();
      expect(events, hasLength(2));
      expect(events.first['contexts'].toString(), contains('did:plc:actor'));
      expect(
        events.last['contexts'].toString(),
        isNot(contains('did:plc:actor')),
      );
      expect(events.first['contexts'].toString(), contains('signedIn'));
      expect(transport.payloads.join(), contains('StateError'));
      expect(transport.payloads.join(), contains('loadIdentity'));
      expect(transport.payloads.join(), isNot(contains('PRIVATE_MODEL_PROSE')));
    },
  );
  test(
    'IT-016 root retry bursts preserve terminal across platform and SDK Logs',
    () async {
      for (final logsEnabled in [false, true]) {
        final transport = SerializedTransport();
        final local = <String>[];
        await Sentry.init((options) {
          options
            ..dsn = 'https://public@example.invalid/1'
            ..transport = transport
            ..enableLogs = logsEnabled;
          configureDiagnosticOptions(options);
        });
        final subscription = configureRootLogForwarding(
          reporter: const SentryErrorReporter(),
          platformSink: local.add,
        );
        Logger.root.level = Level.ALL;
        for (var index = 0; index < 10; index++) {
          final terminal = index == 9;
          Logger('Worker').severe(
            DiagnosticMessage(
              'Operation failed',
              context: ReportContext(
                feature: 'Schedule',
                operation: 'publish',
                classification: 'schedule.failed',
                outcome: terminal
                    ? DiagnosticOutcome.terminal
                    : DiagnosticOutcome.retry,
                safeDiagnostics: {
                  'failureStage': terminal ? 'publication' : 'upload',
                  'attempt': index + 1,
                },
                workflow: const PrivateOperationalFailureContext(
                  accountDid: 'did:plc:initiator',
                  workflowRef: '40000000-0000-4000-8000-000000000001',
                ),
              ),
            ),
            StateError('PRIVATE_RETRY'),
            StackTrace.fromString(
              '#0 publishScheduledPost (package:craftsky_app/worker.dart:12:3)',
            ),
          );
        }
        await const SentryErrorReporter().captureException(
          StateError('PRIVATE_RETRY'),
          stackTrace: StackTrace.fromString(
            '#0 publishScheduledPost (package:craftsky_app/worker.dart:12:3)',
          ),
          context: const ReportContext(
            feature: 'Schedule',
            operation: 'publish',
            classification: 'schedule.failed',
            outcome: DiagnosticOutcome.terminal,
          ),
        );
        await transport.issueReceived.future.timeout(
          const Duration(seconds: 5),
        );
        await subscription.cancel();
        await Sentry.close();
        final selected = local
            .map((line) => jsonDecode(line) as Map<String, dynamic>)
            .toList();
        expect(selected, hasLength(10));
        final payloads = transport.payloads
            .map((payload) => jsonDecode(payload) as Map<String, dynamic>)
            .toList();
        final issues = payloads
            .where((payload) => payload.containsKey('exception'))
            .toList();
        final logs = payloads
            .expand(
              (payload) =>
                  (payload['items'] as List<dynamic>?) ?? const <dynamic>[],
            )
            .cast<Map<String, dynamic>>()
            .toList();
        expect(issues, hasLength(1));
        expect(issues.single['exception'].toString(), contains('StateError'));
        expect(
          issues.single['exception'].toString(),
          contains('publishScheduledPost'),
        );
        expect(logs, hasLength(logsEnabled ? 10 : 0));
        expect(
          '${local.join()}${transport.payloads.join()}',
          isNot(contains('PRIVATE_RETRY')),
        );
      }
    },
  );

  test(
    'UT-015 changing identifiers preserve stable grouping dimensions',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      for (var index = 0; index < 20; index++) {
        await const SentryErrorReporter().captureException(
          StateError('PRIVATE_$index'),
          stackTrace: StackTrace.fromString(
            '#0 readPublishedPost (package:craftsky_app/post.dart:12:3)',
          ),
          context: ReportContext(
            feature: 'Post',
            operation: 'read',
            classification: 'post.read',
            safeDiagnostics: {
              'appViewRequestId':
                  '00000000-0000-4000-8000-'
                  '${index.toString().padLeft(12, '0')}',
            },
            workflow: PublicRecordContext(
              actorDid: 'did:plc:actor$index',
              targetDid: 'did:plc:target$index',
            ),
          ),
        );
      }
      await const SentryErrorReporter().captureException(
        const FormatException('PRIVATE_PARSE'),
        context: const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'post.read',
        ),
      );
      await Sentry.close();
      final events = transport.payloads
          .map((payload) => jsonDecode(payload) as Map<String, dynamic>)
          .toList();
      expect(events, hasLength(21));
      final tags = jsonEncode(events.first['tags']);
      final requests = <String>{};
      for (final event in events.take(20)) {
        expect(jsonEncode(event['tags']), tags);
        expect(event['fingerprint'], isNull);
        final contexts = event['contexts'] as Map<String, dynamic>;
        requests.add(
          (contexts['operation'] as Map<String, dynamic>)['appViewRequestId']
              as String,
        );
        expect(
          (contexts['diagnostic'] as Map<String, dynamic>)['actorDid'],
          startsWith('did:plc:actor'),
        );
        expect(event['exception'].toString(), contains('StateError'));
      }
      expect(requests, hasLength(20));
      expect(events.last['exception'].toString(), contains('FormatException'));
      expect(transport.payloads.join(), isNot(contains('PRIVATE_')));
    },
  );

  test('AT-006 private failure context survives final SDK selection', () async {
    final transport = SerializedTransport();
    await Sentry.init((options) {
      options
        ..dsn = 'https://public@example.invalid/1'
        ..transport = transport;
      configureDiagnosticOptions(options);
    });
    addTearDown(Sentry.close);
    await const SentryErrorReporter().captureException(
      StateError('opaque private worker prose'),
      context: const ReportContext(
        feature: 'Schedule',
        operation: 'schedule.publish',
        classification: 'schedule.failed',
        workflow: PrivateOperationalFailureContext(
          accountDid: 'did:plc:alice',
          workflowRef: '00000000-0000-4000-8000-000000000401',
        ),
      ),
    );
    await Sentry.close();
    final payload = transport.payloads.join();
    expect(payload, contains('operationAccountDid'));
    expect(payload, contains('did:plc:alice'));
    expect(payload, contains('00000000-0000-4000-8000-000000000401'));
    expect(payload, isNot(contains('opaque private worker prose')));
  });

  test(
    'IT-009 real provider retains original cause once and '
    'suppresses expected owners',
    () async {
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
        reporter: const SentryErrorReporter(),
        platformSink: local.add,
      );
      addTearDown(subscription.cancel);
      final cause = StateError('opaque private provider canary');
      final provider = FutureProvider<int>(
        name: 'publicReadProvider',
        (ref) => Future<int>.error(
          cause,
          StackTrace.fromString(
            '#0 readPublishedPost (package:craftsky_app/post.dart:12:3)',
          ),
        ),
      );
      for (var i = 0; i < 2; i++) {
        final container = ProviderContainer(
          retry: appProviderRetry,
          observers: [const ProviderLogger(reporter: SentryErrorReporter())],
        );
        await expectLater(container.read(provider.future), throwsStateError);
        container.dispose();
      }
      await const SentryErrorReporter().captureException(
        const ApiUnauthorized(),
        context: const ReportContext(
          feature: 'Auth',
          operation: 'resume',
          classification: 'auth.session_expired',
        ),
      );
      await const SentryErrorReporter().captureException(
        const ApiNetworkError('opaque offline'),
        context: const ReportContext(
          feature: 'Api',
          operation: 'read',
          classification: 'network',
        ),
      );
      await const SentryErrorReporter().captureException(
        const ApiNetworkError('opaque terminal'),
        context: const ReportContext(
          feature: 'Api',
          operation: 'read',
          classification: 'network',
          outcome: DiagnosticOutcome.terminal,
        ),
      );
      await Sentry.close();
      final events = transport.payloads
          .map(jsonDecode)
          .cast<Map<String, dynamic>>()
          .where((payload) => payload.containsKey('exception'))
          .toList();
      expect(events.length, 2);
      expect(
        events.take(1).map((event) => event.toString()),
        everyElement(contains('StateError')),
      );
      expect(
        events.take(1).map((event) => event.toString()),
        everyElement(contains('readPublishedPost')),
      );
      expect(local.join(), contains('StateError'));
      expect(local.join(), contains('readPublishedPost'));
      expect(
        local.join() + transport.payloads.join(),
        isNot(contains('opaque private provider canary')),
      );
      writeDiagnosticEvidence(
        'flutter-provider',
        local: local,
        exported: transport.payloads,
      );
    },
  );
  test(
    'IT-010 mapped API parse failure retains original '
    'typed cause and supplied stack',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      const underlying = FormatException(
        'opaque private body',
        'private JSON canary',
      );
      final stack = StackTrace.fromString(
        '#0 decodePublicResponse (package:craftsky_app/api.dart:12:3)',
      );
      final api = ApiNetworkError(
        'network',
        details: ApiFailureDetails(cause: underlying, stackTrace: stack),
      );
      final mapped = AppErrorMapper.map(api);
      await const SentryErrorReporter().captureException(
        mapped,
        stackTrace: stack,
        context: ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: mapped.sentryClassification,
          safeDiagnostics: mapped.safeDiagnostics,
        ),
      );
      await Sentry.close();
      final data = transport.payloads.join();
      expect(data, contains('FormatException'));
      expect(data, contains('decodePublicResponse'));
      expect(data, isNot(contains('opaque private body')));
      expect(data, isNot(contains('private JSON canary')));
    },
  );

  test(
    'UT-006 server request ID survives mapper and final issue without sampling',
    () async {
      const requestId = 'b1b5a67d-0480-40b3-a9dc-201354ef91ed';
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport
          ..environment = 'test'
          ..release = 'test-release';
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      const error = ApiServerError(
        'http_500',
        details: ApiFailureDetails(statusCode: 500, requestId: requestId),
      );
      final mapped = AppErrorMapper.map(error);
      await const SentryErrorReporter().captureException(
        error,
        context: ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'api.server_error',
          safeDiagnostics: mapped.safeDiagnostics,
        ),
      );
      await const SentryErrorReporter().captureException(
        const ApiServerError('http_500'),
        context: const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'api.server_error',
        ),
      );
      await Sentry.close();
      final events = transport.payloads
          .map(jsonDecode)
          .cast<Map<String, dynamic>>()
          .toList();
      expect(
        (events.first['contexts'] as Map<String, dynamic>)['operation'],
        containsPair('appViewRequestId', requestId),
      );
      expect(
        (events.first['tags'] as Map<String, dynamic>).containsKey(
          'appViewRequestId',
        ),
        isFalse,
      );
      expect(events.first['release'], 'test-release');
      expect(events.first['environment'], 'test');
      expect(
        (events.last['contexts'] as Map<String, dynamic>).containsKey(
          'correlation',
        ),
        isFalse,
      );
    },
  );
  test('IT-005 actual root Logs gate leaves safe local diagnostics', () async {
    final transport = SerializedTransport();
    await Sentry.init((options) {
      options
        ..dsn = 'https://public@example.invalid/1'
        ..transport = transport
        ..enableLogs = false;
      configureDiagnosticOptions(options);
    });
    addTearDown(Sentry.close);
    final local = <String>[];
    final subscription = configureRootLogForwarding(
      reporter: const SentryErrorReporter(),
      platformSink: local.add,
    );
    addTearDown(subscription.cancel);
    Logger('Storage').warning('device-id write failed; using in-memory only');
    await Future<void>.delayed(Duration.zero);
    await Sentry.close();
    expect(transport.payloads, isEmpty);
    expect(local.length, 1);
    expect(
      local.single,
      contains('device-id write failed; using in-memory only'),
    );
  });
  test(
    'AT-005 actual root forwarding retains distinct '
    'independent Logs and later breadcrumbs',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport
          ..enableLogs = true;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      final subscription = configureRootLogForwarding(
        reporter: const SentryErrorReporter(),
        platformSink: (_) {},
      );
      addTearDown(subscription.cancel);
      Logger.root.level = Level.ALL;
      Logger('Storage').severe('pending handoff storage failed');
      Logger('Storage').severe('confirmed handoff storage failed');
      Logger('Storage').warning('device-id write failed; using in-memory only');
      Logger('Bootstrap').info(
        const DiagnosticMessage(
          'bootstrap complete',
          context: ReportContext(
            feature: 'Bootstrap',
            operation: 'initialize',
            classification: 'lifecycle',
          ),
          significant: true,
        ),
      );
      Logger('Bootstrap').fine('bootstrap starting');
      Logger('Storage').severe(
        'pending handoff mutation failed',
        StateError('opaque private failure'),
        StackTrace.fromString(
          '#0 writePublicRecord (package:craftsky_app/storage.dart:12:3)',
        ),
      );
      await const SentryErrorReporter().captureException(
        StateError('opaque private failure'),
        stackTrace: StackTrace.fromString(
          '#0 writePublicRecord (package:craftsky_app/storage.dart:12:3)',
        ),
        context: const ReportContext(
          feature: 'Storage',
          operation: 'write',
          classification: 'storage.failed',
        ),
      );
      await transport.issueReceived.future.timeout(const Duration(seconds: 5));
      await Sentry.close();
      final payloads = transport.payloads
          .map(jsonDecode)
          .cast<Map<String, dynamic>>()
          .toList();
      final events = payloads
          .where((payload) => payload.containsKey('exception'))
          .toList();
      final logs = payloads
          .expand((payload) => (payload['items'] as List?) ?? const [])
          .cast<Map<String, dynamic>>()
          .toList();
      expect(events.length, 1);
      expect(logs.length, 5);
      expect(
        logs.map((log) => log['body']),
        containsAll([
          'pending handoff storage failed',
          'confirmed handoff storage failed',
          'device-id write failed; using in-memory only',
          'bootstrap complete',
        ]),
      );
      expect(
        logs.map((log) => log['body']),
        isNot(contains('bootstrap starting')),
      );
      expect(
        events.single['breadcrumbs'].toString(),
        contains('bootstrap complete'),
      );
      expect(transport.payloads.join(), contains('writePublicRecord'));
      expect(
        transport.payloads.join(),
        isNot(contains('opaque private failure')),
      );
    },
  );
  test(
    'IT-011 final serialized Logs and traces reject unsupported SDK data',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport
          ..enableLogs = true
          ..tracesSampleRate = 1;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      await Sentry.logger.error(
        'SDK log token=private-log-canary',
        attributes: {'password': SentryAttribute.string('password-canary')},
      );
      await const SentryErrorReporter().captureMessage(
        'pds write completed',
        context: const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'post.failed',
        ),
      );
      final transaction = Sentry.startTransaction('post.read', 'post.read')
        ..setData('private', {'draft': 'draft-canary'})
        ..setMeasurement('private-measurement-canary', 1)
        ..context.description = 'opaque private trace';
      final child = transaction.startChild(
        'private-capability-canary',
        description: 'opaque private trace',
      )..setData('private', {'draft': 'draft-canary'});
      await child.finish();
      await transaction.finish();
      await Sentry.close();
      final serialized = transport.payloads.join();
      for (final canary in [
        'private-log-canary',
        'password-canary',
        'draft-canary',
        'private-capability-canary',
        'private-measurement-canary',
      ]) {
        expect(serialized, isNot(contains(canary)));
      }
      for (final positive in ['pds write completed', 'post.read']) {
        expect(serialized, contains(positive));
      }
    },
  );
  test(
    'AT-007 final serialized SDK events reject enrichment '
    'and retain selected causes',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
        configureDiagnosticOptions(options);
      });
      addTearDown(Sentry.close);
      await Sentry.configureScope((scope) async {
        await scope.setContexts('unknown', {
          'nested': [
            {'draft': 'private-draft-canary'},
          ],
        });
        await scope.setUser(
          SentryUser(
            email: 'email-canary@example.invalid',
            id: 'device-canary',
          ),
        );
        await scope.setTag('operation', 'opaque private operation');
        await scope.addBreadcrumb(
          Breadcrumb(
            message: 'opaque private breadcrumb',
            data: {'draft': 'private-draft-canary'},
          ),
        );
      });
      await const SentryErrorReporter().captureException(
        const ApiUnauthorized(),
        context: const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'auth',
          outcome: DiagnosticOutcome.terminal,
          workflow: PublicRecordContext(targetDid: 'did:plc:target'),
        ),
        stackTrace: StackTrace.fromString(
          '#0 loadPublicRecord (file:///Users/path-canary/work/file.dart:12:3)',
        ),
      );
      await Sentry.captureEvent(
        SentryEvent(
          message: SentryMessage('opaque private SDK prose'),
          exceptions: [
            SentryException(
              type: 'ProviderFailure',
              value: 'opaque private SDK prose',
              stackTrace: SentryStackTrace(
                frames: [
                  SentryStackFrame(
                    function: 'loadPublicRecord',
                    fileName: '/Users/path-canary/file.dart',
                    vars: {'draft': 'private-draft-canary'},
                    contextLine: 'private-line-canary',
                  ),
                ],
              ),
            ),
          ],
          request: SentryRequest(
            url:
                'https://user:password-canary@example.invalid/capability-canary?code=code-canary',
            headers: {'Cookie': 'cookie-canary'},
          ),
        ),
      );
      await Sentry.close();
      expect(transport.payloads.length, 2);
      final serialized = transport.payloads.join();
      for (final canary in [
        'private-draft-canary',
        'email-canary',
        'device-canary',
        'password-canary',
        'capability-canary',
        'code-canary',
        'cookie-canary',
        'opaque private',
        'path-canary',
        'private-line-canary',
      ]) {
        expect(serialized, isNot(contains(canary)));
      }
      for (final positive in [
        'did:plc:target',
        'ApiUnauthorized',
        'ProviderFailure',
        'loadPublicRecord',
      ]) {
        expect(serialized, contains(positive));
      }
    },
  );
}
