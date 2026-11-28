import 'dart:io';
import 'dart:math';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';

final class InstagramZipMetadata {
  const new({
    required this.entryCount,
    required this.centralDirectoryOffset,
    required this.centralDirectorySize,
  });

  final int entryCount;
  final int centralDirectoryOffset;
  final int centralDirectorySize;
}

InstagramZipMetadata readInstagramZipMetadata(
  String path,
  int fileLength,
  int maxArchiveEntries,
  int maxCentralDirectoryBytes,
) {
  if (fileLength < 22) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  }
  final file = File(path).openSync();
  try {
    const maximumTailLength = 22 + 0xffff;
    final tailLength = min(fileLength, maximumTailLength);
    final tailOffset = fileLength - tailLength;
    file.setPositionSync(tailOffset);
    final tail = file.readSync(tailLength);
    final tailData = ByteData.sublistView(tail);
    var eocdIndex = -1;
    for (var index = tail.length - 22; index >= 0; index--) {
      if (_uint32(tailData, index) != ZipDirectory.eocdSignature) continue;
      final commentLength = _uint16(tailData, index + 20);
      if (index + 22 + commentLength == tail.length) {
        eocdIndex = index;
        break;
      }
    }
    if (eocdIndex < 0) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }

    final eocdOffset = tailOffset + eocdIndex;
    var diskNumber = _uint16(tailData, eocdIndex + 4);
    var centralDirectoryDisk = _uint16(tailData, eocdIndex + 6);
    var entriesOnDisk = _uint16(tailData, eocdIndex + 8);
    var entryCount = _uint16(tailData, eocdIndex + 10);
    var centralDirectorySize = _uint32(tailData, eocdIndex + 12);
    var centralDirectoryOffset = _uint32(tailData, eocdIndex + 16);
    var metadataStart = eocdOffset;

    final needsZip64 =
        diskNumber == 0xffff ||
        centralDirectoryDisk == 0xffff ||
        entriesOnDisk == 0xffff ||
        entryCount == 0xffff ||
        centralDirectorySize == 0xffffffff ||
        centralDirectoryOffset == 0xffffffff;
    if (needsZip64) {
      final locatorOffset = eocdOffset - ZipDirectory.zip64EocdLocatorSize;
      if (locatorOffset < 0) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }
      final locator = _readAt(
        file,
        locatorOffset,
        ZipDirectory.zip64EocdLocatorSize,
      );
      final locatorData = ByteData.sublistView(locator);
      if (_uint32(locatorData, 0) != ZipDirectory.zip64EocdLocatorSignature ||
          _uint32(locatorData, 4) != 0 ||
          _uint32(locatorData, 16) != 1) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }
      final zip64Offset = _uint64(locatorData, 8);
      final zip64Record = _readAt(
        file,
        zip64Offset,
        ZipDirectory.zip64EocdSize,
      );
      final zip64Data = ByteData.sublistView(zip64Record);
      final zip64RecordSize = _uint64(zip64Data, 4);
      if (_uint32(zip64Data, 0) != ZipDirectory.zip64EocdSignature ||
          zip64RecordSize < 44 ||
          zip64Offset > locatorOffset - 12 ||
          zip64RecordSize > locatorOffset - 12 - zip64Offset) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }
      diskNumber = _uint32(zip64Data, 16);
      centralDirectoryDisk = _uint32(zip64Data, 20);
      entriesOnDisk = _uint64(zip64Data, 24);
      entryCount = _uint64(zip64Data, 32);
      centralDirectorySize = _uint64(zip64Data, 40);
      centralDirectoryOffset = _uint64(zip64Data, 48);
      metadataStart = zip64Offset;
    }

    if (diskNumber != 0 ||
        centralDirectoryDisk != 0 ||
        entriesOnDisk != entryCount) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    }
    if (entryCount < 0 ||
        centralDirectorySize < 0 ||
        entryCount > maxArchiveEntries ||
        centralDirectorySize > maxCentralDirectoryBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.archiveTooLarge,
      );
    }
    if (centralDirectoryOffset < 0 ||
        centralDirectoryOffset > fileLength ||
        centralDirectorySize > fileLength - centralDirectoryOffset ||
        centralDirectoryOffset + centralDirectorySize > metadataStart) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    return InstagramZipMetadata(
      entryCount: entryCount,
      centralDirectoryOffset: centralDirectoryOffset,
      centralDirectorySize: centralDirectorySize,
    );
  } finally {
    file.closeSync();
  }
}

Uint8List _readAt(RandomAccessFile file, int offset, int length) {
  if (offset < 0 || offset > file.lengthSync() - length) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  }
  file.setPositionSync(offset);
  final bytes = file.readSync(length);
  if (bytes.length != length) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  }
  return bytes;
}

int _uint16(ByteData data, int offset) => data.getUint16(offset, Endian.little);

int _uint32(ByteData data, int offset) => data.getUint32(offset, Endian.little);

int _uint64(ByteData data, int offset) => data.getUint64(offset, Endian.little);
