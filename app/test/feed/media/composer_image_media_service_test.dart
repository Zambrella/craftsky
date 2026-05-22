import 'dart:typed_data';

import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

void main() {
  group('ComposerImageMediaService', () {
    const service = ComposerImageMediaService();

    test('rejects WebP at selection time', () {
      final result = service.validateSelection(
        existing: const [],
        incoming: const [
          LocalImageSelection(name: 'project.webp', mimeType: 'image/webp'),
        ],
      );

      expect(result.accepted, isEmpty);
      expect(result.rejected, hasLength(1));
      expect(
        result.rejected.single.reason,
        ImageSelectionRejection.unsupportedType,
      );
    });

    test('infers supported MIME types from file names case-insensitively', () {
      expect(service.mimeTypeForFileName('PROJECT.JPG'), 'image/jpeg');
      expect(service.mimeTypeForFileName('project.jpeg'), 'image/jpeg');
      expect(service.mimeTypeForFileName('pattern.PNG'), 'image/png');
      expect(service.mimeTypeForFileName('notes.txt'), isEmpty);
    });

    test('accepts only remaining image slots in selection order', () {
      const limitedService = ComposerImageMediaService(
        config: MediaConfig(
          maxImages: 3,
          maxImageBytes: 1024,
          maxAltTextCharacters: 300,
        ),
      );

      final result = limitedService.validateSelection(
        existing: const [
          LocalImageSelection(name: 'existing.jpg', mimeType: 'image/jpeg'),
        ],
        incoming: const [
          LocalImageSelection(name: 'first.jpg', mimeType: ''),
          LocalImageSelection(name: 'second.png', mimeType: 'image/png'),
          LocalImageSelection(name: 'third.jpg', mimeType: 'image/jpeg'),
        ],
      );

      expect(
        result.accepted.map((selection) => selection.name),
        ['first.jpg', 'second.png'],
      );
      expect(result.rejected, hasLength(1));
      expect(result.rejected.single.image.name, 'third.jpg');
      expect(
        result.rejected.single.reason,
        ImageSelectionRejection.imageLimitExceeded,
      );
    });

    test('rejects original files over the configured byte limit', () {
      final result = service.validateOriginalImage(
        sizeBytes: mediaConfig.maxImageBytes + 1,
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        headerBytes: Uint8List(0),
      );

      expect(result.canPrepare, isFalse);
      expect(result.rejectedReason, OriginalImageRejection.tooLarge);
    });

    test('accepts supported originals before header bytes are available', () {
      final result = service.validateOriginalImage(
        sizeBytes: 1024,
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        headerBytes: Uint8List(0),
      );

      expect(result.canPrepare, isTrue);
      expect(result.rejectedReason, isNull);
    });

    test('rejects WebP header bytes before decode', () {
      final result = service.validateOriginalImage(
        sizeBytes: 12,
        fileName: 'project.webp',
        mimeType: 'image/webp',
        headerBytes: Uint8List.fromList([
          0x52,
          0x49,
          0x46,
          0x46,
          0x00,
          0x00,
          0x00,
          0x00,
          0x57,
          0x45,
          0x42,
          0x50,
        ]),
      );

      expect(result.canPrepare, isFalse);
      expect(result.rejectedReason, OriginalImageRejection.unsupportedType);
    });

    test('accepts a supported header when picker metadata is wrong', () {
      final jpegBytes = Uint8List.fromList(
        img.encodeJpg(img.Image(width: 1, height: 1)),
      );

      final result = service.validateOriginalImage(
        sizeBytes: jpegBytes.length,
        fileName: 'project.webp',
        mimeType: 'image/webp',
        headerBytes: jpegBytes.sublist(0, 16),
      );

      expect(result.canPrepare, isTrue);
      expect(result.rejectedReason, isNull);
    });

    test('rejects an unsupported header even with supported metadata', () {
      final result = service.validateOriginalImage(
        sizeBytes: 12,
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        headerBytes: Uint8List.fromList([
          0x52,
          0x49,
          0x46,
          0x46,
          0x00,
          0x00,
          0x00,
          0x00,
          0x57,
          0x45,
          0x42,
          0x50,
        ]),
      );

      expect(result.canPrepare, isFalse);
      expect(result.rejectedReason, OriginalImageRejection.unsupportedType);
    });

    test(
      'strips embedded JPEG EXIF while preserving baked orientation',
      () async {
        final source = img.Image(width: 2, height: 3);
        source.exif.imageIfd
          ..make = 'CameraCo'
          ..model = 'LeakyCam'
          ..software = 'MetadataWriter'
          ..orientation = 6;
        source.exif.gpsIfd.setGpsLocation(latitude: 47.61, longitude: -122.33);
        source.exif.exifIfd.userComment = 'private note';

        final originalBytes = Uint8List.fromList(img.encodeJpg(source));
        final originalExif = img.decodeJpgExif(originalBytes);
        expect(originalExif, isNotNull);
        expect(originalExif!.isEmpty, isFalse);

        final prepared = await service
            .prepareImage(
              bytes: originalBytes,
              fileName: 'project.jpg',
              mimeType: 'image/jpeg',
            )
            .future;

        final preparedExif = img.decodeJpgExif(prepared.bytes);
        expect(preparedExif == null || preparedExif.isEmpty, isTrue);
        expect(prepared.width, 3);
        expect(prepared.height, 2);
        expect(prepared.mimeType, 'image/jpeg');
      },
    );

    test(
      'strips PNG text metadata and reports retained transparency',
      () async {
        final source =
            img.Image(
                width: 2,
                height: 1,
                numChannels: 4,
                textData: {'comment': 'private note'},
              )
              ..setPixelRgba(0, 0, 255, 0, 0, 128)
              ..setPixelRgba(1, 0, 0, 255, 0, 255);

        final prepared = await service
            .prepareImage(
              bytes: Uint8List.fromList(img.encodePng(source)),
              fileName: 'project.png',
              mimeType: 'image/png',
              metadata: const {
                'alt': 'finished quilt block',
                'cameraMake': 'CameraCo',
                'comment': 'private note',
                'gpsLatitude': '47.61',
              },
            )
            .future;

        final decoded = img.decodePng(prepared.bytes);
        expect(decoded, isNotNull);
        expect(decoded!.textData, anyOf(isNull, isEmpty));
        expect(prepared.mimeType, 'image/png');
        expect(prepared.width, 2);
        expect(prepared.height, 1);
        expect(prepared.hasTransparency, isTrue);
        expect(prepared.metadata, {'alt': 'finished quilt block'});
      },
    );

    test('throws a format exception for corrupt image bytes', () async {
      final job = service.prepareImage(
        bytes: Uint8List.fromList([1, 2, 3, 4]),
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
      );

      await expectLater(job.future, throwsA(isA<FormatException>()));
    });

    test('validates prepared upload byte limits', () {
      expect(
        service
            .validatePreparedUploadBytes(
              originalBytes: 100,
              preparedBytes: mediaConfig.maxImageBytes,
            )
            .canUpload,
        isTrue,
      );

      final rejected = service.validatePreparedUploadBytes(
        originalBytes: 100,
        preparedBytes: mediaConfig.maxImageBytes + 1,
      );

      expect(rejected.canUpload, isFalse);
      expect(rejected.rejectedReason, PreparedUploadRejection.tooLarge);
    });

    test('returns an aspect ratio only for positive dimensions', () {
      expect(service.aspectRatioFor(width: 0, height: 10), isNull);
      expect(service.aspectRatioFor(width: 10, height: -1), isNull);

      final aspectRatio = service.aspectRatioFor(width: 4, height: 3);

      expect(aspectRatio, isNotNull);
      expect(aspectRatio!.width, 4);
      expect(aspectRatio.height, 3);
    });
  });
}
