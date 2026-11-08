import 'dart:async';
import 'dart:io';

import 'package:craftsky_app/instagram_migration/services/instagram_export_file_picker.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

base class _SelectedFile extends PlatformFile {
  new(this.uri);
  @override
  final Uri uri;
  bool bytesRead = false;
  @override
  String get name => 'export.json';
  @override
  Future<int?> length() async => null;
  @override
  Future<Uint8List> readAsBytes() async {
    bytesRead = true;
    throw StateError('Native picks must be read by path');
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _Picker extends FilePickerPlatform {
  Future<PlatformFile?> Function() select = () async => null;
  int clears = 0;
  bool failCleanup = false;
  @override
  Future<PlatformFile?> pickFile({
    String? dialogTitle,
    String? initialDirectory,
    FileType type = FileType.any,
    List<String>? allowedExtensions,
    void Function(FilePickerStatus)? onFileLoading,
    int compressionQuality = 0,
    AndroidOptions androidOptions = const AndroidOptions(),
    DarwinOptions darwinOptions = const DarwinOptions(),
    WindowsOptions windowsOptions = const WindowsOptions(),
    LinuxOptions linuxOptions = const LinuxOptions(),
    WebOptions webOptions = const WebOptions(),
  }) {
    expect(type, FileType.custom);
    expect(allowedExtensions, ['json', 'zip']);
    expect(compressionQuality, 0);
    return select();
  }

  @override
  Future<void> clearTemporaryFiles() async {
    clears++;
    if (failCleanup) throw PlatformException(code: 'cleanup_failed');
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late _Picker picker;
  late ProviderContainer container;
  late FilePickerPlatform previous;
  setUp(() {
    previous = FilePickerPlatform.instance;
    picker = _Picker();
    FilePickerPlatform.instance = picker;
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    container = ProviderContainer();
  });
  tearDown(() {
    container.dispose();
    FilePickerPlatform.instance = previous;
    debugDefaultTargetPlatformOverride = null;
  });
  test(
    'cancellation returns no import and clears orphaned cache files',
    () async {
      expect(await container.read(instagramExportFilePickerProvider)(), isNull);
      expect(picker.clears, 1);
    },
  );
  for (final platform in [TargetPlatform.android, TargetPlatform.iOS]) {
    for (final valid in [true, false]) {
      test(
        '$platform removes its picked copy after parse success=$valid',
        () async {
          debugDefaultTargetPlatformOverride = platform;
          final directory = await Directory.systemTemp.createTemp(
            'export-picker',
          );
          addTearDown(() => directory.delete(recursive: true));
          final file = await File('${directory.path}/export.json')
              .writeAsString(
                valid
                    ? '{"relationships_following":[]}'
                    : 'INVALID_PRIVATE_JSON',
              );
          final selected = _SelectedFile(file.uri);
          picker.select = () async => selected;
          final result = container.read(instagramExportFilePickerProvider)();
          if (valid) {
            expect((await result)!.entries, isEmpty);
          } else {
            await expectLater(result, throwsException);
          }
          expect(selected.bytesRead, isFalse);
          expect(file.existsSync(), isFalse);
          expect(picker.clears, platform == TargetPlatform.android ? 1 : 0);
        },
      );
    }
  }
  test('picker failures survive cleanup failures and allow a retry', () async {
    final failure = PlatformException(code: 'unknown_path');
    picker
      ..failCleanup = true
      ..select = () async => throw failure;
    await expectLater(
      container.read(instagramExportFilePickerProvider)(),
      throwsA(same(failure)),
    );
    expect(picker.clears, 1);
    picker.select = () async => null;
    expect(await container.read(instagramExportFilePickerProvider)(), isNull);
  });
  test('concurrent calls cannot clean another import cache', () async {
    final selection = Completer<PlatformFile?>();
    picker.select = () => selection.future;
    final pick = container.read(instagramExportFilePickerProvider);
    final first = pick();
    await expectLater(pick(), throwsStateError);
    expect(picker.clears, 0);
    selection.complete(null);
    expect(await first, isNull);
    expect(picker.clears, 1);
  });
}
