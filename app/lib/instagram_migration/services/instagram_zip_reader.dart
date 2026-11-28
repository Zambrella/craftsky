import 'dart:convert';
import 'dart:io';
import 'dart:math';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_zip_envelope.dart';

const int _stored = ZipFile.zipCompressionStore;
const int _deflate = ZipFile.zipCompressionDeflate;
const int _bzip2 = ZipFile.zipCompressionBZip2;
const _digitalSignature = 0x05054b50;

final class InstagramZipEntry {
  new _(ZipFileHeader header)
    : name = header.filename,
      size = header.uncompressedSize,
      _compressedSize = header.compressedSize,
      _offset = header.localHeaderOffset,
      _flags = header.generalPurposeBitFlag,
      _method = header.compressionMethod,
      _disk = header.diskNumberStart,
      _crc = header.crc32,
      isDirectory =
          header.filename.endsWith('/') ||
          header.externalFileAttributes & 0x10 != 0 ||
          _unixFileType(header) == 0x4000;

  final String name;
  final int size;
  final bool isDirectory;
  final int _offset;
  final int _compressedSize;
  final int _flags;
  final int _method;
  final int _disk;
  final int _crc;
  late final int _end;

  bool get hasSupportedEncoding =>
      _disk == 0 &&
      _flags & 0x41 == 0 &&
      const {_stored, _deflate, _bzip2}.contains(_method);

  bool withinLimits(int maximum, {bool capCompressionRatio = true}) {
    if (_method == _bzip2 &&
        (size > 20 * 1024 * 1024 || _compressedSize > 40 * 1024 * 1024)) {
      return false;
    }
    return size <= maximum &&
        _compressedSize <= maximum + 1024 * 1024 &&
        (!capCompressionRatio ||
            (size > 0 && _compressedSize > 0 && size <= _compressedSize * 200));
  }

  Uint8List readBytes(
    String path,
    int maximum, {
    bool capCompressionRatio = true,
  }) {
    _checkLimit(maximum, capCompressionRatio: capCompressionRatio);
    final bytes = Uint8List(size);
    _read(path, _EntryOutput(size, bytes: bytes), maximum);
    return bytes;
  }

  void writeToFile(String path, File file, int maximum) {
    _checkLimit(maximum);
    final output = _EntryOutput(
      size,
      file: file.openSync(mode: FileMode.write),
    );
    try {
      _read(path, output, maximum);
    } finally {
      output.closeSync();
    }
  }

  void _checkLimit(int maximum, {bool capCompressionRatio = true}) {
    if (!hasSupportedEncoding) _unsupportedArchive();
    if (isDirectory) _invalidArchive();
    if (!withinLimits(maximum, capCompressionRatio: capCompressionRatio)) {
      _fileTooLarge();
    }
  }

  void _read(String path, _EntryOutput output, int maximum) {
    final input = InputFileStream(path);
    try {
      input.setPosition(_offset);
      if (input.length < 30 || input.readUint32() != ZipFile.zipSignature) {
        _invalidArchive();
      }
      input.skip(2);
      final localFlags = input.readUint16();
      final localMethod = input.readUint16();
      input.skip(4);
      final localCrc = input.readUint32();
      final compressed = input.readUint32();
      final uncompressed = input.readUint32();
      final nameLength = input.readUint16();
      final extraLength = input.readUint16();
      if (localFlags != _flags || localMethod != _method) {
        _unsupportedArchive();
      }
      if (nameLength + extraLength > _end - input.position ||
          _compressedSize > _end - input.position - nameLength - extraLength) {
        _invalidArchive();
      }
      // Bit 3 marks a data descriptor. When present, the central directory
      // provides the sizes and checksum because the local header omits them.
      if (_flags & 8 == 0) {
        if (uncompressed != 0xffffffff && uncompressed > maximum) {
          _fileTooLarge();
        }
        if (localCrc != _crc ||
            (compressed != _compressedSize && compressed != 0xffffffff) ||
            (uncompressed != size && uncompressed != 0xffffffff)) {
          _invalidArchive();
        }
      }
      final localName = utf8.decode(input.readBytes(nameLength).toUint8List());
      if (localName != name) _invalidArchive();
      input.skip(extraLength);
      final content = input.readBytes(_compressedSize);
      final valid = switch (_method) {
        _stored => _copyStored(content, output),
        _deflate => const ZLibDecoder().decodeStream(
          content,
          output,
          raw: true,
          maxOutputSize: size,
        ),
        _bzip2 => BZip2Decoder().decodeStream(
          content,
          output,
          verify: true,
          maxOutputSize: size,
        ),
        _ => false,
      };
      output.flush();
      if (!valid || output.length != size || output.crc != _crc) {
        _invalidArchive();
      }
    } on ArchiveException {
      _invalidArchive();
    } on FormatException {
      _invalidArchive();
    } finally {
      input.closeSync();
    }
  }
}

/// Inspect only the bounded central directory. Callers choose which members
/// to read without opening or decoding unrelated local entries.
List<InstagramZipEntry> readInstagramZipDirectory(
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
  final input = InputFileStream(path);
  try {
    final directory = input.subset(
      position: metadata.centralDirectoryOffset,
      length: metadata.centralDirectorySize,
    );
    final entries = <InstagramZipEntry>[];
    while (!directory.isEOS) {
      if (directory.length < 4) _invalidArchive();
      final signature = directory.readUint32();
      if (signature == _digitalSignature) {
        if (directory.length < 2 ||
            directory.readUint16() != directory.length) {
          _invalidArchive();
        }
        break;
      }
      if (signature != ZipFileHeader.signature || directory.length < 42) {
        _invalidArchive();
      }
      if (entries.length == maxArchiveEntries) {
        throw const InstagramImportParseException(
          InstagramImportParseErrorCode.archiveTooLarge,
        );
      }
      if (entries.length == metadata.entryCount) _invalidArchive();

      final data = ByteData.sublistView(directory.peekBytes(42).toUint8List());
      final tail =
          data.getUint16(24, Endian.little) +
          data.getUint16(26, Endian.little) +
          data.getUint16(28, Endian.little);
      if (directory.length < 42 + tail) _invalidArchive();
      final header = ZipFileHeader()..read(directory.readBytes(42 + tail));
      if (_unixFileType(header) == 0xa000) {
        _unsupportedArchive();
      }
      final entry = InstagramZipEntry._(header);
      if (entry._offset < 0 ||
          entry.size < 0 ||
          entry._compressedSize < 0 ||
          entry._offset > metadata.centralDirectoryOffset - 30) {
        _invalidArchive();
      }
      entries.add(entry);
    }
    if (entries.length != metadata.entryCount) _invalidArchive();

    final offsets = entries.toList()
      ..sort((a, b) => a._offset.compareTo(b._offset));
    for (var index = 0; index < offsets.length; index++) {
      final entry = offsets[index]
        .._end = index + 1 == offsets.length
            ? metadata.centralDirectoryOffset
            : offsets[index + 1]._offset;
      if (entry._compressedSize > entry._end - entry._offset - 30) {
        _invalidArchive();
      }
    }
    return entries;
  } finally {
    input.closeSync();
  }
}

int _unixFileType(ZipFileHeader header) =>
    (header.externalFileAttributes >> 16) & 0xf000;

Never _invalidArchive() => throw const InstagramImportParseException(
  InstagramImportParseErrorCode.invalidArchive,
);
Never _unsupportedArchive() => throw const InstagramImportParseException(
  InstagramImportParseErrorCode.unsupportedFormat,
);
Never _fileTooLarge() => throw const InstagramImportParseException(
  InstagramImportParseErrorCode.fileTooLarge,
);

bool _copyStored(InputStream input, _EntryOutput output) {
  output.writeStream(input);
  return true;
}

/// Bound actual decompression independently of the declared entry sizes and
/// compute integrity while streaming video to disk instead of Dart memory.
final class _EntryOutput extends OutputStream {
  new(this.maximum, {this.bytes, this.file})
    : super(byteOrder: ByteOrder.littleEndian);

  final int maximum;
  final Uint8List? bytes;
  final RandomAccessFile? file;
  final _pending = Uint8List(64 * 1024);
  var _pendingCount = 0;
  var _written = 0;
  int crc = 0;

  @override
  int get length => _written + _pendingCount;

  @override
  void writeByte(int value) {
    if (length >= maximum) _invalidArchive();
    _pending[_pendingCount++] = value;
    if (_pendingCount == _pending.length) flush();
  }

  @override
  void writeBytes(List<int> bytes, {int? length}) {
    final count = length ?? bytes.length;
    if (count < 0 || count > bytes.length || count > maximum - this.length) {
      _invalidArchive();
    }
    flush();
    _write(bytes, count);
  }

  void _write(List<int> chunk, int count) {
    if (count == 0) return;
    final memory = bytes;
    if (memory != null) {
      memory.setRange(_written, _written + count, chunk);
    } else {
      file!.writeFromSync(chunk, 0, count);
    }
    crc = getCrc32(
      count == chunk.length
          ? chunk
          : chunk is Uint8List
          ? Uint8List.sublistView(chunk, 0, count)
          : chunk.sublist(0, count),
      crc,
    );
    _written += count;
  }

  @override
  void writeStream(InputStream stream) {
    while (!stream.isEOS) {
      final count = min(_pending.length, stream.length);
      if (count <= 0) _invalidArchive();
      final chunk = stream.readBytes(count).toUint8List();
      if (chunk.length != count) _invalidArchive();
      writeBytes(chunk);
    }
  }

  @override
  void flush() {
    _write(_pending, _pendingCount);
    _pendingCount = 0;
  }

  @override
  void closeSync() => file?.closeSync();

  @override
  void clear() => throw UnsupportedError('Output cannot be reset.');

  @override
  Uint8List subset(int start, [int? end]) =>
      throw UnsupportedError('Output cannot be reread.');
}
