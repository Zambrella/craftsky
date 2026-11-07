import 'dart:io';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource_handle.dart';

Future<LocalVideoPreviewResource> materializeLocalVideo(
  LocalVideoSelection selection,
) async {
  final directory = await Directory.systemTemp.createTemp(
    'craftsky-video-preview-',
  );
  try {
    final file = File(
      '${directory.path}/preview.${selection.mimeType == 'video/quicktime' ? 'mov' : 'mp4'}',
    );
    final sink = file.openWrite();
    var bytes = 0;
    try {
      await sink.addStream(
        selection.openRead().map((chunk) {
          bytes += chunk.length;
          if (bytes > maxVideoSourceBytes || bytes > selection.byteLength) {
            throw const VideoSourceException(VideoSourceRejection.tooLarge);
          }
          return chunk;
        }),
      );
      await sink.flush();
    } finally {
      await sink.close();
    }
    if (bytes != selection.byteLength || bytes == 0) {
      throw const FormatException('Incomplete local video');
    }
    return LocalVideoPreviewResource(file.uri, () async {
      if (directory.existsSync()) await directory.delete(recursive: true);
    });
  } catch (_) {
    await directory.delete(recursive: true);
    rethrow;
  }
}
