import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/media/heif_image_info.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../support/heic_fixture.dart';

void main() {
  test(
    'reads Apple-generated primary dimensions and rotation without decoding',
    () {
      final info = HeifImageInfo.read(rotatedHeicFixture());
      expect((info.width, info.height, info.rotated), (64, 32, true));
    },
  );

  test('rejects truncated metadata without range errors', () {
    final bytes = rotatedHeicFixture();
    var checked = 0;
    for (var end = 0; end < bytes.length; end++) {
      try {
        HeifImageInfo.read(Uint8List.sublistView(bytes, 0, end));
        break; // Pixel payload is not needed once complete metadata is present.
      } on FormatException {
        checked++;
      }
    }
    expect(checked, greaterThan(100));
  });

  test('accepts HEIC selection only on iOS', () {
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    const service = ComposerImageMediaService();
    for (final platform in [
      TargetPlatform.iOS,
      TargetPlatform.android,
    ]) {
      debugDefaultTargetPlatformOverride = platform;
      final result = service.validateOriginalImage(
        sizeBytes: rotatedHeicFixture().length,
        fileName: 'photo.heic',
        mimeType: 'image/heic',
        headerBytes: rotatedHeicFixture(),
      );
      expect(result.canPrepare, platform == TargetPlatform.iOS);
      final selection = service.validateSelection(
        incoming: const [
          LocalImageSelection(name: 'photo.heif', mimeType: 'image/heif'),
        ],
        existing: const [],
      );
      expect(
        selection.accepted.length,
        platform == TargetPlatform.iOS ? 1 : 0,
      );
    }
  });

  test(
    'Apple inspection reports oriented dimensions',
    () async {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      const service = ComposerImageMediaService();
      final inspected = await service
          .inspectImage(
            bytes: rotatedHeicFixture(),
            fileName: 'photo.heic',
            mimeType: 'image/heic',
          )
          .future;
      expect((inspected.width, inspected.height), (32, 64));
    },
  );
  test('rejects HEIC over 48 MP before native decoding', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    final bytes = rotatedHeicFixture();
    final data = ByteData.sublistView(bytes);
    for (var offset = 0; offset + 16 <= bytes.length; offset++) {
      if (String.fromCharCodes(bytes.sublist(offset, offset + 4)) == 'ispe') {
        data
          ..setUint32(offset + 8, 8000)
          ..setUint32(offset + 12, 6001);
      }
    }
    await expectLater(
      const ComposerImageMediaService()
          .prepareImage(
            bytes: bytes,
            fileName: 'large.heic',
            mimeType: 'image/heic',
          )
          .future,
      throwsA(isA<ImageSourceDimensionsTooLargeException>()),
    );
  });
}
