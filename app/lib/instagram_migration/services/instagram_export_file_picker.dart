import 'dart:io';

import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_post_import_workspace.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_selected_export_parser_native.dart'
    as platform;
import 'package:file_picker/file_picker.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path/path.dart' as path;

typedef InstagramExportFilePicker =
    Future<InstagramImportParseResult?> Function();
typedef InstagramPostExportFilePicker =
    Future<InstagramPostExportFile?> Function();

// Both Instagram flows share the plug-in's cache. Only one picker can use it
// until the selected file has been parsed or moved out of that cache.
bool _pickerActive = false;

final instagramExportFilePickerProvider = Provider<InstagramExportFilePicker>(
  (_) =>
      () => _pickExport(
        const ['json', 'zip'],
        platform.parseSelectedInstagramExport,
      ),
);

final instagramPostExportFilePickerProvider =
    Provider<InstagramPostExportFilePicker>(
      (_) =>
          () => _pickExport(const ['zip'], _retainArchive),
    );

Future<T?> _pickExport<T>(
  List<String> extensions,
  Future<T> Function(PlatformFile file) useFile,
) async {
  if (_pickerActive) throw StateError('Instagram export picker is active');
  _pickerActive = true;
  try {
    final file = await FilePicker.pickFile(
      type: FileType.custom,
      allowedExtensions: extensions,
    );
    if (file == null) return null;
    return await useFile(file);
  } finally {
    try {
      // Also removes partial copies from failed picks or interrupted imports.
      await platform.clearExportPickerCache();
    } finally {
      _pickerActive = false;
    }
  }
}

/// One app-owned ZIP in a dedicated temporary directory. The review page owns
/// the lifetime of this copy, separate from the shared document-picker cache.
final class InstagramPostExportFile {
  const new({required this.directory, required this.name});

  final Directory directory;
  final String name;
  String get archivePath => path.join(directory.path, 'archive.zip');

  Future<void> dispose() async {
    try {
      await directory.delete(recursive: true);
    } on FileSystemException {
      // The operating system may already have evicted the temporary copy.
    }
  }
}

Future<InstagramPostExportFile> _retainArchive(PlatformFile file) async {
  final sourcePath = file.path;
  if (sourcePath == null) {
    throw const FileSystemException('Selected export is unavailable');
  }
  // Android and iOS document picks are app-owned copies. Rename on the same
  // volume to avoid reading or duplicating a potentially large export.
  final source = File(sourcePath);
  InstagramPostExportFile? archive;
  try {
    final directory = await instagramPostImportWorkspace
        .createReviewDirectory();
    archive = InstagramPostExportFile(directory: directory, name: file.name);
    await source.rename(archive.archivePath);
    return archive;
  } on Object {
    await archive?.dispose();
    try {
      await source.delete();
    } on FileSystemException {
      // The platform may already have cleaned its copy.
    }
    rethrow;
  }
}
