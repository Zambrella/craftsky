import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  // UT-007 / FR-010, FR-012 / AC-014, AC-016
  test('schema two accepts independent controls atomically', () {
    const raw = '''
{"schemaVersion":2,"revision":"D1",
      "maintenance":{"title":"Break","description":"Back soon"},
      "minimumVersions":{"ios":"1.8.0"},
      "message":{"revision":"A","minVersion":"1.8.0",
        "title":"News","description":"Hello",
        "cta":{"text":"Settings","destination":"settings"}}}''';
    expect(
      () => ServiceStatusDocument.decode(utf8.encode(raw)),
      returnsNormally,
    );
    expect(
      () => ServiceStatusDocument.decode(
        utf8.encode(
          raw.replaceFirst('"minVersion":"1.8.0"', '"minVersion":null'),
        ),
      ),
      throwsFormatException,
    );
  });

  test(
    'mapper round trip strips unknown controls and preserves plain text',
    () {
      final document = ServiceStatusDocument.decode(
        utf8.encode(
          jsonEncode({
            'schemaVersion': 2,
            'revision': 'A',
            'maintenance': {
              'title': '<script>plain</script>',
              'description': 'Body',
              'estimatedRecoveryAt': '2026-10-09T14:30:00+02:00',
            },
            'apiUrl': 'https://example.invalid',
          }),
        ),
      );
      expect(document.maintenance!.title, '<script>plain</script>');
      expect(document.toMap().containsKey('apiUrl'), isFalse);
      expect(
        ServiceStatusDocument.decode(utf8.encode(document.toJson()))
            .maintenance!
            .estimatedRecoveryAt,
        DateTime.utc(2026, 10, 9, 12, 30),
      );
      expect(document.toString(), isNot(contains('script')));
    },
  );
  test('shared literal contract corpus', () {
    final corpus = jsonDecode(
      File(
        '../service-status/contract-fixtures.json',
      ).readAsStringSync(),
    ) as List<dynamic>;
    for (final entry in corpus.cast<Map<String, dynamic>>()) {
      final bytes = utf8.encode(jsonEncode(entry['document']));
      if (entry['accepted'] == true) {
        expect(
          () => ServiceStatusDocument.decode(bytes),
          returnsNormally,
          reason: entry['name'] as String,
        );
      } else {
        expect(
          () => ServiceStatusDocument.decode(bytes),
          throwsFormatException,
          reason: entry['name'] as String,
        );
      }
    }
  });

  test('enforces literal byte and Unicode scalar limits', () {
    Map<String, Object> notice(String title, String message) => {
      'schemaVersion': 2,
      'revision': 'A',
      'maintenance': {'title': title, 'description': message},
    };
    List<int> encode(Map<String, Object> value) =>
        utf8.encode(jsonEncode(value));
    expect(
      () =>
          ServiceStatusDocument.decode(encode(notice('🧶' * 160, 'x' * 2400))),
      returnsNormally,
    );
    for (final value in [notice('🧶' * 161, 'x'), notice('x', 'x' * 2401)]) {
      expect(
        () => ServiceStatusDocument.decode(encode(value)),
        throwsFormatException,
      );
    }
    const minimal = '{"schemaVersion":2,"revision":"A"}';
    expect(
      () => ServiceStatusDocument.decode(utf8.encode(minimal.padRight(16384))),
      returnsNormally,
    );
    expect(
      () => ServiceStatusDocument.decode(utf8.encode(minimal.padRight(16385))),
      throwsFormatException,
    );
    for (final raw in ['<html>Error</html>', 'null', '[]', '{', '42']) {
      expect(
        () => ServiceStatusDocument.decode(utf8.encode(raw)),
        throwsFormatException,
      );
    }
    expect(() => ServiceStatusDocument.decode([0xff]), throwsFormatException);
  });
}
