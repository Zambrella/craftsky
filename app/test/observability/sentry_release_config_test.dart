import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Sentry release symbolication config and docs are present', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    final docs = File(
      '../docs/changes/2026-07-03-flutter-error-handling-sentry/release-symbolication.md',
    );

    expect(pubspec, contains('sentry_dart_plugin:'));
    expect(pubspec, contains('upload_debug_symbols: true'));
    expect(pubspec, contains('upload_source_maps: true'));
    expect(pubspec, contains('upload_sources: true'));
    expect(pubspec, isNot(contains('auth_token:')));

    expect(docs.existsSync(), isTrue);
    final docText = docs.readAsStringSync();
    expect(docText, contains('SENTRY_AUTH_TOKEN'));
    expect(docText, contains('--split-debug-info'));
    expect(docText, contains('--obfuscate'));
    expect(docText, contains('dart run sentry_dart_plugin'));
  });
}
