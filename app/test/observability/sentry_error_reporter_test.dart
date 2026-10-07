import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

void main() {
  test('SDK-T06 native frame metadata survives locals filtering', () async {
    final options = SentryOptions();
    configureDiagnosticOptions(options);
    final event = SentryEvent(
      exceptions: [
        SentryException(
          type: 'NativeFailure',
          value: 'private-error',
          stackTrace: SentryStackTrace(
            snapshot: true,
            lang: 'dart',
            frames: [
              SentryStackFrame(
                function: 'renderPost',
                fileName: 'post.dart',
                lineNo: 12,
                package: 'craftsky_app',
                native: true,
                platform: 'native',
                imageAddr: '0x1000',
                symbolAddr: '0x1010',
                instructionAddr: '0x1020',
                rawFunction: '_renderPost',
                vars: {'draft': 'PRIVATE_DRAFT'},
                contextLine: 'PRIVATE_SOURCE',
              ),
            ],
          ),
        ),
      ],
    );
    final safe = await options.beforeSend!(event, Hint());
    final frame = safe!.exceptions!.single.stackTrace!.frames.single;
    expect(safe.exceptions!.single.stackTrace!.snapshot, isTrue);
    expect(safe.exceptions!.single.stackTrace!.lang, 'dart');
    expect(frame.imageAddr, '0x1000');
    expect(frame.symbolAddr, '0x1010');
    expect(frame.instructionAddr, '0x1020');
    expect(frame.package, 'craftsky_app');
    expect(frame.native, isTrue);
    expect(frame.platform, 'native');
    expect(frame.rawFunction, '_renderPost');
    expect(frame.vars, isEmpty);
    expect(frame.contextLine, isNull);
  });

  test(
    'SIM-T04 final hook preserves SDK stacks and standard metadata',
    () async {
      final options = SentryOptions();
      configureDiagnosticOptions(options);
      final stack = SentryStackTrace(
        frames: [
          SentryStackFrame(
            function: 'loadPost',
            fileName: 'post.dart',
            lineNo: 12,
          ),
        ],
      );
      final event = SentryEvent(
        exceptions: [
          SentryException(
            type: 'StateError',
            value: 'PRIVATE_DRAFT',
            stackTrace: stack,
          ),
        ],
        contexts: Contexts()..['app'] = {'app_version': '1.2.3'},
        request: SentryRequest(url: 'https://private.test/?token=SECRET'),
        user: SentryUser(email: 'private@example.test'),
      );
      final safe = await options.beforeSend!(event, Hint());
      expect(identical(safe, event), isTrue);
      expect(identical(safe!.exceptions!.single.stackTrace, stack), isTrue);
      expect(safe.contexts['app'], {'app_version': '1.2.3'});
      expect(safe.exceptions!.single.value, 'Operation failed');
      expect(safe.request, isNull);
      expect(safe.user, isNull);
    },
  );
}
