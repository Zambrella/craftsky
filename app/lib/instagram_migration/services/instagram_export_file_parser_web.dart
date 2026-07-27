import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';

Future<InstagramImportParseResult> parseInstagramExportPath(
  String path, {
  required int maxArchiveEntries,
  required int maxCentralDirectoryBytes,
}) {
  throw const InstagramImportParseException(
    InstagramImportParseErrorCode.unsupportedFormat,
  );
}
