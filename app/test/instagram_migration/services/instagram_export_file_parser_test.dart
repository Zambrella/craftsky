import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('InstagramExportFileParser', () {
    late Directory temporaryDirectory;

    setUp(() {
      temporaryDirectory = Directory.systemTemp.createTempSync(
        'craftsky-instagram-export-',
      );
    });

    tearDown(() {
      temporaryDirectory.deleteSync(recursive: true);
    });

    test(
      'UT-018 parses only the canonical following entry from a stored ZIP',
      () async {
        final unrelatedBytes = Uint8List(
          InstagramImportParser.maxFileBytes + 1,
        )..setRange(0, 16, utf8.encode('private-sentinel'));
        final exportFile = _writeZip(
          temporaryDirectory,
          targetBytes: _followingBytes(),
          unrelatedBytes: unrelatedBytes,
          stored: true,
        );

        final result = await const InstagramExportFileParser().parsePath(
          exportFile.path,
        );

        expect(result.entries, [
          const InstagramImportEntry(username: 'synthetic.user'),
        ]);
        expect(result.ignoredEntryCount, 0);
        expect(result.duplicateEntryCount, 0);
        expect(result.toString(), isNot(contains('private-sentinel')));
      },
    );

    test('UT-018 parses standalone JSON and a deflated ZIP', () async {
      final followingBytes = _followingBytes(username: 'Synthetic.Deflated');
      final jsonFile = File(
        '${temporaryDirectory.path}/following.json',
      )..writeAsBytesSync(followingBytes);
      final zipFile = _writeZip(
        temporaryDirectory,
        targetBytes: followingBytes,
      );

      const parser = InstagramExportFileParser();
      final jsonResult = await parser.parsePath(jsonFile.path);
      final zipResult = await parser.parsePath(zipFile.path);

      expect(
        jsonResult.entries.map((entry) => entry.username),
        ['synthetic.deflated'],
      );
      expect(zipResult.entries, jsonResult.entries);
    });

    test('UT-018 requires exactly one canonical following entry', () async {
      final missingFile = _writeZip(
        temporaryDirectory,
        targetBytes: _followingBytes(),
        targetPath: 'connections/followers_and_following/following-1.json',
      );
      final duplicateBytes = _zipBytes(
        targetBytes: _followingBytes(),
        secondTargetPath: 'connections/followers_and_following/followinx.json',
      );
      _replaceAll(
        duplicateBytes,
        utf8.encode('connections/followers_and_following/followinx.json'),
        utf8.encode(InstagramExportFileParser.followingEntryPath),
      );
      final duplicateFile = File(
        '${temporaryDirectory.path}/duplicate.zip',
      )..writeAsBytesSync(duplicateBytes);

      await _expectCode(
        missingFile,
        InstagramImportParseErrorCode.unsupportedShape,
      );
      await _expectCode(
        duplicateFile,
        InstagramImportParseErrorCode.unsupportedShape,
      );
    });

    test('UT-018 rejects encrypted and unsupported target entries', () async {
      final encryptedArchive = Archive()
        ..addFile(
          ArchiveFile.bytes(
            InstagramExportFileParser.followingEntryPath,
            _followingBytes(),
          ),
        );
      final encryptedFile =
          File(
            '${temporaryDirectory.path}/encrypted.zip',
          )..writeAsBytesSync(
            ZipEncoder(
              password: 'synthetic-password',
            ).encodeBytes(encryptedArchive),
          );
      final unsupportedBytes = _zipBytes(
        targetBytes: _followingBytes(),
        stored: true,
      );
      final centralHeader = _findSignature(
        unsupportedBytes,
        ZipFileHeader.signature,
      );
      final localHeader = _findSignature(
        unsupportedBytes,
        ZipFile.zipSignature,
      );
      _writeUint16(unsupportedBytes, centralHeader + 10, 99);
      _writeUint16(unsupportedBytes, localHeader + 8, 99);
      final unsupportedFile = File(
        '${temporaryDirectory.path}/unsupported.zip',
      )..writeAsBytesSync(unsupportedBytes);

      await _expectCode(
        encryptedFile,
        InstagramImportParseErrorCode.unsupportedFormat,
      );
      await _expectCode(
        unsupportedFile,
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    });

    test('UT-018 rejects a mismatched local compression method', () async {
      final mismatchedBytes = _zipBytes(
        targetBytes: _followingBytes(),
        stored: true,
      );
      final localHeader = _findSignature(
        mismatchedBytes,
        ZipFile.zipSignature,
      );
      _writeUint16(mismatchedBytes, localHeader + 8, 99);
      final mismatchedFile = File(
        '${temporaryDirectory.path}/mismatched-local-header.zip',
      )..writeAsBytesSync(mismatchedBytes);

      await _expectCode(
        mismatchedFile,
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    });

    test('UT-018 rejects Unix symlink metadata before decoding', () async {
      final symlinkBytes = _zipBytes(
        targetBytes: _followingBytes(),
        unrelatedBytes: Uint8List.fromList(
          utf8.encode('synthetic-private-link-target'),
        ),
        stored: true,
      );
      final unrelatedCentralHeader = _findSignature(
        symlinkBytes,
        ZipFileHeader.signature,
        last: true,
      );
      _writeUint16(symlinkBytes, unrelatedCentralHeader + 4, 0x0314);
      _writeUint32(symlinkBytes, unrelatedCentralHeader + 38, 0xa000 << 16);
      final symlinkFile = File(
        '${temporaryDirectory.path}/unrelated-symlink.zip',
      )..writeAsBytesSync(symlinkBytes);

      await _expectCode(
        symlinkFile,
        InstagramImportParseErrorCode.unsupportedFormat,
      );
    });

    test('UT-018 rejects malformed and CRC-corrupt archives', () async {
      final validBytes = _zipBytes(
        targetBytes: _followingBytes(),
        stored: true,
      );
      final truncatedFile = File(
        '${temporaryDirectory.path}/truncated.zip',
      )..writeAsBytesSync(validBytes.sublist(0, validBytes.length - 1));
      final corruptBytes = Uint8List.fromList(validBytes);
      final canary = utf8.encode('relationships_following');
      final canaryOffset = _findBytes(corruptBytes, canary);
      corruptBytes[canaryOffset] ^= 1;
      final corruptFile = File(
        '${temporaryDirectory.path}/corrupt.zip',
      )..writeAsBytesSync(corruptBytes);

      await _expectCode(
        truncatedFile,
        InstagramImportParseErrorCode.invalidArchive,
      );
      await _expectCode(
        corruptFile,
        InstagramImportParseErrorCode.invalidArchive,
      );
    });

    test('UT-018 accepts valid ZIP64 metadata', () async {
      final zip64File =
          File(
            '${temporaryDirectory.path}/zip64.zip',
          )..writeAsBytesSync(
            _withZip64Metadata(
              _zipBytes(targetBytes: _followingBytes(), stored: true),
            ),
          );

      final result = await const InstagramExportFileParser().parsePath(
        zip64File.path,
      );

      expect(
        result.entries.map((entry) => entry.username),
        ['synthetic.user'],
      );
    });

    test('UT-018 enforces declared and actual target byte limits', () async {
      final declaredBytes = _zipBytes(
        targetBytes: _followingBytes(),
        stored: true,
      );
      final centralHeader = _findSignature(
        declaredBytes,
        ZipFileHeader.signature,
      );
      _writeUint32(
        declaredBytes,
        centralHeader + 24,
        InstagramImportParser.maxFileBytes + 1,
      );
      final declaredFile = File(
        '${temporaryDirectory.path}/declared-too-large.zip',
      )..writeAsBytesSync(declaredBytes);

      final oversizedTarget = Uint8List(
        InstagramImportParser.maxFileBytes + 1,
      );
      final actualBytes = _zipBytes(targetBytes: oversizedTarget);
      final actualCentralHeader = _findSignature(
        actualBytes,
        ZipFileHeader.signature,
      );
      _writeUint32(actualBytes, actualCentralHeader + 24, 1);
      final actualFile = File(
        '${temporaryDirectory.path}/actual-too-large.zip',
      )..writeAsBytesSync(actualBytes);

      await _expectCode(
        declaredFile,
        InstagramImportParseErrorCode.fileTooLarge,
      );
      await _expectCode(
        actualFile,
        InstagramImportParseErrorCode.fileTooLarge,
      );
    });

    test('UT-018 accepts an exactly 20 MiB target', () async {
      final minimalJson = utf8.encode('{"relationships_following":[]}');
      final maximumTarget = Uint8List(InstagramImportParser.maxFileBytes)
        ..fillRange(0, InstagramImportParser.maxFileBytes, 0x20)
        ..setRange(0, minimalJson.length, minimalJson);
      final file = _writeZip(
        temporaryDirectory,
        targetBytes: maximumTarget,
      );

      final result = await const InstagramExportFileParser().parsePath(
        file.path,
      );

      expect(result.entries, isEmpty);
    });

    test('UT-018 enforces archive metadata limits before decoding', () async {
      final base = _zipBytes(
        targetBytes: _followingBytes(),
        stored: true,
      );
      final tooManyEntriesFile =
          File(
            '${temporaryDirectory.path}/too-many-entries.zip',
          )..writeAsBytesSync(
            _withZip64Metadata(
              base,
              entryCount: InstagramExportFileParser.maxArchiveEntries + 1,
            ),
          );
      final oversizedDirectoryFile =
          File(
            '${temporaryDirectory.path}/oversized-directory.zip',
          )..writeAsBytesSync(
            _withZip64Metadata(
              base,
              centralDirectorySize:
                  InstagramExportFileParser.maxCentralDirectoryBytes + 1,
            ),
          );

      await _expectCode(
        tooManyEntriesFile,
        InstagramImportParseErrorCode.archiveTooLarge,
      );
      await _expectCode(
        oversizedDirectoryFile,
        InstagramImportParseErrorCode.archiveTooLarge,
      );
    });

    test(
      'UT-018 enforces actual count and directory byte boundaries',
      () async {
        final archiveBytes = _zipBytes(
          targetBytes: _followingBytes(),
          unrelatedBytes: Uint8List.fromList([1]),
          stored: true,
        );
        final eocdOffset = _findSignature(
          archiveBytes,
          ZipDirectory.eocdSignature,
          last: true,
        );
        final directorySize = ByteData.sublistView(
          archiveBytes,
        ).getUint32(eocdOffset + 12, Endian.little);
        final archiveFile = File(
          '${temporaryDirectory.path}/metadata-boundaries.zip',
        )..writeAsBytesSync(archiveBytes);

        Future<void> expectCode(
          InstagramExportFileParser parser,
          File file,
          InstagramImportParseErrorCode code,
        ) async {
          await expectLater(
            parser.parsePath(file.path),
            throwsA(
              isA<InstagramImportParseException>().having(
                (error) => error.code,
                'code',
                code,
              ),
            ),
          );
        }

        final entryCountAbove = InstagramExportFileParser.withLimits(
          maxArchiveEntries: 1,
          maxCentralDirectoryBytes: directorySize + 1,
        );
        final exactLimits = InstagramExportFileParser.withLimits(
          maxArchiveEntries: 2,
          maxCentralDirectoryBytes: directorySize,
        );
        final belowLimits = InstagramExportFileParser.withLimits(
          maxArchiveEntries: 3,
          maxCentralDirectoryBytes: directorySize + 1,
        );
        final directorySizeAbove = InstagramExportFileParser.withLimits(
          maxArchiveEntries: 3,
          maxCentralDirectoryBytes: directorySize - 1,
        );

        await expectCode(
          entryCountAbove,
          archiveFile,
          InstagramImportParseErrorCode.archiveTooLarge,
        );
        expect(
          (await exactLimits.parsePath(archiveFile.path)).entries,
          hasLength(1),
        );
        expect(
          (await belowLimits.parsePath(archiveFile.path)).entries,
          hasLength(1),
        );
        await expectCode(
          directorySizeAbove,
          archiveFile,
          InstagramImportParseErrorCode.archiveTooLarge,
        );

        final dishonestBytes = Uint8List.fromList(archiveBytes);
        _writeUint16(dishonestBytes, eocdOffset + 8, 1);
        _writeUint16(dishonestBytes, eocdOffset + 10, 1);
        final dishonestFile = File(
          '${temporaryDirectory.path}/dishonest-entry-count.zip',
        )..writeAsBytesSync(dishonestBytes);
        await expectCode(
          entryCountAbove,
          dishonestFile,
          InstagramImportParseErrorCode.archiveTooLarge,
        );
      },
    );

    test('UT-018 preserves the 10,000 normalized-entry limit', () async {
      final zipFile = _writeZip(
        temporaryDirectory,
        targetBytes: _followingBytes(count: 10001),
      );

      await _expectCode(
        zipFile,
        InstagramImportParseErrorCode.tooManyEntries,
      );
    });
  });
}

Uint8List _followingBytes({
  String username = 'Synthetic.User',
  int count = 1,
}) => Uint8List.fromList(
  utf8.encode(
    jsonEncode({
      'relationships_following': List.generate(count, (index) {
        final value = count == 1 ? username : 'synthetic_$index';
        return {
          'title': value,
          'string_list_data': [
            {
              'href': 'https://www.instagram.com/_u/$value',
              'timestamp': 1,
            },
          ],
        };
      }),
    }),
  ),
);

File _writeZip(
  Directory directory, {
  required Uint8List targetBytes,
  String targetPath = InstagramExportFileParser.followingEntryPath,
  Uint8List? unrelatedBytes,
  bool stored = false,
}) {
  final name = DateTime.now().microsecondsSinceEpoch;
  return File('${directory.path}/synthetic-$name.zip')..writeAsBytesSync(
    _zipBytes(
      targetBytes: targetBytes,
      targetPath: targetPath,
      unrelatedBytes: unrelatedBytes,
      stored: stored,
    ),
  );
}

Uint8List _zipBytes({
  required Uint8List targetBytes,
  String targetPath = InstagramExportFileParser.followingEntryPath,
  String? secondTargetPath,
  Uint8List? unrelatedBytes,
  bool stored = false,
}) {
  ArchiveFile file(String path, Uint8List bytes) => stored
      ? ArchiveFile.noCompress(path, bytes.length, bytes)
      : ArchiveFile.bytes(path, bytes);
  final archive = Archive()..addFile(file(targetPath, targetBytes));
  if (secondTargetPath != null) {
    archive.addFile(file(secondTargetPath, targetBytes));
  }
  if (unrelatedBytes != null) {
    archive.addFile(
      ArchiveFile.noCompress(
        'messages/inbox/private-sentinel.bin',
        unrelatedBytes.length,
        unrelatedBytes,
      ),
    );
  }
  return ZipEncoder().encodeBytes(archive);
}

Future<void> _expectCode(
  File file,
  InstagramImportParseErrorCode code,
) async {
  await expectLater(
    const InstagramExportFileParser().parsePath(file.path),
    throwsA(
      isA<InstagramImportParseException>().having(
        (error) => error.code,
        'code',
        code,
      ),
    ),
  );
}

int _findSignature(Uint8List bytes, int signature, {bool last = false}) {
  final data = ByteData.sublistView(bytes);
  if (last) {
    for (var index = bytes.length - 4; index >= 0; index--) {
      if (data.getUint32(index, Endian.little) == signature) return index;
    }
  } else {
    for (var index = 0; index <= bytes.length - 4; index++) {
      if (data.getUint32(index, Endian.little) == signature) return index;
    }
  }
  throw StateError('Synthetic ZIP signature not found');
}

int _findBytes(Uint8List bytes, List<int> needle) {
  for (var index = 0; index <= bytes.length - needle.length; index++) {
    var matches = true;
    for (var offset = 0; offset < needle.length; offset++) {
      if (bytes[index + offset] != needle[offset]) {
        matches = false;
        break;
      }
    }
    if (matches) return index;
  }
  throw StateError('Synthetic ZIP bytes not found');
}

void _replaceAll(Uint8List bytes, List<int> from, List<int> to) {
  if (from.length != to.length) {
    throw ArgumentError('Synthetic replacement must preserve ZIP offsets');
  }
  var replacements = 0;
  for (var index = 0; index <= bytes.length - from.length; index++) {
    var matches = true;
    for (var offset = 0; offset < from.length; offset++) {
      if (bytes[index + offset] != from[offset]) {
        matches = false;
        break;
      }
    }
    if (!matches) continue;
    bytes.setRange(index, index + to.length, to);
    replacements++;
    index += from.length - 1;
  }
  if (replacements != 2) {
    throw StateError('Expected local and central synthetic filenames');
  }
}

void _writeUint16(Uint8List bytes, int offset, int value) =>
    ByteData.sublistView(bytes).setUint16(offset, value, Endian.little);

void _writeUint32(Uint8List bytes, int offset, int value) =>
    ByteData.sublistView(bytes).setUint32(offset, value, Endian.little);

Uint8List _withZip64Metadata(
  Uint8List source, {
  int? entryCount,
  int? centralDirectorySize,
}) {
  final eocdOffset = _findSignature(
    source,
    ZipDirectory.eocdSignature,
    last: true,
  );
  final sourceData = ByteData.sublistView(source);
  final originalEntryCount = sourceData.getUint16(
    eocdOffset + 10,
    Endian.little,
  );
  final originalDirectorySize = sourceData.getUint32(
    eocdOffset + 12,
    Endian.little,
  );
  final originalDirectoryOffset = sourceData.getUint32(
    eocdOffset + 16,
    Endian.little,
  );
  final zip64Record = Uint8List(ZipDirectory.zip64EocdSize);
  ByteData.sublistView(zip64Record)
    ..setUint32(0, ZipDirectory.zip64EocdSignature, Endian.little)
    ..setUint64(4, 44, Endian.little)
    ..setUint16(12, 45, Endian.little)
    ..setUint16(14, 45, Endian.little)
    ..setUint32(16, 0, Endian.little)
    ..setUint32(20, 0, Endian.little)
    ..setUint64(24, entryCount ?? originalEntryCount, Endian.little)
    ..setUint64(32, entryCount ?? originalEntryCount, Endian.little)
    ..setUint64(
      40,
      centralDirectorySize ?? originalDirectorySize,
      Endian.little,
    )
    ..setUint64(48, originalDirectoryOffset, Endian.little);
  final locator = Uint8List(ZipDirectory.zip64EocdLocatorSize);
  ByteData.sublistView(locator)
    ..setUint32(0, ZipDirectory.zip64EocdLocatorSignature, Endian.little)
    ..setUint32(4, 0, Endian.little)
    ..setUint64(8, eocdOffset, Endian.little)
    ..setUint32(16, 1, Endian.little);
  final eocd = Uint8List.fromList(source.sublist(eocdOffset));
  ByteData.sublistView(eocd)
    ..setUint16(8, 0xffff, Endian.little)
    ..setUint16(10, 0xffff, Endian.little)
    ..setUint32(12, 0xffffffff, Endian.little)
    ..setUint32(16, 0xffffffff, Endian.little);
  return (BytesBuilder(copy: false)
        ..add(Uint8List.sublistView(source, 0, eocdOffset))
        ..add(zip64Record)
        ..add(locator)
        ..add(eocd))
      .toBytes();
}
