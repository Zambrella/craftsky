import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:archive/archive.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_post_archive.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as image;

const _metadata = 'your_instagram_activity/media/posts_1.json';
const _timestamp = 1700000000;
const _photo = 'media/posts/one.png';
const _video = 'media/reels/202311/one.mp4';
const _mp4 = [0, 0, 0, 20, 102, 116, 121, 112, 105, 115, 111, 109, 0, 0, 0, 0];

void main() {
  const reader = InstagramPostArchiveReader();
  late Directory directory;

  setUp(() {
    directory = Directory.systemTemp.createTempSync('instagram-post-test-');
  });

  tearDown(() => directory.deleteSync(recursive: true));

  File writeZip(
    Object metadata, {
    Map<String, List<int>> files = const {},
    String metadataPath = _metadata,
    String name = 'archive.zip',
    CompressionType compression = CompressionType.deflate,
    bool zip64 = false,
  }) {
    final archive = Archive()
      ..addFile(
        ArchiveFile.bytes(metadataPath, utf8.encode(jsonEncode(metadata)))
          ..compression = compression,
      );
    for (final entry in files.entries) {
      archive.addFile(
        ArchiveFile.bytes(entry.key, entry.value)..compression = compression,
      );
    }
    final encoded = ZipEncoder().encodeBytes(archive);
    return File('${directory.path}/$name')
      ..writeAsBytesSync(zip64 ? _withZip64Trailer(encoded) : encoded);
  }

  test(
    'preserves full captions and matches web source key across ZIP64',
    () async {
      final caption = 'A' * 2200;
      final first = await reader.inspect(
        writeZip([
          _post([_photo], caption: caption),
        ]).path,
      );
      final second = await reader.inspect(
        writeZip(
          [
            _post([_photo], caption: 'Caption changed'),
          ],
          name: 'second.zip',
          metadataPath: 'export/$_metadata',
          zip64: true,
        ).path,
      );

      final post = first.posts.single;
      expect(post.caption, caption);
      expect(post.createdAt.toIso8601String(), '2023-11-14T22:13:20.000Z');
      expect(post.sourceKey, '3b8b90a45597db1f75313b11f40baddb');
      expect(second.posts.single.sourceKey, post.sourceKey);
      expect(post.warnings, contains(InstagramArchiveWarning.captionTooLong));

      final composed = await reader.inspect(
        writeZip([
          _post(['media/posts/caf\u00e9.png']),
        ], name: 'composed.zip').path,
      );
      final combining = await reader.inspect(
        writeZip([
          _post(['media/posts/cafe\u0301.png']),
        ], name: 'combining.zip').path,
      );
      expect(
        composed.posts.single.sourceKey,
        'a6cbd2652d47cf618194f8281c544157',
      );
      expect(composed.posts.single.sourceKey, combining.posts.single.sourceKey);
    },
  );

  test(
    'handles detailed post media order and skips unpublished records',
    () async {
      final detailed = [
        {
          'fbid': '1',
          'timestamp': _timestamp + 10,
          'label_values': [
            {'label': 'Caption', 'value': 'CafÃ©'},
            {'label': 'Draft', 'value': 'False'},
            {'label': 'Published', 'value': 'True'},
            {
              'label': 'Metadata',
              'dict': [
                {
                  'dict': [
                    {
                      'label': 'Media',
                      'media': [_media(_video, time: _timestamp + 1)],
                    },
                  ],
                },
                {
                  'dict': [
                    {
                      'label': 'Media',
                      'media': [_media(_photo)],
                    },
                  ],
                },
              ],
            },
          ],
        },
        {
          'fbid': '2',
          'label_values': [
            {'label': 'Published', 'value': false},
            {
              'label': 'Media',
              'media': [_media('media/posts/unpublished.png')],
            },
          ],
        },
      ];
      final archive = await reader.inspect(
        writeZip(
          detailed,
          metadataPath: 'your_instagram_activity/media/posts.json',
        ).path,
      );

      expect(archive.skipped, 1);
      final post = archive.posts.single;
      expect(post.caption, 'Café');
      expect(post.createdAt.toIso8601String(), '2023-11-14T22:13:20.000Z');
      expect(post.media.map((item) => item.kind), [
        InstagramArchiveMediaKind.photo,
        InstagramArchiveMediaKind.video,
      ]);
      expect(post.warnings, contains(InstagramArchiveWarning.mixedMedia));
    },
  );

  for (final method in [CompressionType.none, CompressionType.bzip2]) {
    test('streams selected reel bytes from a ${method.name} ZIP', () async {
      final archive = await reader.inspect(
        writeZip(
          {
            'ig_reels_media': [
              _post([_video]),
            ],
          },
          metadataPath: 'your_instagram_activity/media/reels.json',
          files: {_video: _mp4},
          compression: method,
        ).path,
      );

      final post = archive.posts.single;
      expect(post.media.single.kind, InstagramArchiveMediaKind.video);
      final extracted = await reader.extract(archive, post, [0]);
      final video = File(extracted.video!);
      try {
        expect(extracted.photos, isEmpty);
        expect(video.readAsBytesSync(), _mp4);
        expect(video.path, endsWith('/video.mp4'));
      } finally {
        await extracted.dispose();
      }
      expect(video.existsSync(), isFalse);
    });
  }

  test('requires one media kind and extracts ordered selections', () async {
    const second = 'media/posts/two.png';
    final firstPhoto = image.encodePng(image.Image(width: 2, height: 1));
    final secondPhoto = image.encodePng(image.Image(width: 1, height: 2));
    final archive = await reader.inspect(
      writeZip(
        [
          _post([_photo, second, _video]),
        ],
        files: {_photo: firstPhoto, second: secondPhoto, _video: _mp4},
      ).path,
    );

    final post = archive.posts.single;
    await expectLater(
      () => reader.extract(archive, post, [0, 2]),
      _archiveError(InstagramPostArchiveError.invalidSelection),
    );
    final preview = await reader.preview(archive, post);
    expect(image.decodeJpg(preview!)!.width, 2);
    final extracted = await reader.extract(archive, post, [1, 0]);
    try {
      expect(extracted.video, isNull);
      expect(extracted.photos.map((path) => File(path).readAsBytesSync()), [
        firstPhoto,
        secondPhoto,
      ]);
    } finally {
      await extracted.dispose();
    }
  });

  test('rejects oversized image headers before thumbnail decoding', () async {
    final oversized = image.encodePng(image.Image(width: 1, height: 1));
    ByteData.sublistView(oversized)
      ..setUint32(16, 12000)
      ..setUint32(20, 12000);
    final archive = await reader.inspect(
      writeZip(
        [
          _post([_photo]),
        ],
        files: {_photo: oversized},
      ).path,
    );
    final post = archive.posts.single;
    expect(await reader.preview(archive, post), isNull);
    await expectLater(
      reader.extract(archive, post, [0]),
      _archiveError(InstagramPostArchiveError.mediaTooLarge),
    );
  });

  test('keeps original WebP bytes for native composer preparation', () async {
    const webpPath = 'media/posts/one.webp';
    final webp = image.encodeWebP(image.Image(width: 2, height: 3));
    final archive = await reader.inspect(
      writeZip(
        [
          _post([webpPath]),
        ],
        files: {webpPath: webp},
      ).path,
    );
    final extracted = await reader.extract(archive, archive.posts.single, [0]);
    try {
      expect(extracted.photos.single, endsWith('.webp'));
      expect(File(extracted.photos.single).readAsBytesSync(), webp);
    } finally {
      await extracted.dispose();
    }
  });

  test(
    'skips unsafe metadata and warns about ambiguous or oversized media',
    () async {
      final oversized = Uint8List(256 * 1024);
      const another = 'media/posts/two.png';
      final file = writeZip(
        [
          _post(['media/posts/../private.png']),
          _post([_photo, another]),
        ],
        files: {
          _photo: [1],
          'media/posts/ONE.png': [1],
          another: oversized,
        },
      );
      final archive = await reader.inspect(file.path);
      expect(archive.skipped, 1);
      final post = archive.posts.single;
      expect(post.media.every((item) => !item.available), isTrue);
      expect(
        post.warnings,
        containsAll([
          InstagramArchiveWarning.unavailableMedia,
          InstagramArchiveWarning.oversizedMedia,
        ]),
      );
      await expectLater(
        () => reader.extract(archive, post, [0]),
        _archiveError(InstagramPostArchiveError.mediaUnavailable),
      );
      final textOnly = await reader.extract(archive, post, []);
      expect(textOnly.photos, isEmpty);
      expect(textOnly.video, isNull);
    },
  );

  test(
    'extracts intact selections when a chosen member cannot be decoded',
    () async {
      const corruptPath = 'media/posts/corrupt.png';
      final png = image.encodePng(image.Image(width: 4, height: 4));
      final archive = await reader.inspect(
        writeZip(
          [
            _post([corruptPath, _photo], caption: 'Keep the original caption'),
          ],
          files: {
            corruptPath: [1, 2, 3],
            _photo: png,
          },
        ).path,
      );
      final post = archive.posts.single;
      expect(post.media.every((item) => item.available), isTrue);

      final media = await reader.extract(
        archive,
        post,
        [0, 1],
        temporaryDirectory: directory.path,
        allowPartial: true,
      );
      try {
        expect(media.unavailableCount, 1);
        expect(media.video, isNull);
        expect(File(media.photos.single).readAsBytesSync(), png);
        expect(post.caption, 'Keep the original caption');
      } finally {
        await media.dispose();
      }
      expect(File(media.photos.single).existsSync(), isFalse);
    },
  );

  test(
    'checks CRC on read and reports changed exports without a path',
    () async {
      final file = writeZip(
        [
          _post([_video]),
        ],
        files: {_video: _mp4},
        compression: CompressionType.none,
      );
      final archive = await reader.inspect(file.path);
      file.writeAsStringSync('changed');
      await expectLater(
        reader.extract(archive, archive.posts.single, [0]),
        _archiveError(InstagramPostArchiveError.archiveChanged),
      );
      await expectLater(
        reader.extract(archive, archive.posts.single, [0], allowPartial: true),
        _archiveError(InstagramPostArchiveError.archiveChanged),
      );

      final corrupt = writeZip(
        [
          _post([_video]),
        ],
        files: {_video: _mp4},
        compression: CompressionType.none,
      );
      final data = corrupt.readAsBytesSync();
      // Locate the selected entry by its complete stored payload and change one
      // byte while leaving the header and checksums untouched.
      var corrupted = false;
      for (var index = 0; index <= data.length - _mp4.length; index++) {
        if (Iterable<int>.generate(_mp4.length).every(
          (offset) => data[index + offset] == _mp4[offset],
        )) {
          data[index + _mp4.length - 1] = 1;
          corrupted = true;
          break;
        }
      }
      expect(corrupted, isTrue);
      corrupt.writeAsBytesSync(data);
      final damaged = await reader.inspect(corrupt.path);
      await expectLater(
        reader.extract(damaged, damaged.posts.single, [0]),
        _archiveError(InstagramPostArchiveError.invalidArchive),
      );
    },
  );
}

Map<String, Object> _post(List<String> paths, {String caption = 'Caption'}) => {
  'creation_timestamp': _timestamp,
  'title': caption,
  'media': [for (final path in paths) _media(path)],
};

Map<String, Object> _media(String path, {int time = _timestamp}) => {
  'uri': path,
  'creation_timestamp': time,
};

Matcher _archiveError(InstagramPostArchiveError code) => throwsA(
  isA<InstagramPostArchiveException>().having(
    (error) => error.code,
    'code',
    code,
  ),
);

Uint8List _withZip64Trailer(Uint8List zip) {
  final end = zip.length - 22;
  final eocd = ByteData.sublistView(zip, end);
  final entries = eocd.getUint16(10, Endian.little);
  final size = eocd.getUint32(12, Endian.little);
  final offset = eocd.getUint32(16, Endian.little);
  final trailer = ByteData(76)
    ..setUint32(0, 0x06064b50, Endian.little)
    ..setUint64(4, 44, Endian.little)
    ..setUint16(12, 45, Endian.little)
    ..setUint16(14, 45, Endian.little)
    ..setUint64(24, entries, Endian.little)
    ..setUint64(32, entries, Endian.little)
    ..setUint64(40, size, Endian.little)
    ..setUint64(48, offset, Endian.little)
    ..setUint32(56, 0x07064b50, Endian.little)
    ..setUint64(64, end, Endian.little)
    ..setUint32(72, 1, Endian.little);
  eocd
    ..setUint16(4, 0xffff, Endian.little)
    ..setUint16(6, 0xffff, Endian.little)
    ..setUint16(8, 0xffff, Endian.little)
    ..setUint16(10, 0xffff, Endian.little)
    ..setUint32(12, 0xffffffff, Endian.little)
    ..setUint32(16, 0xffffffff, Endian.little);
  return Uint8List.fromList([
    ...zip.sublist(0, end),
    ...trailer.buffer.asUint8List(),
    ...zip.sublist(end),
  ]);
}
