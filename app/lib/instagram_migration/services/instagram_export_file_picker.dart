import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_web.dart'
    if (dart.library.io) 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_native.dart'
    as platform;
import 'package:file_selector/file_selector.dart' as file_selector;
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef InstagramExportFilePicker =
    Future<InstagramImportParseResult?> Function();

final instagramExportFilePickerProvider = Provider<InstagramExportFilePicker>(
  (_) => () async {
    const exportFiles = file_selector.XTypeGroup(
      label: 'Instagram export',
      extensions: ['json', 'zip'],
      mimeTypes: ['application/json', 'application/zip'],
      uniformTypeIdentifiers: [
        'public.json',
        'public.zip-archive',
        'com.pkware.zip-archive',
      ],
    );
    final file = await file_selector.openFile(
      acceptedTypeGroups: const [exportFiles],
    );
    if (file == null) return null;
    return platform.parseSelectedInstagramExport(file);
  },
);
