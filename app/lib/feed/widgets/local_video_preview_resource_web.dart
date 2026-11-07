import 'dart:js_interop';
import 'dart:typed_data';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource_handle.dart';
import 'package:web/web.dart' as web;

Future<LocalVideoPreviewResource> materializeLocalVideo(
  LocalVideoSelection selection,
) async {
  final bytes = BytesBuilder(copy: false);
  await for (final chunk in selection.openRead()) {
    if (bytes.length + chunk.length > maxVideoSourceBytes ||
        bytes.length + chunk.length > selection.byteLength) {
      throw const VideoSourceException(VideoSourceRejection.tooLarge);
    }
    bytes.add(chunk);
  }
  if (bytes.length != selection.byteLength || bytes.length == 0) {
    throw const FormatException('Incomplete local video');
  }
  final blob = web.Blob(
    <web.BlobPart>[bytes.takeBytes().toJS].toJS,
    web.BlobPropertyBag(type: selection.mimeType),
  );
  final url = web.URL.createObjectURL(blob);
  return LocalVideoPreviewResource(Uri.parse(url), () async {
    web.URL.revokeObjectURL(url);
  });
}
