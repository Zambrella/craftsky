import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:file_selector/file_selector.dart' show XFile;

Future<InstagramImportParseResult> parseSelectedInstagramExport(
  XFile file,
) async {
  final length = await file.length();
  if (length > InstagramImportParser.maxFileBytes) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.fileTooLarge,
    );
  }
  return const InstagramImportParser().parseJson(await file.readAsBytes());
}
