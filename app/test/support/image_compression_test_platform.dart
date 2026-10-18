import 'dart:isolate';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:flutter_image_compress/flutter_image_compress.dart';
import 'package:image/image.dart' as img;

/// A codec substitute for host widget tests, which cannot load native plugins.
/// Native plugin behavior is exercised separately by integration tests.
class TestImageCompressionPlatform extends FlutterImageCompressPlatform {
  @override
  dynamic noSuchMethod(Invocation invocation) =>
      throw UnimplementedError(invocation.memberName.toString());

  @override
  Future<Uint8List> compressWithList(
    Uint8List bytes, {
    int minWidth = 1920,
    int minHeight = 1080,
    int quality = 95,
    int rotate = 0,
    int inSampleSize = 1,
    bool autoCorrectionAngle = true,
    CompressFormat format = CompressFormat.jpeg,
    bool keepExif = false,
  }) => Isolate.run(() {
    if (keepExif || !autoCorrectionAngle || rotate != 0) {
      throw StateError('Unexpected metadata/orientation policy');
    }
    final decoded = img.decodeImage(bytes);
    if (decoded == null) throw const FormatException('Invalid image');
    var image = img.bakeOrientation(decoded);
    image.exif.clear();
    image
      ..textData = null
      ..iccProfile = null;
    // Reproduce the package's cover-style dimension parameters. The caller
    // must compute an aspect-preserving target to enforce maximum dimensions.
    final scale = math.min(
      1,
      math.max(minWidth / image.width, minHeight / image.height),
    );
    final width = math.max(1, (image.width * scale).floor());
    final height = math.max(1, (image.height * scale).floor());
    if (width != image.width || height != image.height) {
      image = img.copyResize(
        image,
        width: width,
        height: height,
        interpolation: img.Interpolation.average,
      );
    }
    return Uint8List.fromList(
      format == CompressFormat.png
          ? img.encodePng(image, singleFrame: true, level: 9)
          : img.encodeJpg(
              image,
              quality: quality,
              chroma: img.JpegChroma.yuv420,
            ),
    );
  });
}
