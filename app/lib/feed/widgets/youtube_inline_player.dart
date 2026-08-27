import 'dart:async';

import 'package:craftsky_app/feed/media/youtube_external.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:youtube_player_iframe/youtube_player_iframe.dart';

typedef YouTubePlayerBuilder =
    Widget Function(
      BuildContext context,
      YouTubeExternal external,
      VoidCallback onPlaybackError,
    );

final youtubePlayerBuilderProvider = Provider<YouTubePlayerBuilder>(
  (_) =>
      (context, external, onPlaybackError) => YouTubeInlinePlayer(
        key: ValueKey(external.videoId),
        external: external,
        onPlaybackError: onPlaybackError,
      ),
);

class YouTubeInlinePlayer extends StatefulWidget {
  const YouTubeInlinePlayer({
    required this.external,
    required this.onPlaybackError,
    super.key,
  });

  final YouTubeExternal external;
  final VoidCallback onPlaybackError;

  @override
  State<YouTubeInlinePlayer> createState() => _YouTubeInlinePlayerState();
}

class _YouTubeInlinePlayerState extends State<YouTubeInlinePlayer> {
  late final YoutubePlayerController _controller;
  late final StreamSubscription<YoutubePlayerValue> _subscription;
  var _reportedError = false;

  @override
  void initState() {
    super.initState();
    _controller = YoutubePlayerController.fromVideoId(
      videoId: widget.external.videoId,
      startSeconds: widget.external.startSeconds.toDouble(),
      autoPlay: true,
      params: const YoutubePlayerParams(
        // The package defaults to the youtube-nocookie.com player host.
        showFullscreenButton: true,
        strictRelatedVideos: true,
      ),
    );
    _subscription = _controller.stream.listen((value) {
      if (value.hasError && !_reportedError) {
        _reportedError = true;
        widget.onPlaybackError();
      }
    });
  }

  @override
  void dispose() {
    unawaited(_subscription.cancel());
    unawaited(_controller.close());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => YoutubePlayer(
    controller: _controller,
    aspectRatio: widget.external.isShort ? 9 / 16 : 16 / 9,
  );
}
