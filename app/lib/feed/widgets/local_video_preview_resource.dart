import 'package:craftsky_app/feed/models/video_thumbnail_frame.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource_handle.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource_native.dart'
    as platform;
import 'package:flutter/services.dart';

export 'local_video_preview_resource_handle.dart';

Future<LocalVideoPreviewResource> createLocalVideoPreviewResource(
  LocalVideoSelection selection,
) async {
  final uri = selection.localUri;
  if (uri != null && const {'file', 'content'}.contains(uri.scheme)) {
    return LocalVideoPreviewResource(uri, () async {});
  }
  return await platform.materializeLocalVideo(selection);
}

Future<Uint8List?> readLocalVideoThumbnailFrame(
  Uri uri,
  Duration duration,
  VideoThumbnailFrame frame,
) => readLocalVideoFrameAt(
  uri,
  Duration(milliseconds: frame.timestampMilliseconds(duration)),
);

Future<Uint8List?> readLocalVideoFrameAt(
  Uri uri,
  Duration position,
) async {
  try {
    // The existing midpointFrame method accepts any timestamp.
    return await const MethodChannel('social.craftsky/local_video')
        .invokeMethod<Uint8List>('midpointFrame', {
          'path': uri.toString(),
          'timeUs': position.inMicroseconds,
        });
  } on PlatformException {
    return null;
  } on MissingPluginException {
    return null;
  }
}
