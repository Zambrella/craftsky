import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/platform_log.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

void main() {
  test(
    'default console sink prints every protected stack frame on short lines',
    () {
      final output = <String>[];
      final stack = StackTrace.fromString(
        [
          for (var i = 0; i < 240; i++)
            '#$i load$i (package:craftsky_app/source.dart:${i + 1}:3)',
          '<asynchronous suspension>',
          '#240 token=PRIVATE_TOKEN (/Users/private/source.dart:241:3)',
        ].join('\n'),
      );
      runZoned(
        () => DiagnosticEmitter().emitLocal(
          LogRecord(
            Level.SEVERE,
            'Read failed',
            'Test',
            StateError('PRIVATE_BODY'),
            stack,
          ),
        ),
        zoneSpecification: ZoneSpecification(
          print: (_, _, _, line) => output.add(line),
        ),
      );

      expect(output.length, greaterThan(240));
      expect(output.every((line) => utf8.encode(line).length <= 800), isTrue);
      final record = jsonDecode(output.join('\n')) as Map<String, dynamic>;
      final frames = record['stack'] as List<dynamic>;
      expect(frames, hasLength(242));
      expect(frames[239], containsPair('function', 'load239'));
      expect(frames[240], containsPair('frame', '<asynchronous suspension>'));
      expect(output.join(), isNot(contains('PRIVATE_TOKEN')));
      expect(output.join(), isNot(contains('PRIVATE_BODY')));
      expect(output.join(), isNot(contains('/Users/private')));
    },
  );

  test('wraps unusually long Unicode fields without losing any characters', () {
    final record = jsonEncode({'frame': List.filled(1500, '🧶').join()});
    final lines = platformDiagnosticLines(record).toList();
    expect(lines.every((line) => utf8.encode(line).length <= 800), isTrue);
    expect(jsonDecode(lines.join()), jsonDecode(record));
  });
}
