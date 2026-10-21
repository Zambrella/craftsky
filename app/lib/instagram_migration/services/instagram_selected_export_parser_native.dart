import 'dart:io';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser.dart';
import 'package:file_selector/file_selector.dart' show XFile;
import 'package:flutter/services.dart';

Future<InstagramImportParseResult> parseSelectedInstagramExport(XFile file) =>
    const InstagramExportFileParser().parsePath(file.path);

/// Android's generic file selector eagerly allocates the entire export twice.
/// Stream it into cache natively, parse by path, then remove the private copy.
Future<InstagramImportParseResult?> pickAndroidInstagramExport() async {
  const channel = MethodChannel('social.craftsky.app/instagram_export');
  final path = await channel.invokeMethod<String>('open');
  if (path == null) return null;
  try {
    return await const InstagramExportFileParser().parsePath(path);
  } finally {
    try {
      await File(path).delete();
    } on FileSystemException {
      // Cache eviction may already have removed the temporary copy.
    }
  }
}
