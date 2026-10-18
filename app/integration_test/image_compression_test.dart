import 'dart:isolate';
import 'dart:math';
import 'dart:typed_data';

import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/media/platform_image_compression.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:integration_test/integration_test.dart';

import '../test/support/heic_fixture.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('native JPEG processing bakes orientation and strips metadata', (
    tester,
  ) async {
    final image = img.Image(width: 40, height: 60);
    image.exif.imageIfd
      ..orientation = 6
      ..make = 'CraftSkyMetadataCanary';
    image.exif.gpsIfd.setGpsLocation(latitude: 47.61, longitude: -122.33);
    final prepared = await const ComposerImageMediaService()
        .prepareImage(
          bytes: Uint8List.fromList(img.encodeJpg(image)),
          fileName: 'oriented.jpg',
          mimeType: 'image/jpeg',
        )
        .future;
    expect((prepared.width, prepared.height), (60, 40));
    final exif = img.decodeJpgExif(prepared.bytes);
    expect(exif?.imageIfd.make ?? '', isEmpty);
    expect(exif?.gpsIfd.isEmpty ?? true, isTrue);
    expect(prepared.bytes.length, lessThanOrEqualTo(2000000));
  });

  testWidgets(
    'native iOS HEIC becomes an upright private-metadata-free JPEG',
    (tester) async {
      if (!supportsAppleHeic) return;
      final prepared = await const ComposerImageMediaService()
          .prepareImage(
            bytes: rotatedHeicFixture(),
            fileName: 'camera.heic',
            mimeType: 'image/heic',
          )
          .future;
      expect(prepared.mimeType, 'image/jpeg');
      expect((prepared.width, prepared.height), (32, 64));
      final decoded = img.decodeJpg(prepared.bytes)!;
      expect((decoded.width, decoded.height), (32, 64));
      final exif = img.decodeJpgExif(prepared.bytes);
      expect(exif?.imageIfd.make ?? '', isEmpty);
      expect(exif?.gpsIfd.isEmpty ?? true, isTrue);
      expect(prepared.bytes.length, lessThanOrEqualTo(1000000));
    },
  );

  testWidgets('native processing accepts a real 48 MP source', (tester) async {
    // Generate in a disposable isolate so the source bitmap is not retained
    // by the test while the native decoder processes the encoded photo.
    final original = await Isolate.run(
      () => Uint8List.fromList(
        img.encodeJpg(img.Image(width: 8000, height: 6000)),
      ),
    );
    const service = ComposerImageMediaService(
      config: MediaConfig(
        maxImages: 1,
        maxImageBytes: 1000000,
        targetImageBytes: 950000,
        maxAltTextCharacters: 0,
      ),
    );
    final prepared = await service
        .prepareImage(
          bytes: original,
          fileName: 'large.jpg',
          mimeType: 'image/jpeg',
        )
        .future;
    expect((prepared.width, prepared.height), (4000, 3000));
    expect(prepared.bytes.length, lessThanOrEqualTo(1000000));
  });

  testWidgets('native PNG preserves transparent pixels', (tester) async {
    final image = img.Image(width: 16, height: 16, numChannels: 4)
      ..setPixelRgba(0, 0, 255, 0, 0, 0)
      ..setPixelRgba(1, 0, 255, 0, 0, 255);
    final prepared = await const ComposerImageMediaService()
        .prepareImage(
          bytes: Uint8List.fromList(img.encodePng(image)),
          fileName: 'transparent.png',
          mimeType: 'image/png',
        )
        .future;
    expect(prepared.mimeType, 'image/png');
    expect(prepared.hasTransparency, isTrue);
    final decoded = img.decodePng(prepared.bytes)!;
    expect(decoded.getPixel(0, 0).a, 0);
    expect(decoded.getPixel(1, 0).a, 255);
  });

  testWidgets('native quality and resize loop meets a hard byte limit', (
    tester,
  ) async {
    final random = Random(42);
    final image = img.Image(width: 512, height: 512);
    for (final pixel in image) {
      pixel.setRgb(
        random.nextInt(256),
        random.nextInt(256),
        random.nextInt(256),
      );
    }
    const service = ComposerImageMediaService(
      config: MediaConfig(
        maxImages: 1,
        maxImageBytes: 18000,
        targetImageBytes: 16000,
        maxAltTextCharacters: 0,
      ),
    );
    final prepared = await service
        .prepareImage(
          bytes: Uint8List.fromList(img.encodeJpg(image, quality: 95)),
          fileName: 'noisy.jpg',
          mimeType: 'image/jpeg',
        )
        .future;
    expect(prepared.bytes.length, lessThanOrEqualTo(18000));
    expect(img.decodeJpg(prepared.bytes), isNotNull);
  });
}
