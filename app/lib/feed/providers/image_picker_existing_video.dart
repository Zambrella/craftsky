import 'dart:typed_data';

import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:flutter/services.dart';
import 'package:image_picker/image_picker.dart';

final class VideoProbeResult {
  const new({
    required this.duration,
    required this.width,
    required this.height,
    required this.posterBytes,
  });

  final Duration? duration;
  final int width;
  final int height;
  final Uint8List posterBytes;
}

final class ImagePickerExistingVideo implements ExistingVideoPicker {
  new({
    Future<XFile?> Function()? pick,
    Future<VideoProbeResult> Function(XFile file)? probe,
  }) : _pick =
           pick ?? (() => ImagePicker().pickVideo(source: ImageSource.gallery)),
       _probe = probe ?? _probeVideo;

  final Future<XFile?> Function() _pick;
  final Future<VideoProbeResult> Function(XFile file) _probe;

  @override
  Future<LocalVideoSelection?> pickExisting() async {
    final file = await _pick();
    if (file == null) return null;
    final byteLength = await file.length();
    if (byteLength > maxVideoSourceBytes) {
      throw const VideoSourceException(VideoSourceRejection.tooLarge);
    }
    if (byteLength <= 0) throw const FormatException('Empty video file');
    final header = BytesBuilder(copy: false);
    await file.openRead(0, 12).forEach(header.add);
    final headerBytes = header.takeBytes();
    final mimeType = detectVideoSourceMimeType(headerBytes);
    if (mimeType == null) {
      throw const VideoSourceException(VideoSourceRejection.unsupportedType);
    }
    final metadata = await _probe(file);
    final pathName = Uri.file(file.path).pathSegments.lastOrNull;
    final displayName = file.name.trim().isNotEmpty
        ? file.name
        : pathName?.trim().isNotEmpty ?? false
        ? pathName!
        : mimeType == 'video/quicktime'
        ? 'video.mov'
        : 'video.mp4';
    return LocalVideoSelection(
      displayName: displayName,
      localUri: Uri.tryParse(file.path)?.hasScheme == true
          ? Uri.parse(file.path)
          : Uri.file(file.path),
      mimeType: mimeType,
      byteLength: byteLength,
      duration: metadata.duration,
      width: metadata.width,
      height: metadata.height,
      headerBytes: headerBytes,
      openRead: file.openRead,
      posterBytes: metadata.posterBytes,
    );
  }
}

const _videoProbeChannel = MethodChannel('social.craftsky/local_video');

Future<VideoProbeResult> _probeVideo(XFile file) async {
  // Extract from the file on both mobile platforms before creating a player.
  final data = await _videoProbeChannel.invokeMapMethod<String, Object?>(
    'probe',
    {'path': file.path},
  );
  final width = data?['width'];
  final height = data?['height'];
  final durationMs = data?['durationMs'];
  final poster = data?['posterBytes'];
  if (width is! int ||
      width <= 0 ||
      height is! int ||
      height <= 0 ||
      poster is! Uint8List ||
      poster.isEmpty) {
    throw DiagnosticStateError('Video metadata unavailable');
  }
  return VideoProbeResult(
    duration: durationMs is int && durationMs > 0
        ? Duration(milliseconds: durationMs)
        : null,
    width: width,
    height: height,
    posterBytes: poster,
  );
}
