final class YouTubeExternal {
  const YouTubeExternal({
    required this.videoId,
    required this.isShort,
    this.startSeconds = 0,
  });

  final String videoId;
  final bool isShort;
  final int startSeconds;
}

/// Recognizes the public YouTube URL forms that CraftSky can safely embed.
///
/// The original URI remains the source of truth and should always be used for
/// external navigation. This parser only extracts bounded player parameters;
/// it never accepts an iframe URL supplied by post content.
YouTubeExternal? parseYouTubeExternal(Uri uri) {
  if (uri.scheme != 'https' && uri.scheme != 'http') {
    return null;
  }

  final host = uri.host.toLowerCase();
  final segments = uri.pathSegments
      .where((segment) => segment.isNotEmpty)
      .toList();
  String? videoId;
  var isShort = false;

  if (host == 'youtu.be') {
    videoId = segments.firstOrNull;
  } else if (_youtubeHosts.contains(host)) {
    if (segments case ['shorts', final id, ...]) {
      videoId = id;
      isShort = true;
    } else if (segments case ['live', final id, ...]) {
      videoId = id;
    } else if (segments.isEmpty || segments.first == 'watch') {
      videoId = uri.queryParameters['v'];
    }
  }

  if (videoId == null || !_videoId.hasMatch(videoId)) {
    return null;
  }

  return YouTubeExternal(
    videoId: videoId,
    isShort: isShort,
    startSeconds: _parseStartSeconds(
      uri.queryParameters['t'] ?? uri.queryParameters['start'],
    ),
  );
}

const _youtubeHosts = {
  'youtube.com',
  'www.youtube.com',
  'm.youtube.com',
  'music.youtube.com',
};

final _videoId = RegExp(r'^[A-Za-z0-9_-]{11}$');

int _parseStartSeconds(String? value) {
  if (value == null || value.isEmpty) {
    return 0;
  }
  final seconds = int.tryParse(value);
  if (seconds != null) {
    return seconds < 0 ? 0 : seconds;
  }

  final match = RegExp(
    r'^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$',
  ).firstMatch(value);
  if (match == null || match.group(0)!.isEmpty) {
    return 0;
  }
  final hours = int.tryParse(match.group(1) ?? '') ?? 0;
  final minutes = int.tryParse(match.group(2) ?? '') ?? 0;
  final remainingSeconds = int.tryParse(match.group(3) ?? '') ?? 0;
  return hours * 3600 + minutes * 60 + remainingSeconds;
}
