import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';

/// Keeps the PNG available until the native sheet completes or is dismissed.
Future<ShareResult> shareProfileCardImage(
  Uint8List bytes, {
  required Rect origin,
}) async {
  final directory = await (await getTemporaryDirectory()).createTemp(
    'craftsky-share-card-',
  );
  try {
    final file = File('${directory.path}/CraftSky-profile.png');
    await file.writeAsBytes(bytes, flush: true);
    return await SharePlus.instance.share(
      ShareParams(
        files: [XFile(file.path, mimeType: 'image/png')],
        fileNameOverrides: ['CraftSky-profile.png'],
        sharePositionOrigin: origin,
      ),
    );
  } finally {
    try {
      await directory.delete(recursive: true);
    } on FileSystemException {
      // OS-managed temporary storage remains eligible for automatic cleanup.
    }
  }
}
