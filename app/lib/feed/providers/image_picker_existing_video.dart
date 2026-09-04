import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:image_picker/image_picker.dart';
import 'package:media_kit/media_kit.dart';
import 'package:media_kit_video/media_kit_video.dart';

final class VideoProbeResult {
  const VideoProbeResult({
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
  ImagePickerExistingVideo({
    Future<XFile?> Function()? pick,
    Future<VideoProbeResult> Function(XFile file)? probe,
  }) : _pick =
           pick ?? (() => ImagePicker().pickVideo(source: ImageSource.gallery)),
       _probe = probe ?? _probeWithMediaKit;

  final Future<XFile?> Function() _pick;
  final Future<VideoProbeResult> Function(XFile file) _probe;

  @override
  Future<LocalVideoSelection?> pickExisting() async {
    final file = await _pick();
    if (file == null) return null;
    final byteLength = await file.length();
    final metadata = await _probe(file);
    final header = BytesBuilder(copy: false);
    await file.openRead(0, 12).forEach(header.add);
    final pathName = Uri.file(file.path).pathSegments.lastOrNull;
    final displayName = file.name.trim().isNotEmpty
        ? file.name
        : pathName?.trim().isNotEmpty ?? false
        ? pathName!
        : 'video.mp4';
    return LocalVideoSelection(
      displayName: displayName,
      mimeType: file.mimeType ?? 'video/mp4',
      byteLength: byteLength,
      duration: metadata.duration,
      width: metadata.width,
      height: metadata.height,
      headerBytes: header.takeBytes(),
      openRead: file.openRead,
      posterBytes: metadata.posterBytes,
    );
  }
}

Future<VideoProbeResult> _probeWithMediaKit(XFile file) async {
  final player = Player();
  final controller = VideoController(player);
  try {
    final parsed = Uri.tryParse(file.path);
    final uri = parsed != null && parsed.hasScheme
        ? parsed
        : Uri.file(file.path);
    await player.open(Media(uri.toString()), play: false);
    final duration = await resolveVideoDuration(
      current: player.state.duration,
      changes: player.stream.duration,
      timeout: const Duration(seconds: 10),
    );
    final width = (player.state.width ?? 0) > 0
        ? player.state.width!
        : await player.stream.width
              .firstWhere((value) => (value ?? 0) > 0)
              .then((value) => value!)
              .timeout(const Duration(seconds: 10));
    final height = (player.state.height ?? 0) > 0
        ? player.state.height!
        : await player.stream.height
              .firstWhere((value) => (value ?? 0) > 0)
              .then((value) => value!)
              .timeout(const Duration(seconds: 10));
    await controller.waitUntilFirstFrameRendered.timeout(
      const Duration(seconds: 10),
    );
    final poster = await player.screenshot();
    if (poster == null || poster.isEmpty) {
      throw StateError('Video poster unavailable');
    }
    return VideoProbeResult(
      duration: duration,
      width: width,
      height: height,
      posterBytes: poster,
    );
  } finally {
    await player.dispose();
  }
}

Future<Duration?> resolveVideoDuration({
  required Duration current,
  required Stream<Duration> changes,
  required Duration timeout,
}) async {
  if (current > Duration.zero) return current;
  try {
    return await changes
        .map<Duration?>((value) => value)
        .firstWhere(
          (value) => value != null && value > Duration.zero,
          orElse: () => null,
        )
        .timeout(timeout);
  } on TimeoutException {
    return null;
  }
}
