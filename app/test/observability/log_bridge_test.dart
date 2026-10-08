import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../test_support/serialized_sentry_transport.dart';

void main() {
  for (final enabled in [false, true]) {
    test(
      'SDK-T05 native logging gate $enabled keeps independent local fallback',
      () async {
        final transport = _LoggingTransport();
        await Sentry.init((options) {
          options
            ..dsn = 'https://public@example.invalid/1'
            ..transport = transport
            ..enableLogs = enabled;
          configureDiagnosticOptions(options);
        });
        addTearDown(Sentry.close);
        final local = <String>[];
        final sub = configureRootLogForwarding(platformSink: local.add);
        addTearDown(sub.cancel);
        Logger.root.level = Level.ALL;
        Logger('Storage')
          ..info('Storage initialization started')
          ..warning('Storage unavailable token=PRIVATE_TOKEN')
          ..severe(
            'Storage read failed',
            StateError('PRIVATE_BODY'),
            StackTrace.current,
          )
          ..fine('Verbose diagnostics');
        if (enabled) {
          // Wait for the serialized batch, not one event-loop turn: the native
          // integration enriches logs asynchronously before buffering them.
          await transport.logsReceived.future.timeout(
            const Duration(seconds: 15),
          );
        }
        await Sentry.close();
        final payloads = transport.payloads
            .map(jsonDecode)
            .cast<Map<String, dynamic>>()
            .toList();
        expect(payloads.where((p) => p.containsKey('exception')), isEmpty);
        final logs = payloads
            .expand((p) => (p['items'] as List?) ?? [])
            .toList();
        expect(logs, hasLength(enabled ? 3 : 0));
        if (enabled) {
          expect(logs.toString(), contains('auto.log.logging'));
          expect(logs.toString(), contains('Storage initialization started'));
          expect(logs.toString(), isNot(contains('Verbose diagnostics')));
        }
        expect(local, hasLength(3));
        expect(local.join(), contains('Storage initialization started'));
        expect(local.join(), isNot(contains('Verbose diagnostics')));
        expect(local.join(), contains('StateError'));
        expect(
          local.join() + transport.payloads.join(),
          isNot(contains('PRIVATE_')),
        );
      },
    );
  }
}

class _LoggingTransport extends SerializedTransport {
  final logsReceived = Completer<void>();

  @override
  Future<SentryId?> send(SentryEnvelope envelope) async {
    final id = await super.send(envelope);
    final logs = payloads
        .map(jsonDecode)
        .cast<Map<String, dynamic>>()
        .expand((payload) => (payload['items'] as List?) ?? [])
        .length;
    if (logs >= 3 && !logsReceived.isCompleted) logsReceived.complete();
    return id;
  }
}
