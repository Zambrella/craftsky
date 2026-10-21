import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_web.dart'
    if (dart.library.io) 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_native.dart'
    as platform;
import 'package:file_picker/file_picker.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef InstagramExportFilePicker =
    Future<InstagramImportParseResult?> Function();

// This import flow is the only file_picker consumer. Serialize it so cache
// cleanup cannot remove another import's copy while its parser is reading it.
bool _pickerActive = false;

final instagramExportFilePickerProvider = Provider<InstagramExportFilePicker>(
  (_) => () async {
    if (_pickerActive) throw StateError('Instagram export picker is active');
    _pickerActive = true;
    try {
      final file = await FilePicker.pickFile(
        type: FileType.custom,
        allowedExtensions: const ['json', 'zip'],
      );
      if (file == null) return null;
      return await platform.parseSelectedInstagramExport(file);
    } finally {
      try {
        // Also removes partial copies from failed picks or interrupted imports.
        await platform.clearExportPickerCache();
      } finally {
        _pickerActive = false;
      }
    }
  },
);
