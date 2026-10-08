import 'dart:convert';
import 'dart:ui';

import 'package:craftsky_app/main.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

class PlatformOpaqueFailure implements Exception {
  @override
  String toString() => 'opaque private failure';
}

void main() {
  test(
    'CON-T01 root console defaults to INFO with protected debug opt-in',
    () async {
      final lines = <String>[];
      final previous = Logger.root.level;
      Logger.root.level = Level.FINE;
      addTearDown(() => Logger.root.level = previous);
      final subscription = configureRootLogForwarding(platformSink: lines.add);
      addTearDown(subscription.cancel);
      final log = Logger('ConsoleProbe')
        ..fine('Debug transition token=debug-canary')
        ..info('Application ready token=info-canary')
        ..warning('Operation retrying');
      expect(lines, hasLength(2));
      final infoRecord = jsonDecode(lines.first) as Map<String, dynamic>;
      expect(infoRecord['severity'], 'INFO');
      expect(lines.join(), isNot(contains('info-canary')));
      await subscription.cancel();
      lines.clear();
      final debugSubscription = configureRootLogForwarding(
        platformSink: lines.add,
        debugLogs: true,
      );
      addTearDown(debugSubscription.cancel);
      log.fine('Debug transition token=debug-canary');
      expect(lines, hasLength(1));
      final debugRecord = jsonDecode(lines.single) as Map<String, dynamic>;
      expect(debugRecord['severity'], 'FINE');
      expect(lines.single, isNot(contains('debug-canary')));
    },
  );

  test(
    'SIM-T03 Unicode and oversized identifiers remain a complete brief record',
    () {
      final lines = <String>[];
      final stack = StackTrace.fromString(
        List.generate(
          100,
          (i) => '#$i loadPost (package:craftsky_app/post.dart:12:3)',
        ).join('\n'),
      );
      DiagnosticEmitter(platformSink: lines.add).emitLocal(
        LogRecord(
          Level.SEVERE,
          '🧶' * 10000,
          'Post',
          StateError('PRIVATE_DRAFT'),
          stack,
          null,
          DiagnosticMessage(
            '🧶' * 10000,
            context: ReportContext(
              feature: 'Post',
              operation: 'read',
              classification: 'post.failed',
              safeDiagnostics: const {'appViewRequestId': 'normal-request'},
              workflow: PublicRecordContext(
                actorDid: 'did:plc:actor',
                recordUri: 'at://did:plc:actor/${'x' * 10000}',
              ),
            ),
          ),
        ),
      );
      final selected = jsonDecode(lines.single) as Map<String, dynamic>;
      expect(
        utf8.encode(lines.single).length,
        lessThanOrEqualTo(maxPlatformDiagnosticBytes),
      );
      expect(selected['appViewRequestId'], 'normal-request');
      expect(
        (selected['diagnostic'] as Map<String, dynamic>)['recordUri'],
        '[OMITTED: console identifier]',
      );
      expect((selected['cause'] as Map<String, dynamic>)['type'], 'StateError');
      expect(selected['stack'], hasLength(2));
    },
  );

  test('SIM-T03 console fallback emits one bounded brief record', () {
    final local = <String>[];
    DiagnosticEmitter(
      platformSink: local.add,
      environment: 'production',
      release: 'craftsky@1.0.0+1',
    ).emitLocal(
      LogRecord(
        Level.SEVERE,
        'Post load failed token=PRIVATE_TOKEN',
        'Post',
        StateError('PRIVATE_DRAFT_PROSE'),
        StackTrace.fromString(
          List.generate(
            64,
            (i) => '#$i loadPost (package:craftsky_app/post.dart:12:3)',
          ).join('\n'),
        ),
        null,
        const DiagnosticMessage(
          'Post load failed',
          context: ReportContext(
            feature: 'Post',
            operation: 'read',
            classification: 'post.read',
            safeDiagnostics: {
              'appViewRequestId': 'brief-request',
              'failureStage': 'load',
            },
            workflow: PrivateOperationalFailureContext(
              accountDid: 'did:plc:owner',
              workflowRef: '40000000-0000-4000-8000-000000000001',
            ),
          ),
        ),
      ),
    );
    expect(local, hasLength(1));
    final decoded = jsonDecode(local.single) as Map<String, dynamic>;
    expect(decoded, isNot(contains('chunk')));
    expect(decoded['cause'].toString(), contains('StateError'));
    expect(decoded['stack'].toString(), contains('loadPost'));
    expect(decoded['operation'], 'read');
    expect(decoded['appViewRequestId'], 'brief-request');
    expect(
      decoded['diagnostic'].toString(),
      contains('40000000-0000-4000-8000-000000000001'),
    );
    expect(
      utf8.encode(local.single).length,
      lessThanOrEqualTo(maxPlatformDiagnosticBytes),
    );
    expect(local.single, isNot(contains('PRIVATE_TOKEN')));
    expect(local.single, isNot(contains('PRIVATE_DRAFT_PROSE')));
  });
  test(
    'UT-006 local record retains configured process metadata without Sentry',
    () {
      final records = <String>[];
      DiagnosticEmitter(
        platformSink: records.add,
        environment: 'production',
        release: 'craftsky@1.0.0+1',
      ).emitLocal(
        LogRecord(
          Level.WARNING,
          'Operation failed',
          'Post',
          const FormatException('PRIVATE_CAUSE'),
        ),
      );
      expect(records.single, contains('production'));
      expect(records.single, contains('craftsky@1.0.0+1'));
      expect(records.single, contains('FormatException'));
      expect(records.single, isNot(contains('PRIVATE_CAUSE')));
    },
  );
  test(
    'IT-011 root platform output selects before formatting '
    'with Sentry disabled',
    () async {
      final records = <String>[];
      final subscription = configureRootLogForwarding(
        platformSink: records.add,
      );
      addTearDown(subscription.cancel);
      Logger.root.level = Level.ALL;
      final oldFlutter = FlutterError.onError;
      final oldPlatform = PlatformDispatcher.instance.onError;
      final oldBuilder = ErrorWidget.builder;
      addTearDown(() {
        FlutterError.onError = oldFlutter;
        PlatformDispatcher.instance.onError = oldPlatform;
        ErrorWidget.builder = oldBuilder;
      });
      registerErrorHandlers();
      final safeStack = StackTrace.fromString(
        '#0 loadPublicRecord (file:///Users/path-canary/work/file.dart:12:3)',
      );
      FlutterError.onError!(
        FlutterErrorDetails(
          exception: PlatformOpaqueFailure(),
          stack: safeStack,
        ),
      );
      PlatformDispatcher.instance.onError!(PlatformOpaqueFailure(), safeStack);
      Logger('Post').severe(
        'Platform failure token=PRIVATE_LOG_TOKEN',
        PlatformOpaqueFailure(),
        StackTrace.fromString(
          '#0 loadPublicRecord (file:///Users/path-canary/work/file.dart:12:3)',
        ),
      );
      await Future<void>.delayed(Duration.zero);
      expect(records, isNotEmpty);
      final serialized = records.join();
      expect(serialized, isNot(contains('PRIVATE_LOG_TOKEN')));
      expect(serialized, isNot(contains('path-canary')));
      expect(serialized, contains('PlatformOpaqueFailure'));
      expect(serialized, contains('loadPublicRecord'));
      for (final record in records) {
        expect(utf8.encode(record).length, lessThanOrEqualTo(2048));
        expect(jsonDecode(record), isA<Map<String, dynamic>>());
      }
    },
  );
}
