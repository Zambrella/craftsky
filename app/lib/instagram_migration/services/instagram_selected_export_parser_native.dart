import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser.dart';
import 'package:file_selector/file_selector.dart' show XFile;

Future<InstagramImportParseResult> parseSelectedInstagramExport(XFile file) =>
    const InstagramExportFileParser().parsePath(file.path);
