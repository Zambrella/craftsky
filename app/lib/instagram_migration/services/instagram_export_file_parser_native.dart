import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:isolate';
import 'dart:math';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_limits.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_zip_envelope.dart';

Future<InstagramImportParseResult> parseInstagramExportPath(
  String path, {
  required int maxArchiveEntries,
  required int maxCentralDirectoryBytes,
}) => Isolate.run(
  () => _parseInstagramExportPathSync(
    path,
    maxArchiveEntries,
    maxCentralDirectoryBytes,
  ),
);

InstagramImportParseResult _parseInstagramExportPathSync(
  String path,
  int maxArchiveEntries,
  int maxCentralDirectoryBytes,
) {
  try {
    final file = File(path);
    final length = file.lengthSync();
    final input = file.openSync();
    late final Uint8List signature;
    try {
      signature = input.readSync(min(length, 4));
    } finally {
      input.closeSync();
    }
    if (_hasZipSignature(signature)) {
      return _parseZip(
        path,
        length,
        maxArchiveEntries,
        maxCentralDirectoryBytes,
      );
    }
    if (length > InstagramImportParser.maxFileBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    return const InstagramImportParser().parseJson(file.readAsBytesSync());
  } on InstagramImportParseException {
    rethrow;
  } on Object {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  }
}

bool _hasZipSignature(Uint8List bytes) =>
    bytes.length >= 4 &&
    bytes[0] == 0x50 &&
    bytes[1] == 0x4b &&
    ((bytes[2] == 0x03 && bytes[3] == 0x04) ||
        (bytes[2] == 0x05 && bytes[3] == 0x06) ||
        (bytes[2] == 0x07 && bytes[3] == 0x08));

InstagramImportParseResult _parseZip(
  String path,
  int fileLength,
  int maxArchiveEntries,
  int maxCentralDirectoryBytes,
) {
  final metadata = readInstagramZipMetadata(
    path,
    fileLength,
    maxArchiveEntries,
    maxCentralDirectoryBytes,
  );
  _preflightCentralDirectory(path, metadata, maxArchiveEntries);
  InputFileStream? input;
  Archive? archive;
  try {
    input = InputFileStream(path);
    final decoder = ZipDecoder();
    archive = decoder.decodeStream(input);
    final directory = decoder.directory;
    final headers = decoder.directory.fileHeaders;
    if (headers.length != metadata.entryCount ||
        directory.totalCentralDirectoryEntries != metadata.entryCount ||
        directory.totalCentralDirectoryEntriesOnThisDisk !=
            metadata.entryCount ||
        directory.centralDirectoryOffset != metadata.centralDirectoryOffset ||
        directory.centralDirectorySize != metadata.centralDirectorySize ||
        directory.numberOfThisDisk != 0 ||
        directory.diskWithTheStartOfTheCentralDirectory != 0) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    final targets = headers
        .where(
          (header) => header.filename == instagramFollowingEntryPath,
        )
        .toList(growable: false);
    if (targets.length != 1) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedShape,
      );
    }
    final header = targets.single;
    if (header.generalPurposeBitFlag & 1 != 0) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    }
    if (header.compressionMethod != ZipFile.zipCompressionStore &&
        header.compressionMethod != ZipFile.zipCompressionDeflate &&
        header.compressionMethod != ZipFile.zipCompressionBZip2) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    }
    if (header.uncompressedSize > InstagramImportParser.maxFileBytes) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    _validateTargetLocalHeader(path, header, metadata);
    final zipFile = header.file;
    final entry = archive.find(instagramFollowingEntryPath);
    if (zipFile == null ||
        entry == null ||
        !entry.isFile ||
        entry.isSymbolicLink ||
        zipFile.filename != header.filename ||
        zipFile.flags & 1 != 0) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }

    final output = _BoundedOutputStream(InstagramImportParser.maxFileBytes);
    entry.writeContent(output);
    final bytes = output.takeBytes();
    if (bytes.length != header.uncompressedSize ||
        output.crc32 != header.crc32) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    return const InstagramImportParser().parseJson(bytes);
  } on InstagramImportParseException {
    rethrow;
  } on Object {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.invalidArchive,
    );
  } finally {
    archive?.clearSync();
    input?.closeSync();
  }
}

const int _centralDirectoryDigitalSignature = 0x05054b50;

void _preflightCentralDirectory(
  String path,
  InstagramZipMetadata metadata,
  int maxArchiveEntries,
) {
  final file = File(path).openSync();
  try {
    final directoryEnd =
        metadata.centralDirectoryOffset + metadata.centralDirectorySize;
    file.setPositionSync(metadata.centralDirectoryOffset);
    var position = metadata.centralDirectoryOffset;
    var actualEntryCount = 0;
    var targetCount = 0;
    final expectedTarget = utf8.encode(instagramFollowingEntryPath);

    while (position < directoryEnd) {
      if (directoryEnd - position < 4) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }
      final signatureBytes = _readCurrent(file, 4);
      final signature = _uint32(ByteData.sublistView(signatureBytes), 0);
      if (signature == _centralDirectoryDigitalSignature) {
        if (directoryEnd - position < 6) {
          throw const InstagramImportParseException(
            InstagramImportParseErrorCode.invalidArchive,
          );
        }
        final signatureLengthBytes = _readCurrent(file, 2);
        final signatureLength = _uint16(
          ByteData.sublistView(signatureLengthBytes),
          0,
        );
        if (position + 6 + signatureLength != directoryEnd) {
          throw const InstagramImportParseException(
            InstagramImportParseErrorCode.invalidArchive,
          );
        }
        _readCurrent(file, signatureLength);
        position = directoryEnd;
        break;
      }
      if (signature != ZipFileHeader.signature ||
          directoryEnd - position < 46) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }

      actualEntryCount++;
      if (actualEntryCount > maxArchiveEntries) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.archiveTooLarge,
        );
      }
      final fixedHeaderTail = _readCurrent(file, 42);
      final fixedHeader = Uint8List(46)
        ..setRange(0, 4, signatureBytes)
        ..setRange(4, 46, fixedHeaderTail);
      final data = ByteData.sublistView(fixedHeader);
      final filenameLength = _uint16(data, 28);
      final extraLength = _uint16(data, 30);
      final commentLength = _uint16(data, 32);
      final entryLength = 46 + filenameLength + extraLength + commentLength;
      if (entryLength > directoryEnd - position) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }

      final filename = _readCurrent(file, filenameLength);
      final creatorSystem = _uint16(data, 4) >> 8;
      final unixFileType = (_uint32(data, 38) >> 16) & 0xf000;
      if (creatorSystem == 3 && unixFileType == 0xa000) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.unsupportedFormat,
        );
      }
      if (_bytesEqual(filename, expectedTarget)) {
        targetCount++;
      }
      file.setPositionSync(position + entryLength);
      position += entryLength;
    }

    if (position != directoryEnd || actualEntryCount != metadata.entryCount) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    if (targetCount != 1) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedShape,
      );
    }
  } finally {
    file.closeSync();
  }
}

void _validateTargetLocalHeader(
  String path,
  ZipFileHeader header,
  InstagramZipMetadata metadata,
) {
  if (header.diskNumberStart != 0) {
    throw const InstagramImportParseException(
      InstagramImportParseErrorCode.unsupportedFormat,
    );
  }
  final file = File(path).openSync();
  try {
    final local = _readAt(file, header.localHeaderOffset, 30);
    final data = ByteData.sublistView(local);
    if (_uint32(data, 0) != ZipFile.zipSignature) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    final flags = _uint16(data, 6);
    final compressionMethod = _uint16(data, 8);
    if (flags & 1 != 0 ||
        flags != header.generalPurposeBitFlag ||
        compressionMethod != header.compressionMethod ||
        (compressionMethod != ZipFile.zipCompressionStore &&
            compressionMethod != ZipFile.zipCompressionDeflate &&
            compressionMethod != ZipFile.zipCompressionBZip2)) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    }
    final filenameLength = _uint16(data, 26);
    final extraLength = _uint16(data, 28);
    final filename = _readAt(
      file,
      header.localHeaderOffset + 30,
      filenameLength,
    );
    final expectedFilename = utf8.encode(header.filename);
    if (!_bytesEqual(filename, expectedFilename)) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    final contentOffset =
        header.localHeaderOffset + 30 + filenameLength + extraLength;
    if (contentOffset > metadata.centralDirectoryOffset ||
        header.compressedSize >
            metadata.centralDirectoryOffset - contentOffset) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.invalidArchive,
      );
    }
    if (flags & 0x08 == 0) {
      final localCompressedSize = _uint32(data, 18);
      final localUncompressedSize = _uint32(data, 22);
      if (localUncompressedSize != 0xffffffff &&
          localUncompressedSize > InstagramImportParser.maxFileBytes) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.fileTooLarge,
        );
      }
      if (_uint32(data, 14) != header.crc32 ||
          (localCompressedSize != header.compressedSize &&
              localCompressedSize != 0xffffffff) ||
          (localUncompressedSize != header.uncompressedSize &&
              localUncompressedSize != 0xffffffff)) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.invalidArchive,
        );
      }
    }
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

Uint8List _readCurrent(RandomAccessFile file, int length) {
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

bool _bytesEqual(List<int> first, List<int> second) {
  if (first.length != second.length) return false;
  for (var index = 0; index < first.length; index++) {
    if (first[index] != second[index]) return false;
  }
  return true;
}

final class _BoundedOutputStream extends OutputStream {
  new(this.maximumLength) : super(byteOrder: ByteOrder.littleEndian);

  final int maximumLength;
  final BytesBuilder _bytes = BytesBuilder(copy: false);

  @override
  int length = 0;

  int crc32 = 0;

  @override
  void clear() {
    _bytes.clear();
    length = 0;
    crc32 = 0;
  }

  @override
  void flush() {}

  @override
  void writeByte(int value) => writeBytes([value]);

  @override
  void writeBytes(List<int> bytes, {int? length}) {
    final byteCount = length ?? bytes.length;
    if (byteCount < 0 ||
        byteCount > bytes.length ||
        this.length > maximumLength - byteCount) {
      throw const InstagramImportParseException(
        InstagramImportParseErrorCode.fileTooLarge,
      );
    }
    final chunk = bytes is Uint8List && byteCount == bytes.length
        ? bytes
        : Uint8List.fromList(bytes.take(byteCount).toList(growable: false));
    _bytes.add(chunk);
    this.length += byteCount;
    crc32 = getCrc32(chunk, crc32);
  }

  @override
  void writeStream(InputStream stream) {
    const chunkSize = 64 * 1024;
    while (!stream.isEOS) {
      final count = min(chunkSize, stream.length);
      if (count <= 0) break;
      writeBytes(stream.readBytes(count).toUint8List());
    }
  }

  @override
  Uint8List subset(int start, [int? end]) {
    final bytes = _bytes.toBytes();
    return Uint8List.sublistView(bytes, start, end);
  }

  Uint8List takeBytes() => _bytes.takeBytes();
}
