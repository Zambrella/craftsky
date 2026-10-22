import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:file_picker/file_picker.dart';

Future<InstagramImportParseResult> parseSelectedInstagramExport(
  PlatformFile file,
) async {
  final length = await file.length();
  if (length == null || length > InstagramImportParser.maxFileBytes) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.fileTooLarge,
    );
  }
  return const InstagramImportParser().parseJson(await file.readAsBytes());
}

Future<void> clearExportPickerCache() async {}
