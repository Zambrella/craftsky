import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Sentry and auth secrets are not committed in app source/config', () {
    final paths = [
      'pubspec.yaml',
      ...Directory(
        'lib',
      ).listSync(recursive: true).whereType<File>().map((file) => file.path),
    ];
    final forbidden = RegExp(
      r'(SENTRY_AUTH_TOKEN\s*=|Authorization:\s*Bearer|Cookie:\s*|pds[_-]?token|appview[_-]?session[_-]?token)',
      caseSensitive: false,
    );
    final offenders = <String>[];

    for (final path in paths) {
      final text = File(path).readAsStringSync();
      if (forbidden.hasMatch(text)) offenders.add(path);
    }

    expect(offenders, isEmpty);
  });
}
