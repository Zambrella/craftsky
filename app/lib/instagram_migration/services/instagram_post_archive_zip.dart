part of 'instagram_post_archive.dart';

const int _stored = ZipFile.zipCompressionStore;
const int _deflate = ZipFile.zipCompressionDeflate;
const int _bzip2 = ZipFile.zipCompressionBZip2;
const _directorySignature = 0x05054b50;

final class _ZipEntry {
  new(ZipFileHeader header)
    : name = header.filename,
      offset = header.localHeaderOffset,
      size = header.uncompressedSize,
      compressedSize = header.compressedSize,
      flags = header.generalPurposeBitFlag,
      method = header.compressionMethod,
      crc = header.crc32;

  final String name;
  final int offset;
  final int size;
  final int compressedSize;
  final int flags;
  final int method;
  final int crc;
  late final int end;

  bool withinLimits(int maximum) {
    if (method == _bzip2 &&
        (size > 20 * 1024 * 1024 || compressedSize > 40 * 1024 * 1024)) {
      return false;
    }
    return size > 0 &&
        size <= maximum &&
        compressedSize > 0 &&
        compressedSize <= maximum + 1024 * 1024 &&
        size / compressedSize <= 200;
  }
}

Never _invalidZip() => throw const InstagramPostArchiveException(
  InstagramPostArchiveError.invalidArchive,
);

List<_ZipEntry> _readDirectory(String path, int length) {
  final InstagramZipMetadata envelope;
  try {
    envelope = readInstagramZipMetadata(
      path,
      length,
      instagramMaxArchiveEntries,
      instagramMaxCentralDirectoryBytes,
    );
  } on InstagramImportParseException catch (error) {
    throw InstagramPostArchiveException(switch (error.code) {
      InstagramImportParseErrorCode.archiveTooLarge =>
        InstagramPostArchiveError.archiveTooLarge,
      InstagramImportParseErrorCode.unsupportedFormat =>
        InstagramPostArchiveError.unsupportedArchive,
      _ => InstagramPostArchiveError.invalidArchive,
    });
  }

  final input = InputFileStream(path);
  try {
    // Read only the bounded central directory, not the local entries. The
    // generic ZipDecoder opens every local header and may inspect symlinks.
    final directory = input.subset(
      position: envelope.centralDirectoryOffset,
      length: envelope.centralDirectorySize,
    );
    final entries = <_ZipEntry>[];
    for (var index = 0; index < envelope.entryCount; index++) {
      if (directory.length < 46 ||
          directory.readUint32() != ZipFileHeader.signature) {
        _invalidZip();
      }
      final fixed = directory.peekBytes(42).toUint8List();
      final data = ByteData.sublistView(fixed);
      final tail =
          data.getUint16(24, Endian.little) +
          data.getUint16(26, Endian.little) +
          data.getUint16(28, Endian.little);
      if (directory.length < 42 + tail) _invalidZip();
      final header = ZipFileHeader()..read(directory.readBytes(42 + tail));
      final mode = header.externalFileAttributes >> 16;
      final symlink = header.versionMadeBy >> 8 == 3 && mode & 0xf000 == 0xa000;
      if (header.generalPurposeBitFlag & 1 != 0 ||
          header.diskNumberStart != 0 ||
          symlink ||
          !{_stored, _deflate, _bzip2}.contains(header.compressionMethod)) {
        throw const InstagramPostArchiveException(
          InstagramPostArchiveError.unsupportedArchive,
        );
      }
      final entry = _ZipEntry(header);
      if (entry.offset < 0 ||
          entry.size < 0 ||
          entry.compressedSize < 0 ||
          entry.offset > envelope.centralDirectoryOffset - 30) {
        _invalidZip();
      }
      entries.add(entry);
    }
    if (!directory.isEOS) {
      // The optional digital-signature record is the only allowed tail.
      if (directory.length < 6 ||
          directory.readUint32() != _directorySignature ||
          directory.readUint16() != directory.length) {
        _invalidZip();
      }
    }

    final offsets = entries.toList()
      ..sort((a, b) => a.offset.compareTo(b.offset));
    for (var i = 0; i < offsets.length; i++) {
      final entry = offsets[i]
        ..end = i + 1 == offsets.length
            ? envelope.centralDirectoryOffset
            : offsets[i + 1].offset;
      if (entry.compressedSize > entry.end - entry.offset - 30) {
        _invalidZip();
      }
    }
    return entries;
  } finally {
    input.closeSync();
  }
}

Uint8List _readEntryBytes(String path, _ZipEntry entry) {
  final bytes = Uint8List(entry.size);
  _readEntry(path, entry, _EntryOutput(entry.size, bytes: bytes));
  return bytes;
}

void _readEntry(String path, _ZipEntry entry, _EntryOutput output) {
  final input = InputFileStream(path);
  try {
    input.setPosition(entry.offset);
    if (input.length < 30 || input.readUint32() != ZipFile.zipSignature) {
      _invalidZip();
    }
    input.skip(2);
    final flags = input.readUint16();
    final method = input.readUint16();
    input.skip(4);
    final crc = input.readUint32();
    final compressed = input.readUint32();
    final uncompressed = input.readUint32();
    final nameLength = input.readUint16();
    final extraLength = input.readUint16();
    if (flags != entry.flags ||
        method != entry.method ||
        nameLength + extraLength > entry.end - input.position ||
        entry.compressedSize >
            entry.end - input.position - nameLength - extraLength) {
      _invalidZip();
    }
    if (flags & 8 == 0 &&
        (crc != entry.crc ||
            (compressed != entry.compressedSize && compressed != 0xffffffff) ||
            (uncompressed != entry.size && uncompressed != 0xffffffff))) {
      _invalidZip();
    }
    final name = utf8.decode(input.readBytes(nameLength).toUint8List());
    if (name != entry.name) _invalidZip();
    input.skip(extraLength);
    final content = input.readBytes(entry.compressedSize);
    final valid = switch (method) {
      _stored => _copyStored(content, output),
      _deflate => const ZLibDecoder().decodeStream(
        content,
        output,
        raw: true,
        maxOutputSize: entry.size,
      ),
      _bzip2 => BZip2Decoder().decodeStream(
        content,
        output,
        verify: true,
        maxOutputSize: entry.size,
      ),
      _ => false,
    };
    output.flush();
    if (!valid || output.length != entry.size || output.crc != entry.crc) {
      _invalidZip();
    }
  } finally {
    input.closeSync();
  }
}

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
    if (length >= maximum) _invalidZip();
    _pending[_pendingCount++] = value;
    if (_pendingCount == _pending.length) flush();
  }

  @override
  void writeBytes(List<int> bytes, {int? length}) {
    final count = length ?? bytes.length;
    if (count < 0 || count > bytes.length || count > maximum - this.length) {
      _invalidZip();
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
      if (count <= 0) _invalidZip();
      final chunk = stream.readBytes(count).toUint8List();
      if (chunk.length != count) _invalidZip();
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
