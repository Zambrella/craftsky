import 'package:flutter/foundation.dart';
import 'package:flutter_image_compress/flutter_image_compress.dart';

bool get supportsAppleHeic =>
    !kIsWeb && defaultTargetPlatform == TargetPlatform.iOS;

/// Native calls remain on the root isolate; the plugin dispatches its work.
/// Never enable native debug logging: some implementations write source files.
Future<Uint8List> compressPlatformImage({
  required Uint8List bytes,
  required int width,
  required int height,
  required int quality,
  required bool png,
}) async {
  final result = await FlutterImageCompress.compressWithList(
    bytes,
    minWidth: width,
    minHeight: height,
    quality: quality,
    format: png ? CompressFormat.png : CompressFormat.jpeg,
    // Package defaults: strip EXIF and bake orientation.
  );
  if (result.isEmpty) throw const FormatException('Image compression failed');
  return result;
}
