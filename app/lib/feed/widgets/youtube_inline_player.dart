import 'dart:async';

import 'package:craftsky_app/feed/media/youtube_external.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
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

  // Keep the package overlay inside the card. Its mobile fullscreen transition
  // can reattach the same WebView ID on iOS, so fullscreen uses a fresh route.
  @override
  Widget build(BuildContext context) => Overlay.wrap(
    alwaysSizeToContent: true,
    child: YoutubePlayer(
      controller: _controller,
      aspectRatio: widget.external.isShort ? 9 / 16 : 16 / 9,
      autoFullScreen: false,
      enableFullScreenOnVerticalDrag: false,
      controlsBuilder: (context, _) => _FullscreenControl(
        onPressed: _openFullscreen,
      ),
    ),
  );

  Future<void> _openFullscreen() async {
    final startSeconds = await _controller.currentTime;
    if (!mounted) return;

    await _controller.pauseVideo();
    if (!mounted) return;

    final result = await Navigator.of(context, rootNavigator: true)
        .push<_FullscreenResult>(
          MaterialPageRoute(
            fullscreenDialog: true,
            builder: (_) => _YouTubeFullscreenPage(
              external: widget.external,
              startSeconds: startSeconds,
            ),
          ),
        );
    if (!mounted || result == null) return;
    if (result.failed) {
      widget.onPlaybackError();
      return;
    }

    await _controller.seekTo(
      seconds: result.seconds,
      allowSeekAhead: true,
    );
    await _controller.playVideo();
  }
}

class _FullscreenControl extends StatelessWidget {
  const _FullscreenControl({required this.onPressed, this.close = false});

  final VoidCallback onPressed;
  final bool close;

  @override
  Widget build(BuildContext context) => Stack(
    children: [
      PositionedDirectional(
        end: 4,
        bottom: 4,
        child: IconButton.filledTonal(
          key: Key(close ? 'youtube-exit-fullscreen' : 'youtube-fullscreen'),
          tooltip: close
              ? MaterialLocalizations.of(context).closeButtonTooltip
              : AppLocalizations.of(context).youtubeEnterFullscreen,
          onPressed: onPressed,
          icon: Icon(
            close
                ? CraftskyIconsBold.fullscreenExit
                : CraftskyIconsBold.fullscreen,
          ),
        ),
      ),
    ],
  );
}

class _YouTubeFullscreenPage extends StatefulWidget {
  const _YouTubeFullscreenPage({
    required this.external,
    required this.startSeconds,
  });

  final YouTubeExternal external;
  final double startSeconds;

  @override
  State<_YouTubeFullscreenPage> createState() => _YouTubeFullscreenPageState();
}

class _YouTubeFullscreenPageState extends State<_YouTubeFullscreenPage> {
  late final YoutubePlayerController _controller;
  late final StreamSubscription<YoutubePlayerValue> _subscription;
  late final Set<Factory<OneSequenceGestureRecognizer>> _gestureRecognizers;
  var _isClosing = false;

  @override
  void initState() {
    super.initState();
    unawaited(
      SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky),
    );
    _controller = YoutubePlayerController.fromVideoId(
      videoId: widget.external.videoId,
      startSeconds: widget.startSeconds,
      autoPlay: true,
      params: const YoutubePlayerParams(
        strictRelatedVideos: true,
      ),
    );
    _subscription = _controller.stream.listen((value) {
      if (value.hasError) unawaited(_finish(failed: true));
    });
    _gestureRecognizers = {
      Factory<OneSequenceGestureRecognizer>(
        () => _DownwardDismissGestureRecognizer(
          onDismiss: () => unawaited(_finish()),
        ),
      ),
    };
  }

  @override
  void dispose() {
    unawaited(_subscription.cancel());
    unawaited(_controller.close());
    unawaited(SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge));
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => CallbackShortcuts(
    bindings: {
      const SingleActivator(LogicalKeyboardKey.escape): () =>
          unawaited(_finish()),
    },
    child: Focus(
      autofocus: true,
      child: PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, _) {
          if (!didPop) unawaited(_finish());
        },
        child: Scaffold(
          backgroundColor: Colors.black,
          body: Overlay.wrap(
            child: Center(
              child: YoutubePlayer(
                controller: _controller,
                aspectRatio: widget.external.isShort ? 9 / 16 : 16 / 9,
                autoFullScreen: false,
                enableFullScreenOnVerticalDrag: false,
                backgroundColor: Colors.black,
                gestureRecognizers: _gestureRecognizers,
                controlsBuilder: (context, _) => _FullscreenControl(
                  close: true,
                  onPressed: () => unawaited(_finish()),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );

  Future<void> _finish({bool failed = false}) async {
    if (_isClosing || !mounted) return;
    _isClosing = true;
    final seconds = failed
        ? widget.startSeconds
        : await _controller.currentTime;
    if (!mounted) return;
    Navigator.pop(
      context,
      _FullscreenResult(seconds: seconds, failed: failed),
    );
  }
}

class _FullscreenResult {
  const _FullscreenResult({required this.seconds, required this.failed});

  final double seconds;
  final bool failed;
}

class _DownwardDismissGestureRecognizer extends OneSequenceGestureRecognizer {
  _DownwardDismissGestureRecognizer({required this.onDismiss});

  static const _dismissDistance = 96;

  final VoidCallback onDismiss;
  int? _pointer;
  Offset? _origin;

  @override
  void addAllowedPointer(PointerDownEvent event) {
    if (_pointer != null) return;
    super.addAllowedPointer(event);
    _pointer = event.pointer;
    _origin = event.position;
  }

  @override
  void handleEvent(PointerEvent event) {
    if (event.pointer != _pointer) return;
    final origin = _origin;
    if (event is PointerMoveEvent && origin != null) {
      final offset = event.position - origin;
      if (offset.dy > _dismissDistance && offset.dy > offset.dx.abs()) {
        invokeCallback<void>('onDismiss', onDismiss);
        resolve(GestureDisposition.rejected);
        stopTrackingPointer(event.pointer);
        return;
      }
      if (offset.dy < -kTouchSlop || offset.dx.abs() > kTouchSlop) {
        resolve(GestureDisposition.accepted);
        stopTrackingPointer(event.pointer);
        return;
      }
    }
    if (event is PointerUpEvent) {
      resolve(GestureDisposition.accepted);
      stopTrackingPointer(event.pointer);
    } else if (event is PointerCancelEvent) {
      resolve(GestureDisposition.rejected);
      stopTrackingPointer(event.pointer);
    }
  }

  @override
  void didStopTrackingLastPointer(int pointer) {
    _pointer = null;
    _origin = null;
  }

  @override
  String get debugDescription => 'downward dismiss';
}
