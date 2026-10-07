import 'dart:convert';

import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/observability/diagnostic_details.dart';
import 'package:craftsky_app/shared/observability/diagnostic_outcome.dart';
import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:craftsky_app/shared/observability/sentry_sanitizer.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

final class OpaqueProviderFailure implements Exception {
  @override
  String toString() => 'opaque linen sentence';
}

final class PatternFailure implements Exception {
  @override
  String toString() => 'public linen pattern access_token=credential-canary';
}

void main() {
  test('UT-013 stack omission marker fits frame and byte budgets', () {
    final frames = selectedStack(
      StackTrace.fromString(
        List.generate(
          100,
          (index) =>
              '#$index loadPublicRecord (package:craftsky_app/provider.dart:12:3)',
        ).join('\n'),
      ),
    );
    expect(frames.length, lessThanOrEqualTo(64));
    expect(utf8.encode(jsonEncode(frames)).length, lessThanOrEqualTo(10000));
    expect(frames.first['function'], 'loadPublicRecord');
    expect(frames.last['omitted'], isNotNull);
  });
  test('UT-003 unknown API code does not acquire public prose provenance', () {
    final selected = SentrySanitizer.sanitizeContext({
      'appViewError': 'opaque_private_identifier',
      'httpStatus': 500,
    });
    expect(selected['appViewError'], 'opaque_private_identifier');
    expect(selected['httpStatus'], 500);
    expect(
      SentrySanitizer.sanitizeContext({
        'appViewError': 'not_found',
      })['appViewError'],
      'not_found',
    );
  });
  test('UT-011 wrapped AppError preserves explicit expected outcome', () {
    final error = AppError(
      AppErrorKind.unexpected,
      reportableOverride: false,
      diagnosticCause: StateError('PRIVATE_CAUSE'),
    );
    const context = ReportContext(
      feature: 'Post',
      operation: 'read',
      classification: 'post.read',
    );
    expect(isExpectedDiagnostic(error, context), isTrue);
    expect(
      isExpectedDiagnostic(
        error,
        const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'post.read',
          outcome: DiagnosticOutcome.terminal,
        ),
      ),
      isFalse,
    );
  });
  test('UT-013 UTF8 bounds sanitize before truncation', () {
    for (final size in [
      maxDiagnosticTextBytes - 1,
      maxDiagnosticTextBytes,
      maxDiagnosticTextBytes + 1,
    ]) {
      final selected = sanitizeKnownDiagnosticText('x' * size);
      expect(
        utf8.encode(selected).length,
        lessThanOrEqualTo(maxDiagnosticTextBytes),
      );
      if (size > maxDiagnosticTextBytes) {
        expect(selected, endsWith('[TRUNCATED]'));
      }
    }
    final selected = sanitizeKnownDiagnosticText(
      '${'🧶' * 1024} access_token=${'secret-canary' * 512}',
    );
    expect(
      utf8.encode(selected).length,
      lessThanOrEqualTo(maxDiagnosticTextBytes),
    );
    expect(selected, endsWith('[TRUNCATED]'));
    expect(selected, isNot(contains('secret-canary')));
  });
  test('UT-012 admitted mixed known formats preserve public context', () {
    const text =
        'public linen did:plc:target at://did:plc:target/social.craftsky.feed.post/record '
        'Cookie=session-cookie-canary email=user-email-canary@example.invalid '
        'https://userinfo-canary:password-canary@example.invalid/private/capability-canary?X-Amz-Signature=signature-canary '
        'code%3Dencoded-code-canary%26state%3Dencoded-state-canary '
        '/Users/local-path-canary/project/file.dart '
        '-----BEGIN PRIVATE '
        'KEY-----\nprivate-key-canary\n-----END PRIVATE KEY-----';
    final selected = sanitizeKnownDiagnosticText(text);
    for (final canary in [
      'cookie-canary',
      'email-canary',
      'userinfo-canary',
      'password-canary',
      'capability-canary',
      'signature-canary',
      'code-canary',
      'state-canary',
      'local-path-canary',
      'private-key-canary',
    ]) {
      expect(selected, isNot(contains(canary)));
    }
    expect(selected, contains('public linen'));
    expect(
      selected,
      contains('at://did:plc:target/social.craftsky.feed.post/record'),
    );
  });
  test(
    'UT-003 unknown prose excluded with concrete type and supplied stack',
    () async {
      SentryEvent? emitted;
      await Sentry.init((options) {
        configureDiagnosticOptions(options);
        final filter = options.beforeSend!;
        options
          ..dsn = 'https://public@example.invalid/1'
          ..beforeSend = (event, hint) async {
            emitted = await filter(event, hint);
            return null;
          };
      });
      addTearDown(Sentry.close);
      await const SentryErrorReporter().captureException(
        OpaqueProviderFailure(),
        context: const ReportContext(
          feature: 'Provider',
          operation: 'load',
          classification: 'provider.failed',
        ),
        stackTrace: StackTrace.fromString(
          '#0      loadPublicRecord (package:craftsky_app/provider.dart:12:3)',
        ),
      );
      expect(emitted, isNotNull);
      final exceptions = emitted!.exceptions!;
      expect(exceptions.last.type, 'OpaqueProviderFailure');
      expect(exceptions.last.value, isNot(contains('opaque linen sentence')));
      expect(
        exceptions.last.stackTrace?.frames.map((frame) => frame.function),
        contains('loadPublicRecord'),
      );
    },
  );
  test('UT-001 approved static API explanation survives', () async {
    SentryEvent? emitted;
    await Sentry.init((options) {
      configureDiagnosticOptions(options);
      final filter = options.beforeSend!;
      options
        ..dsn = 'https://public@example.invalid/1'
        ..beforeSend = (event, hint) async {
          emitted = await filter(event, hint);
          return null;
        };
    });
    addTearDown(Sentry.close);
    const error = ApiUnauthorized();
    await const SentryErrorReporter().captureException(
      error,
      context: const ReportContext(
        feature: 'API',
        operation: 'read',
        classification: 'auth',
        outcome: DiagnosticOutcome.terminal,
      ),
    );
    expect(emitted!.exceptions!.last.type, 'ApiUnauthorized');
    expect(emitted!.exceptions!.last.value, 'Operation failed');
    expect(emitted!.contexts['failure'].toString(), contains(error.toString()));
  });

  test('UT-002 supplied frames exclude local user path', () async {
    SentryEvent? emitted;
    await Sentry.init((options) {
      configureDiagnosticOptions(options);
      final filter = options.beforeSend!;
      options
        ..dsn = 'https://public@example.invalid/1'
        ..beforeSend = (event, hint) async {
          emitted = await filter(event, hint);
          return null;
        };
    });
    addTearDown(Sentry.close);
    await const SentryErrorReporter().captureException(
      OpaqueProviderFailure(),
      context: const ReportContext(
        feature: 'Provider',
        operation: 'load',
        classification: 'provider.failed',
      ),
      stackTrace: StackTrace.fromString(
        '#0      loadPublicRecord (file:///Users/private-canary/work/provider.dart:12:3)',
      ),
    );
    final frames = emitted!.exceptions!.last.stackTrace!.frames;
    expect(frames.map((frame) => frame.function), contains('loadPublicRecord'));
    expect(frames.map((frame) => frame.fileName), contains('provider.dart'));
    expect(emitted!.toJson().toString(), isNot(contains('private-canary')));
  });

  test(
    'UT-004 explicit public actors survive independently of exception prose',
    () async {
      SentryEvent? emitted;
      await Sentry.init((options) {
        configureDiagnosticOptions(options);
        final filter = options.beforeSend!;
        options
          ..dsn = 'https://public@example.invalid/1'
          ..beforeSend = (event, hint) async {
            emitted = await filter(event, hint);
            return null;
          };
      });
      addTearDown(Sentry.close);
      await const SentryErrorReporter().captureException(
        OpaqueProviderFailure(),
        context: const ReportContext(
          feature: 'Post',
          operation: 'read',
          classification: 'post.failed',
          workflow: PublicRecordContext(
            actorDid: 'did:plc:actor',
            targetDid: 'did:plc:target',
            recordUri: 'at://did:plc:target/social.craftsky.feed.post/record',
            handle: 'target.example.invalid',
            cid: 'public-cid',
            nsid: 'social.craftsky.feed.post',
            recordKey: 'record',
          ),
        ),
      );
      final context = emitted!.contexts['diagnostic'] as Map<String, dynamic>;
      expect(context['actorDid'], 'did:plc:actor');
      expect(context['targetDid'], 'did:plc:target');
      expect(
        context['recordUri'],
        'at://did:plc:target/social.craftsky.feed.post/record',
      );
      expect(context['handle'], 'target.example.invalid');
      expect(context['cid'], 'public-cid');
      expect(context['nsid'], 'social.craftsky.feed.post');
      expect(context['recordKey'], 'record');
      expect(emitted!.tags?.containsKey('actorDid'), isFalse);
      expect(
        emitted!.toJson().toString(),
        isNot(contains('opaque linen sentence')),
      );
    },
  );

  test('UT-004 malformed public DID is marked as attempted', () async {
    SentryEvent? emitted;
    await Sentry.init((options) {
      configureDiagnosticOptions(options);
      final filter = options.beforeSend!;
      options
        ..dsn = 'https://public@example.invalid/1'
        ..beforeSend = (event, hint) async {
          emitted = await filter(event, hint);
          return null;
        };
    });
    addTearDown(Sentry.close);
    await const SentryErrorReporter().captureException(
      OpaqueProviderFailure(),
      context: const ReportContext(
        feature: 'Profile',
        operation: 'read',
        classification: 'validation',
        workflow: AttemptedPublicDidContext('did:plc:bad!'),
      ),
    );
    final context = emitted!.contexts['diagnostic'] as Map<String, dynamic>;
    expect(context['attemptedDid'], 'did:plc:bad!');
    expect(context['identifierValid'], false);
    expect(context['validationReason'], 'invalid DID');
  });

  test('UT-005 excerpt requires published failure provenance', () async {
    final events = <SentryEvent>[];
    await Sentry.init((options) {
      configureDiagnosticOptions(options);
      final filter = options.beforeSend!;
      options
        ..dsn = 'https://public@example.invalid/1'
        ..beforeSend = (event, hint) async {
          events.add((await filter(event, hint))!);
          return null;
        };
    });
    addTearDown(Sentry.close);
    for (final workflow in <DiagnosticWorkflow?>[
      const PublishedRecordParseFailureContext(
        record: PublicRecordContext(),
        text: 'public linen pattern access_token=credential-canary',
      ),
      null, // Identical text in a draft/failed publication is unknown prose.
    ]) {
      await const SentryErrorReporter().captureException(
        PatternFailure(),
        context: ReportContext(
          feature: 'Post',
          operation: 'decode',
          classification: 'post.failed',
          workflow: workflow,
        ),
      );
    }
    expect(events, hasLength(2));
    expect(events.first.toJson().toString(), contains('public linen pattern'));
    expect(
      events.last.toJson().toString(),
      isNot(contains('public linen pattern')),
    );
    for (final event in events) {
      expect(event.toJson().toString(), isNot(contains('credential-canary')));
    }
  });

  test(
    'SIM-T01 client excludes paths and server prose without a route catalogue',
    () {
      final fields = SentrySanitizer.sanitizeContext({
        'httpMethod': 'GET',
        'incomingPath': '/v1/saves/PRIVATE',
        'appViewMessage': 'PRIVATE',
        'appViewError': 'future_code',
        'appViewRequestId': 'request-1',
      });
      expect(fields, {
        'httpMethod': 'GET',
        'appViewError': 'future_code',
        'appViewRequestId': 'request-1',
      });
    },
  );
}
