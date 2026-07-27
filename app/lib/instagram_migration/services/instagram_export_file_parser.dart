import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser_web.dart'
    if (dart.library.io) 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser_native.dart'
    as platform;
import 'package:craftsky_app/instagram_migration/services/instagram_export_limits.dart';

final class InstagramExportFileParser {
  const InstagramExportFileParser()
    : _maxArchiveEntries = instagramMaxArchiveEntries,
      _maxCentralDirectoryBytes = instagramMaxCentralDirectoryBytes;

  /// Tightens, but can never loosen, the production limits for boundary tests.
  const InstagramExportFileParser.withLimits({
    required int maxArchiveEntries,
    required int maxCentralDirectoryBytes,
  }) : _maxArchiveEntries = maxArchiveEntries < 1
           ? 1
           : maxArchiveEntries > instagramMaxArchiveEntries
           ? instagramMaxArchiveEntries
           : maxArchiveEntries,
       _maxCentralDirectoryBytes = maxCentralDirectoryBytes < 1
           ? 1
           : maxCentralDirectoryBytes > instagramMaxCentralDirectoryBytes
           ? instagramMaxCentralDirectoryBytes
           : maxCentralDirectoryBytes;

  static const String followingEntryPath = instagramFollowingEntryPath;
  static const int maxArchiveEntries = instagramMaxArchiveEntries;
  static const int maxCentralDirectoryBytes = instagramMaxCentralDirectoryBytes;

  final int _maxArchiveEntries;
  final int _maxCentralDirectoryBytes;

  Future<InstagramImportParseResult> parsePath(String path) =>
      platform.parseInstagramExportPath(
        path,
        maxArchiveEntries: _maxArchiveEntries,
        maxCentralDirectoryBytes: _maxCentralDirectoryBytes,
      );
}
