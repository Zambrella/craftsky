import 'dart:convert';
import 'dart:io';
import 'dart:isolate';
import 'dart:math';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:characters/characters.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_limits.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_zip_envelope.dart';
import 'package:crypto/crypto.dart';
import 'package:image/image.dart' as image;
import 'package:image_size_getter/image_size_getter.dart' as dimensions;
import 'package:mime/mime.dart';
import 'package:path/path.dart' as path;
import 'package:unorm_dart/unorm_dart.dart' as unorm;

part 'instagram_post_archive_records.dart';
part 'instagram_post_archive_zip.dart';

const int _maxCandidateBytes = 32 * 1024 * 1024;
const int _maxCombinedCandidateBytes = 128 * 1024 * 1024;
const _maxPosts = 25000;
const _maxPhotoBytes = 40000000;
const _maxImageSide = 12000;
const _maxImagePixels = 25000000;

enum InstagramArchiveMediaKind { photo, video, unsupported }

enum InstagramArchiveWarning {
  captionTooLong,
  tooManyPhotos,
  multipleVideos,
  mixedMedia,
  unsupportedMedia,
  unavailableMedia,
  oversizedMedia,
}

enum InstagramPostArchiveError {
  invalidArchive,
  archiveTooLarge,
  unsupportedArchive,
  noPosts,
  unsupportedPosts,
  archiveChanged,
  mediaUnavailable,
  mediaTooLarge,
  unsupportedMedia,
  invalidSelection,
}

final class InstagramPostArchiveException implements Exception {
  const new(this.code);
  final InstagramPostArchiveError code;

  @override
  String toString() => 'InstagramPostArchiveException(${code.name})';
}

final class InstagramArchiveMedia {
  const new _(this.kind, this._entry);

  final InstagramArchiveMediaKind kind;
  final _ZipEntry? _entry;

  bool get available => _entry != null;
}

final class InstagramArchivePost {
  const new _({
    required this.sourceKey,
    required this.createdAt,
    required this.caption,
    required this.captionLength,
    required this.media,
    required this.warnings,
  });

  final String sourceKey;
  final DateTime createdAt;
  final String caption;
  final int captionLength;
  final List<InstagramArchiveMedia> media;
  final List<InstagramArchiveWarning> warnings;
}

/// Metadata and offsets only. The caller owns the selected archive file.
final class InstagramPostArchive {
  const new _(
    this._path,
    this._length,
    this._modified,
    this.posts,
    this.skipped,
  );

  final String _path;
  final int _length;
  final DateTime _modified;
  final List<InstagramArchivePost> posts;
  final int skipped;
}

final class InstagramExtractedMedia {
  const new _(this.photos, this.video, this._directory, this.unavailableCount);

  final List<String> photos;
  final String? video;
  final String? _directory;
  final int unavailableCount;

  Future<void> dispose() async {
    final directory = _directory;
    if (directory == null) return;
    try {
      await Directory(directory).delete(recursive: true);
    } on FileSystemException {
      // Temporary files may already have been evicted.
    }
  }
}

/// Inspect metadata and unpack only selected files, away from the UI isolate.
final class InstagramPostArchiveReader {
  const new();

  Future<InstagramPostArchive> inspect(String path) =>
      Isolate.run(() => _readArchive(path));

  /// [mediaIndices] address the original ordered list in [post].
  /// Partial extraction counts selected entries that could not be decoded.
  Future<InstagramExtractedMedia> extract(
    InstagramPostArchive archive,
    InstagramArchivePost post,
    List<int> mediaIndices, {
    String? temporaryDirectory,
    bool allowPartial = false,
  }) {
    final selected = _select(archive, post, mediaIndices);
    final source = _source(archive);
    return _extractInIsolate(
      source,
      selected,
      temporaryDirectory,
      allowPartial,
    );
  }

  Future<Uint8List?> preview(
    InstagramPostArchive archive,
    InstagramArchivePost post,
  ) async {
    if (!archive.posts.contains(post)) return null;
    for (final item in post.media) {
      final entry = item._entry;
      if (item.kind != InstagramArchiveMediaKind.photo || entry == null) {
        continue;
      }
      return await _previewInIsolate(_source(archive), entry);
    }
    return null;
  }
}

typedef _ArchiveSource = ({String path, int length, DateTime modified});

_ArchiveSource _source(InstagramPostArchive archive) =>
    (path: archive._path, length: archive._length, modified: archive._modified);

List<InstagramArchiveMedia> _select(
  InstagramPostArchive archive,
  InstagramArchivePost post,
  List<int> indices,
) {
  if (!archive.posts.contains(post) ||
      indices.length > defaultMaxPostImages ||
      indices.toSet().length != indices.length ||
      indices.any((index) => index < 0 || index >= post.media.length)) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.invalidSelection,
    );
  }
  final ordered = indices.toList()..sort();
  final selected = ordered.map((index) => post.media[index]).toList();
  if (selected.any((item) => item.kind == InstagramArchiveMediaKind.video) &&
      selected.length != 1) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.invalidSelection,
    );
  }
  if (selected.any((item) => !item.available)) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.mediaUnavailable,
    );
  }
  return selected;
}

Future<InstagramExtractedMedia> _extractInIsolate(
  _ArchiveSource source,
  List<InstagramArchiveMedia> selected,
  String? temporaryDirectory,
  bool allowPartial,
) => Isolate.run(
  () => _guardArchive(
    () => _extract(source, selected, temporaryDirectory, allowPartial),
  ),
);

Future<Uint8List?> _previewInIsolate(
  _ArchiveSource source,
  _ZipEntry entry,
) => Isolate.run(() {
  try {
    _checkSource(source);
    final photo = _readPhoto(source.path, entry);
    final decoded = _decodePhoto(photo.bytes, photo.extension);
    final orientation = decoded.exif.imageIfd.orientation ?? 1;
    final rotated = orientation >= 5 && orientation <= 8;
    final width = rotated ? decoded.height : decoded.width;
    final height = rotated ? decoded.width : decoded.height;
    final scale = min(1, 160 / max(width, height));
    final thumbnail = image.copyResize(
      decoded,
      width: max(1, (width * scale).round()),
      height: max(1, (height * scale).round()),
    );
    return image.encodeJpg(thumbnail, quality: 75);
  } on Object {
    // A thumbnail is optional. Extraction still fails closed for bad media.
    return null;
  }
});

T _guardArchive<T>(T Function() read) {
  try {
    return read();
  } on InstagramPostArchiveException {
    rethrow;
  } on Object {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.invalidArchive,
    );
  }
}

InstagramPostArchive _readArchive(String filePath) => _guardArchive(() {
  final stat = FileStat.statSync(filePath);
  final entries = _readDirectory(filePath, stat.size);
  final byName = <String, _ZipEntry>{};
  final ambiguousNames = <String>{};
  final candidates = <({String wrapper, bool detailed, _ZipEntry entry})>[];
  for (final entry in entries) {
    if (!_isSafePath(entry.name)) continue;
    final key = _comparisonPath(entry.name);
    final previous = byName[key];
    final metadata = _matchMetadataPath(entry.name);
    if (previous != null) {
      ambiguousNames.add(key);
      if (metadata != null || _matchMetadataPath(previous.name) != null) {
        throw const InstagramPostArchiveException(
          InstagramPostArchiveError.invalidArchive,
        );
      }
      continue;
    }
    byName[key] = entry;
    if (metadata != null) {
      candidates.add((
        wrapper: metadata.wrapper,
        detailed: metadata.detailed,
        entry: entry,
      ));
    }
  }

  if (candidates.isEmpty) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.noPosts,
    );
  }
  final wrapper = candidates.first.wrapper;
  if (candidates.any((candidate) => candidate.wrapper != wrapper)) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.invalidArchive,
    );
  }

  var totalBytes = 0;
  var skipped = 0;
  var recognized = 0;
  final records = <_SourcePost>[];
  for (final candidate in candidates) {
    totalBytes += candidate.entry.size;
    if (totalBytes > _maxCombinedCandidateBytes ||
        !candidate.entry.withinLimits(_maxCandidateBytes)) {
      throw const InstagramPostArchiveException(
        InstagramPostArchiveError.archiveTooLarge,
      );
    }
    final bytes = _readEntryBytes(filePath, candidate.entry);
    final data = jsonDecode(utf8.decode(bytes));
    final result = _parseRecords(data, detailed: candidate.detailed);
    records.addAll(result.posts);
    skipped += result.skipped;
    recognized += result.recognized;
    if (records.length + skipped > _maxPosts) {
      throw const InstagramPostArchiveException(
        InstagramPostArchiveError.archiveTooLarge,
      );
    }
  }

  if (recognized == 0) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.unsupportedPosts,
    );
  }
  final merged = _mergeRecords(records);
  skipped += merged.skipped;
  final latest = DateTime.now().toUtc().add(const Duration(days: 1));
  final posts = <InstagramArchivePost>[];
  final fingerprints = <String>{};
  for (final post in merged.posts) {
    final createdAt = _normalizeDate(post.timestamp, latest);
    if (createdAt == null) {
      skipped++;
      continue;
    }
    final key = _fingerprint(createdAt, post);
    if (!fingerprints.add(key)) {
      throw const InstagramPostArchiveException(
        InstagramPostArchiveError.invalidArchive,
      );
    }

    final caption = _decodeCaption(post.caption);
    final captionLength = caption.characters.length;
    final photoCount = post.media
        .where((media) => media.kind == InstagramArchiveMediaKind.photo)
        .length;
    final videoCount = post.media
        .where((media) => media.kind == InstagramArchiveMediaKind.video)
        .length;
    final warnings = <InstagramArchiveWarning>{
      if (captionLength > 2000 || utf8.encode(caption).length > 20000)
        InstagramArchiveWarning.captionTooLong,
      if (photoCount > defaultMaxPostImages)
        InstagramArchiveWarning.tooManyPhotos,
      if (videoCount > 1) InstagramArchiveWarning.multipleVideos,
      if (photoCount > 0 && videoCount > 0) InstagramArchiveWarning.mixedMedia,
    };
    final media = <InstagramArchiveMedia>[];
    for (final sourceMedia in post.media) {
      final name = '$wrapper${sourceMedia.path}';
      final canonicalName = _comparisonPath(name);
      final entry = byName[canonicalName];
      _ZipEntry? available;
      if (sourceMedia.kind == InstagramArchiveMediaKind.unsupported) {
        warnings.add(InstagramArchiveWarning.unsupportedMedia);
      } else if (entry == null ||
          entry.name != name ||
          ambiguousNames.contains(canonicalName)) {
        warnings.add(InstagramArchiveWarning.unavailableMedia);
      } else if (!entry.withinLimits(
        sourceMedia.kind == InstagramArchiveMediaKind.video
            ? maxVideoSourceBytes
            : _maxPhotoBytes,
      )) {
        warnings.add(InstagramArchiveWarning.oversizedMedia);
      } else {
        available = entry;
      }
      media.add(InstagramArchiveMedia._(sourceMedia.kind, available));
    }
    posts.add(
      InstagramArchivePost._(
        sourceKey: key,
        createdAt: createdAt,
        caption: caption,
        captionLength: captionLength,
        media: List.unmodifiable(media),
        warnings: List.unmodifiable(warnings),
      ),
    );
  }
  _checkSource((path: filePath, length: stat.size, modified: stat.modified));
  return InstagramPostArchive._(
    filePath,
    stat.size,
    stat.modified,
    List.unmodifiable(posts),
    skipped,
  );
});

String _comparisonPath(String value) => unorm.nfkc(value).toLowerCase();

void _checkSource(_ArchiveSource source) {
  final stat = FileStat.statSync(source.path);
  if (stat.type != FileSystemEntityType.file ||
      stat.size != source.length ||
      stat.modified != source.modified) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.archiveChanged,
    );
  }
}

InstagramExtractedMedia _extract(
  _ArchiveSource source,
  List<InstagramArchiveMedia> selected,
  String? temporaryDirectory,
  bool allowPartial,
) {
  _checkSource(source);
  if (selected.isEmpty) {
    return const InstagramExtractedMedia._([], null, null, 0);
  }
  final directory = temporaryDirectory == null
      ? Directory.systemTemp.createTempSync('craftsky-instagram-')
      : Directory(temporaryDirectory).createTempSync('media-');
  try {
    final photos = <String>[];
    String? video;
    var unavailableCount = 0;
    for (final item in selected) {
      final entry = item._entry!;
      try {
        if (item.kind == InstagramArchiveMediaKind.video) {
          video = _extractVideo(source.path, entry, directory);
        } else {
          final photo = _readPhoto(source.path, entry);
          final file = File(
            path.join(
              directory.path,
              'photo-${photos.length + 1}.${photo.extension}',
            ),
          )..writeAsBytesSync(photo.bytes);
          photos.add(file.path);
        }
      } on Object catch (error) {
        // The selected member can be damaged or unsupported while other
        // members are usable. File I/O and archive identity failures still
        // abort the extraction.
        if (!allowPartial ||
            (error is! InstagramPostArchiveException &&
                error is! ArchiveException &&
                error is! FormatException)) {
          rethrow;
        }
        unavailableCount++;
      }
    }
    _checkSource(source);
    return InstagramExtractedMedia._(
      List.unmodifiable(photos),
      video,
      directory.path,
      unavailableCount,
    );
  } on Object {
    try {
      directory.deleteSync(recursive: true);
    } on FileSystemException {
      // Preserve the original extraction error.
    }
    rethrow;
  }
}

String _extractVideo(String archive, _ZipEntry entry, Directory directory) {
  final temp = File(path.join(directory.path, 'video'));
  final output = _EntryOutput(
    entry.size,
    file: temp.openSync(mode: FileMode.write),
  );
  try {
    _readEntry(archive, entry, output);
  } finally {
    output.closeSync();
  }
  final input = temp.openSync();
  final Uint8List header;
  try {
    header = input.readSync(64);
  } finally {
    input.closeSync();
  }
  final extension = switch (detectVideoSourceMimeType(header)) {
    'video/mp4' => 'mp4',
    'video/quicktime' => 'mov',
    _ => throw const InstagramPostArchiveException(
      InstagramPostArchiveError.unsupportedMedia,
    ),
  };
  return temp.renameSync(path.join(directory.path, 'video.$extension')).path;
}

typedef _Photo = ({Uint8List bytes, String extension});

_Photo _readPhoto(String path, _ZipEntry entry) {
  final bytes = _readEntryBytes(path, entry);
  try {
    switch (lookupMimeType('', headerBytes: bytes)) {
      case 'image/jpeg':
        final size = const dimensions.JpegDecoder().getSize(
          dimensions.MemoryInput(bytes),
        );
        _validateImageDimensions(size.width, size.height);
        return (bytes: bytes, extension: 'jpg');
      case 'image/png':
        final size = const dimensions.PngDecoder().getSize(
          dimensions.MemoryInput(bytes),
        );
        _validateImageDimensions(size.width, size.height);
        final data = ByteData.sublistView(bytes);
        var offset = 8;
        while (offset + 12 <= bytes.length) {
          final length = data.getUint32(offset);
          final type = String.fromCharCodes(bytes, offset + 4, offset + 8);
          if (type == 'acTL') {
            throw const InstagramPostArchiveException(
              InstagramPostArchiveError.unsupportedMedia,
            );
          }
          if (type == 'IDAT' || type == 'IEND') break;
          offset += 12 + length;
        }
        return (bytes: bytes, extension: 'png');
      case 'image/webp':
        final decoder = image.WebPDecoder()..maxPixels = _maxImagePixels;
        final info = decoder.startDecode(bytes);
        if (info == null || info.hasAnimation) break;
        _validateImageDimensions(info.width, info.height);
        return (bytes: bytes, extension: 'webp');
    }
  } on InstagramPostArchiveException {
    rethrow;
  } on Object {
    // Do not let a codec error include contents or paths in an error report.
  }
  throw const InstagramPostArchiveException(
    InstagramPostArchiveError.unsupportedMedia,
  );
}

void _validateImageDimensions(int width, int height) {
  if (width <= 0 || height <= 0) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.unsupportedMedia,
    );
  }
  if (width > _maxImageSide ||
      height > _maxImageSide ||
      width * height > _maxImagePixels) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.mediaTooLarge,
    );
  }
}

image.Image _decodePhoto(Uint8List bytes, String extension) {
  final decoder = switch (extension) {
    'jpg' => image.JpegDecoder(maxPixels: _maxImagePixels),
    'webp' => image.WebPDecoder()..maxPixels = _maxImagePixels,
    _ => image.PngDecoder(maxPixels: _maxImagePixels),
  };
  final decoded = decoder.decode(bytes, frame: 0);
  if (decoded == null) {
    throw const InstagramPostArchiveException(
      InstagramPostArchiveError.unsupportedMedia,
    );
  }
  return decoded;
}
