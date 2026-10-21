import 'dart:io';

import 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_native.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('social.craftsky.app/instagram_export');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test('cancelled native selection returns no import', () async {
    messenger.setMockMethodCallHandler(channel, (_) async => null);
    expect(await pickAndroidInstagramExport(), isNull);
  });

  test(
    'successful path import removes the private native cache copy',
    () async {
      final directory = await Directory.systemTemp.createTemp(
        'export-picker-test',
      );
      addTearDown(() => directory.delete(recursive: true));
      final file = await File('${directory.path}/export.tmp')
          .writeAsString('{"relationships_following":[]}');
      messenger.setMockMethodCallHandler(channel, (_) async => file.path);
      final result = await pickAndroidInstagramExport();
      expect(result!.entries, isEmpty);
      expect(file.existsSync(), isFalse);
    },
  );

  test('parse failure removes the private native cache copy', () async {
    final directory = await Directory.systemTemp.createTemp(
      'export-picker-test',
    );
    addTearDown(() => directory.delete(recursive: true));
    final file = await File('${directory.path}/export.tmp')
        .writeAsString('PRIVATE_INVALID_JSON');
    messenger.setMockMethodCallHandler(channel, (call) async {
      expect(call.method, 'open');
      expect(call.arguments, isNull);
      return file.path;
    });
    await expectLater(pickAndroidInstagramExport(), throwsException);
    expect(file.existsSync(), isFalse);
  });
}
