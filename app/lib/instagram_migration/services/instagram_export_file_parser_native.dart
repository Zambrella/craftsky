import 'dart:io';
import 'dart:isolate';
import 'dart:typed_data';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_limits.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_zip_reader.dart';

Future<InstagramImportParseResult> parseInstagramExportPath(
  String path, {
  required int maxArchiveEntries,
  required int maxCentralDirectoryBytes,
}) => Isolate.run(
  () => _parseInstagramExportPathSync(
    path,
    maxArchiveEntries,
    maxCentralDirectoryBytes,
  ),
);

InstagramImportParseResult _parseInstagramExportPathSync(
  String path,
  int maxArchiveEntries,
  int maxCentralDirectoryBytes,
) {
  try {
    final input = File(path).openSync();
    try {
      final length = input.lengthSync();
      if (_hasZipSignature(input.readSync(4))) {
        return _parseZip(
          path,
          length,
          maxArchiveEntries,
          maxCentralDirectoryBytes,
        );
      }
      if (length > InstagramImportParser.maxFileBytes) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.fileTooLarge,
        );
      }
      input.setPositionSync(0);
      final bytes = input.readSync(length);
      if (bytes.length != length || input.readByteSync() != -1) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }
      return const InstagramImportParser().parseJson(bytes);
    } finally {
      input.closeSync();
    }
  } on InstagramImportParseException {
    rethrow;
  } on Object {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  }
}

bool _hasZipSignature(Uint8List bytes) =>
    bytes.length >= 4 &&
    bytes[0] == 0x50 &&
    bytes[1] == 0x4b &&
    ((bytes[2] == 0x03 && bytes[3] == 0x04) ||
        (bytes[2] == 0x05 && bytes[3] == 0x06) ||
        (bytes[2] == 0x07 && bytes[3] == 0x08));

InstagramImportParseResult _parseZip(
  String path,
  int fileLength,
  int maxArchiveEntries,
  int maxCentralDirectoryBytes,
) {
  final directory = readInstagramZipDirectory(
    path,
    fileLength,
    maxArchiveEntries,
    maxCentralDirectoryBytes,
  );
  final targets = directory
      .where((entry) => entry.name == instagramFollowingEntryPath)
      .toList(growable: false);
  if (targets.length != 1) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.unsupportedShape,
    );
  }
  final bytes = targets.single.readBytes(
    path,
    InstagramImportParser.maxFileBytes,
    capCompressionRatio: false,
  );
  return const InstagramImportParser().parseJson(bytes);
}
