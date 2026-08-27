import 'dart:async';

import 'package:craftsky_app/feed/media/youtube_external.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:youtube_player_iframe/youtube_player_iframe.dart';

typedef YouTubePlayerBuilder =
    Widget Function(
      BuildContext context,
      YouTubeExternal external,
    );

final youtubePlayerBuilderProvider = Provider<YouTubePlayerBuilder>(
  (_) =>
      (context, external) => YouTubeInlinePlayer(
        key: ValueKey(external.videoId),
        external: external,
      ),
);

class YouTubeInlinePlayer extends StatefulWidget {
  const YouTubeInlinePlayer({required this.external, super.key});

  final YouTubeExternal external;

  @override
  State<YouTubeInlinePlayer> createState() => _YouTubeInlinePlayerState();
}

class _YouTubeInlinePlayerState extends State<YouTubeInlinePlayer> {
  late final YoutubePlayerController _controller =
      YoutubePlayerController.fromVideoId(
        videoId: widget.external.videoId,
        startSeconds: widget.external.startSeconds.toDouble(),
        autoPlay: true,
        params: const YoutubePlayerParams(
          // The package defaults to the youtube-nocookie.com player host.
          showFullscreenButton: true,
          strictRelatedVideos: true,
        ),
      );

  @override
  void dispose() {
    unawaited(_controller.close());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => YoutubePlayer(
    controller: _controller,
    aspectRatio: widget.external.isShort ? 9 / 16 : 16 / 9,
  );
}
