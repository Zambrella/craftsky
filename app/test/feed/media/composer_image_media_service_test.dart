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

      expect(result.accepted.map((selection) => selection.name), [
        'first.jpg',
        'second.png',
      ]);
      expect(result.rejected, hasLength(1));
      expect(result.rejected.single.image.name, 'third.jpg');
      expect(
        result.rejected.single.reason,
        ImageSelectionRejection.imageLimitExceeded,
      );
    });

    test('accepts compressible originals above the upload limit', () {
      final result = service.validateOriginalImage(
        sizeBytes: mediaConfig.maxImageBytes + 1,
        fileName: 'project.jpg',
        mimeType: 'image/jpeg',
        headerBytes: Uint8List(0),
      );

      expect(result.canPrepare, isTrue);
      expect(result.rejectedReason, isNull);
    });

    test('rejects original files over the source byte limit', () {
      final result = service.validateOriginalImage(
        sizeBytes: mediaConfig.maxSourceImageBytes + 1,
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

    test('inspects preview dimensions with baked orientation', () async {
      final source = img.Image(width: 2, height: 3)
        ..exif.imageIfd.orientation = 6;
      final originalBytes = Uint8List.fromList(img.encodeJpg(source));

      final inspected = await service
          .inspectImage(
            bytes: originalBytes,
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
          )
          .future;

      expect(inspected.width, 3);
      expect(inspected.height, 2);
    });

    test('inspects dimensions without decoding JPEG pixel data', () async {
      final headerOnly = _jpegHeader(width: 16, height: 8);

      final inspected = await service
          .inspectImage(
            bytes: headerOnly,
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
          )
          .future;

      expect((inspected.width, inspected.height), (16, 8));
      await expectLater(
        service
            .prepareImage(
              bytes: headerOnly,
              fileName: 'project.jpg',
              mimeType: 'image/jpeg',
            )
            .future,
        throwsA(isA<FormatException>()),
      );
    });

    test('inspects progressive JPEG dimensions', () async {
      final inspected = await service
          .inspectImage(
            bytes: _jpegHeader(width: 16, height: 8, frameMarker: 0xc2),
            fileName: 'progressive.jpg',
            mimeType: 'image/jpeg',
          )
          .future;

      expect((inspected.width, inspected.height), (16, 8));
    });

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

    test('normalizes truncated PNG inspection failures', () async {
      final job = service.inspectImage(
        bytes: Uint8List.fromList([
          0x89,
          0x50,
          0x4e,
          0x47,
          0x0d,
          0x0a,
          0x1a,
          0x0a,
        ]),
        fileName: 'project.png',
        mimeType: 'image/png',
      );

      await expectLater(job.future, throwsA(isA<FormatException>()));
    });

    test('rejects unsafe source geometry before pixel processing', () async {
      const guardedService = ComposerImageMediaService(
        config: MediaConfig(
          maxImages: 4,
          maxImageBytes: 2000000,
          maxAltTextCharacters: 300,
          maxSourceImageSide: 20,
          maxSourceImagePixels: 400,
        ),
      );

      await expectLater(
        guardedService
            .prepareImage(
              bytes: Uint8List.fromList(
                img.encodePng(img.Image(width: 21, height: 10)),
              ),
              fileName: 'project.png',
              mimeType: 'image/png',
            )
            .future,
        throwsA(isA<ImageSourceDimensionsTooLargeException>()),
      );
    });

    test('reports oversized source geometry with safe diagnostics', () async {
      final bytes = _jpegHeader(width: 4284, height: 5712);

      await expectLater(
        service
            .inspectImage(
              bytes: bytes,
              fileName: 'project.jpg',
              mimeType: 'image/jpeg',
            )
            .future,
        throwsA(
          isA<ImageSourceDimensionsTooLargeException>()
              .having((error) => error.width, 'width', 4284)
              .having((error) => error.height, 'height', 5712)
              .having(
                (error) => error.maxPixels,
                'maxPixels',
                mediaConfig.maxSourceImagePixels,
              ),
        ),
      );
    });

    test('accepts exactly 16 MP and rejects the next pixel row', () async {
      final accepted = await service
          .inspectImage(
            bytes: _jpegHeader(width: 4000, height: 4000),
            fileName: 'boundary.jpg',
            mimeType: 'image/jpeg',
          )
          .future;

      expect((accepted.width, accepted.height), (4000, 4000));
      await expectLater(
        service
            .inspectImage(
              bytes: _jpegHeader(width: 4000, height: 4001),
              fileName: 'over-boundary.jpg',
              mimeType: 'image/jpeg',
            )
            .future,
        throwsA(isA<ImageSourceDimensionsTooLargeException>()),
      );
    });

    test('proportionally resizes output to its geometry limit', () async {
      const resizingService = ComposerImageMediaService(
        config: MediaConfig(
          maxImages: 4,
          maxImageBytes: 2000000,
          maxAltTextCharacters: 300,
          maxSourceImageSide: 100,
          maxSourceImagePixels: 10000,
          maxImageWidth: 20,
          maxImageHeight: 20,
        ),
      );

      final prepared = await resizingService
          .prepareImage(
            bytes: Uint8List.fromList(
              img.encodePng(img.Image(width: 21, height: 10)),
            ),
            fileName: 'project.png',
            mimeType: 'image/png',
          )
          .future;

      expect((prepared.width, prepared.height), (20, 9));
    });

    test('compresses noisy JPEG bytes under the configured target', () async {
      const compressionService = ComposerImageMediaService(
        config: MediaConfig(
          maxImages: 4,
          maxImageBytes: 50000,
          maxAltTextCharacters: 300,
          targetImageBytes: 45000,
          maxImageWidth: 512,
          maxImageHeight: 512,
        ),
      );
      final source = img.Image(width: 512, height: 512);
      var value = 0x12345678;
      for (final pixel in source) {
        value = (1103515245 * value + 12345) & 0x7fffffff;
        pixel
          ..r = value & 0xff
          ..g = (value >> 8) & 0xff
          ..b = (value >> 16) & 0xff;
      }

      final prepared = await compressionService
          .prepareImage(
            bytes: Uint8List.fromList(img.encodeJpg(source)),
            fileName: 'noise.jpg',
            mimeType: 'image/jpeg',
          )
          .future;

      expect(prepared.bytes.length, lessThanOrEqualTo(50000));
      expect(prepared.mimeType, 'image/jpeg');
    });

    test(
      'scheduled preparation proportionally resizes over-budget pixels',
      () async {
        const scheduledService = ComposerImageMediaService(
          scheduledImageLimits: ScheduledImageLimits(
            maxWidth: 20,
            maxHeight: 20,
            maxPixels: 100,
            maxAspectRatio: 20,
          ),
        );
        final source = img.Image(width: 20, height: 10);

        final prepared = await scheduledService
            .prepareScheduledImage(
              bytes: Uint8List.fromList(img.encodePng(source)),
              fileName: 'project.png',
              mimeType: 'image/png',
            )
            .future;

        expect((prepared.width, prepared.height), (14, 7));
        final decoded = img.decodePng(prepared.bytes);
        expect(decoded, isNotNull);
        expect((decoded!.width, decoded.height), (14, 7));
      },
    );

    test('scheduled preparation rejects an extreme aspect ratio', () async {
      const scheduledService = ComposerImageMediaService(
        scheduledImageLimits: ScheduledImageLimits(
          maxWidth: 21,
          maxHeight: 21,
          maxPixels: 21,
          maxAspectRatio: 20,
        ),
      );

      await expectLater(
        scheduledService
            .prepareScheduledImage(
              bytes: Uint8List.fromList(
                img.encodePng(img.Image(width: 21, height: 1)),
              ),
              fileName: 'panorama.png',
              mimeType: 'image/png',
            )
            .future,
        throwsA(
          isA<FormatException>().having(
            (error) => error.message,
            'message',
            contains('aspect ratio'),
          ),
        ),
      );
    });

    test('validates prepared upload byte limits', () {
      expect(
        service
            .validatePreparedUploadBytes(
              originalBytes: 100,
              preparedBytes: mediaConfig.maxImageBytes,
              width: mediaConfig.maxImageWidth,
              height: mediaConfig.maxImageHeight,
            )
            .canUpload,
        isTrue,
      );

      final rejected = service.validatePreparedUploadBytes(
        originalBytes: 100,
        preparedBytes: mediaConfig.maxImageBytes + 1,
        width: mediaConfig.maxImageWidth,
        height: mediaConfig.maxImageHeight,
      );

      expect(rejected.canUpload, isFalse);
      expect(rejected.rejectedReason, PreparedUploadRejection.tooLarge);

      final oversizedDimensions = service.validatePreparedUploadBytes(
        originalBytes: 100,
        preparedBytes: 100,
        width: mediaConfig.maxImageWidth + 1,
        height: mediaConfig.maxImageHeight,
      );
      expect(oversizedDimensions.canUpload, isFalse);
      expect(
        oversizedDimensions.rejectedReason,
        PreparedUploadRejection.invalidDimensions,
      );
    });

    test('validates prepared upload aspect ratio boundaries', () {
      for (final dimensions in [(20, 1), (1, 20)]) {
        final accepted = service.validatePreparedUploadBytes(
          originalBytes: 100,
          preparedBytes: 100,
          width: dimensions.$1,
          height: dimensions.$2,
        );
        expect(accepted.canUpload, isTrue);
      }

      for (final dimensions in [(21, 1), (1, 21)]) {
        final rejected = service.validatePreparedUploadBytes(
          originalBytes: 100,
          preparedBytes: 100,
          width: dimensions.$1,
          height: dimensions.$2,
        );
        expect(rejected.canUpload, isFalse);
        expect(
          rejected.rejectedReason,
          PreparedUploadRejection.invalidDimensions,
        );
      }
    });

    test('enforces aspect ratio during canonical preparation', () async {
      for (final dimensions in [(20, 1), (1, 20)]) {
        final prepared = await service
            .prepareImage(
              bytes: Uint8List.fromList(
                img.encodePng(
                  img.Image(width: dimensions.$1, height: dimensions.$2),
                ),
              ),
              fileName: 'boundary.png',
              mimeType: 'image/png',
            )
            .future;
        expect((prepared.width, prepared.height), dimensions);
      }

      for (final dimensions in [(21, 1), (1, 21)]) {
        await expectLater(
          service
              .prepareImage(
                bytes: Uint8List.fromList(
                  img.encodePng(
                    img.Image(width: dimensions.$1, height: dimensions.$2),
                  ),
                ),
                fileName: 'panorama.png',
                mimeType: 'image/png',
              )
              .future,
          throwsA(
            isA<FormatException>().having(
              (error) => error.message,
              'message',
              contains('aspect ratio'),
            ),
          ),
        );
      }
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

Uint8List _jpegHeader({
  required int width,
  required int height,
  int frameMarker = 0xc0,
}) {
  return Uint8List.fromList([
    0xff,
    0xd8,
    0xff,
    frameMarker,
    0x00,
    0x11,
    0x08,
    height >> 8,
    height & 0xff,
    width >> 8,
    width & 0xff,
    0x03,
    0x01,
    0x11,
    0x00,
    0x02,
    0x11,
    0x00,
    0x03,
    0x11,
    0x00,
    0xff,
    0xd9,
  ]);
}
