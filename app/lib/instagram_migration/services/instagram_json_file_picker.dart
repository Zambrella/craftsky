import 'dart:typed_data';

import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:file_selector/file_selector.dart' as file_selector;
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef InstagramJsonFilePicker = Future<Uint8List?> Function();

final instagramJsonFilePickerProvider = Provider<InstagramJsonFilePicker>(
  (_) => () async {
    const jsonFiles = file_selector.XTypeGroup(
      label: 'JSON',
      extensions: ['json'],
    );
    final file = await file_selector.openFile(
      acceptedTypeGroups: const [jsonFiles],
    );
    if (file == null) return null;
    final length = await file.length();
    if (length > InstagramImportParser.maxFileBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    return file.readAsBytes();
  },
);
