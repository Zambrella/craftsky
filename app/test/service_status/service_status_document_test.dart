import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('serializes the status document using the project mapper', () {
    final document = ServiceStatusDocument.decode(
      utf8.encode(
        jsonEncode({
          'schemaVersion': 1,
          'mode': 'maintenance',
          'revision': 'A',
          'title': 'Back soon',
          'message': 'Maintenance',
          'estimatedRecoveryAt': '2026-10-09T14:30:00+02:00',
          'unknownControl': true,
        }),
      ),
    );
    expect(document.toMap(), {
      'schemaVersion': 1,
      'mode': 'maintenance',
      'revision': 'A',
      'title': 'Back soon',
      'message': 'Maintenance',
      'estimatedRecoveryAt': '2026-10-09T12:30:00.000Z',
    });
    expect(
      ServiceStatusDocument.decode(utf8.encode(document.toJson())).revision,
      'A',
    );
  });
  // UT-007 / FR-002, RULE-003 / AC-005, AC-019.
  test('remote controls cannot alter required mode or plain prose', () {
    final document = ServiceStatusDocument.decode(
      utf8.encode(
        jsonEncode({
          'schemaVersion': 1,
          'mode': 'maintenance',
          'revision': 'plain',
          'title': '<script>alert(1)</script>',
          'message':
              '[Open](https://example.invalid) <a href="https://example.invalid">link</a>',
          'apiUrl': 'https://example.invalid',
          'action': {'url': 'https://example.invalid'},
          'featureFlags': {'skipAuth': true},
          'startsAt': '2999-01-01T00:00:00Z',
          'endsAt': '2000-01-01T00:00:00Z',
        }),
      ),
    );
    expect(document.mode, ServiceStatusMode.maintenance);
    expect(document.title, '<script>alert(1)</script>');
    expect(document.message, contains('<a href='));
    expect(document.estimatedRecoveryAt, isNull);
    expect(document.toString(), isNot(contains('example.invalid')));
  });

  // UT-001 / FR-002, FR-003 / AC-003, AC-004, AC-005.
  test('accepts maintenance without a publication time', () {
    final document = ServiceStatusDocument.decode(
      utf8.encode('''
{"schemaVersion":1,"mode":"maintenance","revision":"A",
 "title":"Back soon","message":"We are maintaining CraftSky."}
'''),
    );
    expect(document.mode, ServiceStatusMode.maintenance);
    expect(document.revision, 'A');
    expect(document.title, 'Back soon');
    expect(document.message, 'We are maintaining CraftSky.');
    expect(document.estimatedRecoveryAt, isNull);
  });

  test('accepts all modes and ignores optional remote controls', () {
    for (final mode in ServiceStatusMode.values) {
      final document = ServiceStatusDocument.decode(
        utf8.encode(
          jsonEncode({
            'schemaVersion': 1,
            'mode': mode.name,
            'revision': 'notice_B-2',
            'title': 'An update',
            'message': 'Public information',
            'estimatedRecoveryAt': '2026-10-09T14:30:00+02:00',
            'externalLink': 'https://example.invalid',
            'startsAt': '2099-01-01T00:00:00Z',
            'features': ['posts'],
          }),
        ),
      );
      expect(document.mode, mode);
      expect(document.estimatedRecoveryAt, DateTime.utc(2026, 10, 9, 12, 30));
    }
  });

  // UT-002 / FR-002, FR-003, NFR-001, RULE-003 / AC-005, AC-019.
  test('rejects unsupported schema before accepting a document', () {
    expect(
      () => ServiceStatusDocument.decode(
        utf8.encode(
          '{"schemaVersion":2,"mode":"normal","revision":"A"}',
        ),
      ),
      throwsFormatException,
    );
  });
  test('shared literal contract corpus', () {
    final corpus =
        jsonDecode(
              File(
                '../service-status/contract-fixtures.json',
              ).readAsStringSync(),
            )
            as List<dynamic>;
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
      'schemaVersion': 1,
      'mode': 'maintenance',
      'revision': 'A',
      'title': title,
      'message': message,
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
    const minimal = '{"schemaVersion":1,"mode":"normal","revision":"A"}';
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
