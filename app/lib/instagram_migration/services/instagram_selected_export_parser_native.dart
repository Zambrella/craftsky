import 'dart:io';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// Document picks on Android and iOS are app-owned temporary copies.
/// Parse by path so large ZIPs never need to enter Dart memory in full.
Future<InstagramImportParseResult> parseSelectedInstagramExport(
  PlatformFile file,
) async {
  final path = file.path;
  if (path == null) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  }
  try {
    return await const InstagramExportFileParser().parsePath(path);
  } finally {
    if (defaultTargetPlatform == TargetPlatform.android ||
        defaultTargetPlatform == TargetPlatform.iOS) {
      try {
        await File(path).delete();
      } on FileSystemException {
        // Cache eviction may already have removed the temporary copy.
      }
    }
  }
}

Future<void> clearExportPickerCache() async {
  // Android clears only its file_picker directory. The iOS implementation
  // clears all NSTemporaryDirectory contents, including unrelated app files;
  // on iOS we only delete the selected copy above.
  if (defaultTargetPlatform != TargetPlatform.android) {
    return;
  }
  try {
    await FilePicker.clearTemporaryFiles();
  } on PlatformException {
    // Best effort: cleanup must not replace the pick or parse outcome.
  }
}
