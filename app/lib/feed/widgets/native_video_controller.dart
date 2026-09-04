import 'dart:async';
import 'dart:ui' show FontFeature;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:media_kit/media_kit.dart';
import 'package:media_kit_video/media_kit_video.dart';

Widget nativeVideoControls(VideoState state) => _NativeVideoControls(
  state: state,
  fullscreen: isFullscreen(state.context),
);

List<DeviceOrientation> nativeVideoFullscreenOrientations({
  required bool isPortraitVideo,
}) => isPortraitVideo
    ? const [DeviceOrientation.portraitUp, DeviceOrientation.portraitDown]
    : const [
        DeviceOrientation.landscapeLeft,
        DeviceOrientation.landscapeRight,
      ];

List<DeviceOrientation> nativeVideoRestoreOrientations(
  Orientation orientation,
) => orientation == Orientation.portrait
    ? const [DeviceOrientation.portraitUp, DeviceOrientation.portraitDown]
    : const [
        DeviceOrientation.landscapeLeft,
        DeviceOrientation.landscapeRight,
      ];

class NativeVideoTapSurface extends StatelessWidget {
  const NativeVideoTapSurface({
    required this.onTap,
    required this.child,
    super.key,
  });

  final VoidCallback onTap;
  final Widget child;

  @override
  Widget build(BuildContext context) => GestureDetector(
    behavior: HitTestBehavior.translucent,
    onTap: onTap,
    child: child,
  );
}

class NativeVideoControlsVisibility extends StatefulWidget {
  const NativeVideoControlsVisibility({
    required this.controls,
    this.hideAfter = const Duration(seconds: 3),
    super.key,
  });

  final Widget controls;
  final Duration hideAfter;

  @override
  State<NativeVideoControlsVisibility> createState() =>
      _NativeVideoControlsVisibilityState();
}

class _NativeVideoControlsVisibilityState
    extends State<NativeVideoControlsVisibility> {
  Timer? _hideTimer;
  bool _visible = true;

  @override
  void initState() {
    super.initState();
    _restartTimer();
  }

  @override
  void didUpdateWidget(NativeVideoControlsVisibility oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.hideAfter != oldWidget.hideAfter) _restartTimer();
  }

  @override
  void dispose() {
    _hideTimer?.cancel();
    super.dispose();
  }

  void _restartTimer() {
    _hideTimer?.cancel();
    _hideTimer = Timer(widget.hideAfter, () {
      if (mounted) setState(() => _visible = false);
    });
  }

  void _showControls() {
    if (!_visible) setState(() => _visible = true);
    _restartTimer();
  }

  void _toggleControls() {
    _hideTimer?.cancel();
    setState(() => _visible = !_visible);
    if (_visible) _restartTimer();
  }

  @override
  Widget build(BuildContext context) => MouseRegion(
    onHover: (_) => _showControls(),
    child: Stack(
      fit: StackFit.expand,
      children: [
        NativeVideoTapSurface(
          key: const Key('native-video-tap-surface'),
          onTap: _toggleControls,
          child: const SizedBox.expand(),
        ),
        AnimatedOpacity(
          key: const Key('native-video-controls-opacity'),
          opacity: _visible ? 1 : 0,
          duration: const Duration(milliseconds: 200),
          child: IgnorePointer(
            ignoring: !_visible,
            child: ExcludeSemantics(
              excluding: !_visible,
              child: Listener(
                onPointerDown: (_) => _showControls(),
                onPointerMove: (_) => _showControls(),
                child: widget.controls,
              ),
            ),
          ),
        ),
      ],
    ),
  );
}

TextStyle nativeVideoTimestampStyle(TextStyle? base) =>
    (base ?? const TextStyle()).copyWith(
      fontFamily: 'monospace',
      fontFeatures: const [FontFeature.tabularFigures()],
    );

class _NativeVideoControls extends StatelessWidget {
  const _NativeVideoControls({required this.state, required this.fullscreen});

  final VideoState state;
  final bool fullscreen;

  @override
  Widget build(BuildContext context) {
    final player = state.widget.controller.player;
    return StreamBuilder<bool>(
      stream: player.stream.playing,
      initialData: player.state.playing,
      builder: (context, playingSnapshot) => StreamBuilder<bool>(
        stream: player.stream.completed,
        initialData: player.state.completed,
        builder: (context, completedSnapshot) => StreamBuilder<Duration>(
          stream: player.stream.position,
          initialData: player.state.position,
          builder: (context, positionSnapshot) => StreamBuilder<Duration>(
            stream: player.stream.duration,
            initialData: player.state.duration,
            builder: (context, durationSnapshot) => StreamBuilder<double>(
              stream: player.stream.volume,
              initialData: player.state.volume,
              builder: (context, volumeSnapshot) {
                final playing = playingSnapshot.data ?? false;
                final completed = completedSnapshot.data ?? false;
                final position = positionSnapshot.data ?? Duration.zero;
                final duration = durationSnapshot.data ?? Duration.zero;
                final volume = volumeSnapshot.data ?? 0;
                return _NativeVideoInlineControlLayout(
                  playing: playing,
                  completed: completed,
                  position: position,
                  duration: duration,
                  muted: volume == 0,
                  fullscreen: fullscreen,
                  onPlayPause: () => unawaited(
                    completed
                        ? player.seek(Duration.zero).then((_) => player.play())
                        : player.playOrPause(),
                  ),
                  onSeek: (value) => unawaited(
                    player.seek(Duration(milliseconds: value.round())),
                  ),
                  onMute: () =>
                      unawaited(player.setVolume(volume == 0 ? 100 : 0)),
                  onFullscreen: () => unawaited(
                    fullscreen
                        ? state.exitFullscreen()
                        : state.enterFullscreen(),
                  ),
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}

class _NativeVideoInlineControlLayout extends StatelessWidget {
  const _NativeVideoInlineControlLayout({
    required this.playing,
    required this.completed,
    required this.position,
    required this.duration,
    required this.muted,
    required this.fullscreen,
    required this.onPlayPause,
    required this.onSeek,
    required this.onMute,
    required this.onFullscreen,
  });

  final bool playing;
  final bool completed;
  final Duration position;
  final Duration duration;
  final bool muted;
  final bool fullscreen;
  final VoidCallback onPlayPause;
  final ValueChanged<double> onSeek;
  final VoidCallback onMute;
  final VoidCallback onFullscreen;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final max = duration.inMilliseconds.toDouble().clamp(1.0, double.infinity);
    final value = position.inMilliseconds.toDouble().clamp(0.0, max);
    final playTooltip = completed
        ? l10n.nativeVideoReplay
        : playing
        ? l10n.nativeVideoPause
        : l10n.nativeVideoPlay;
    final controlBar = DecoratedBox(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Colors.transparent, Colors.black87],
        ),
      ),
      child: Row(
        children: [
          IconButton(
            tooltip: playTooltip,
            color: Colors.white,
            onPressed: onPlayPause,
            icon: Icon(
              completed
                  ? Icons.replay
                  : playing
                  ? Icons.pause
                  : Icons.play_arrow,
            ),
          ),
          Text(
            '${_formatVideoTime(position)} / ${_formatVideoTime(duration)}',
            style: nativeVideoTimestampStyle(
              Theme.of(
                context,
              ).textTheme.labelSmall?.copyWith(color: Colors.white),
            ),
          ),
          Expanded(
            child: Slider(value: value, max: max, onChanged: onSeek),
          ),
          IconButton(
            tooltip: muted ? l10n.nativeVideoUnmute : l10n.nativeVideoMute,
            color: Colors.white,
            onPressed: onMute,
            icon: Icon(muted ? Icons.volume_off : Icons.volume_up),
          ),
          IconButton(
            tooltip: fullscreen
                ? l10n.nativeVideoExitFullscreen
                : l10n.nativeVideoEnterFullscreen,
            color: Colors.white,
            onPressed: onFullscreen,
            icon: Icon(fullscreen ? Icons.fullscreen_exit : Icons.fullscreen),
          ),
        ],
      ),
    );
    return NativeVideoControlsVisibility(
      controls: Stack(
        fit: StackFit.expand,
        children: [
          Center(
            child: IconButton.filledTonal(
              tooltip: playTooltip,
              onPressed: onPlayPause,
              icon: Icon(
                completed
                    ? Icons.replay
                    : playing
                    ? Icons.pause
                    : Icons.play_arrow,
              ),
            ),
          ),
          PositionedDirectional(
            start: 0,
            end: 0,
            bottom: 0,
            child: fullscreen
                ? SafeArea(top: false, child: controlBar)
                : controlBar,
          ),
        ],
      ),
    );
  }
}

String _formatVideoTime(Duration value) {
  final minutes = value.inMinutes;
  final seconds = value.inSeconds.remainder(60).toString().padLeft(2, '0');
  return '$minutes:$seconds';
}

final class NativeVideoCaptionTrack {
  const NativeVideoCaptionTrack({
    required this.language,
    required this.label,
    required this.uri,
  });

  final String language;
  final String label;
  final Uri uri;
}

abstract interface class NativeVideoAdapter {
  Future<void> open(Uri playlist, {required bool play});
  Future<void> pause();
  Future<void> setMuted({required bool muted});
  Future<void> selectCaption(NativeVideoCaptionTrack? track);
  Future<void> dispose();
}

abstract interface class NativeVideoViewAdapter implements NativeVideoAdapter {
  Stream<Object> get playbackErrors;
  Stream<bool> get playbackReadyChanges;
  Stream<bool> get playingChanges;
  Widget buildView({required double aspectRatio});
}

final nativeVideoPlaybackCoordinator = NativeVideoPlaybackCoordinator();

final class NativeVideoPlaybackCoordinator {
  NativeVideoController? _active;

  Future<void> activate(NativeVideoController controller) async {
    final previous = _active;
    _active = controller;
    if (previous != null && !identical(previous, controller)) {
      await previous.pauseForCompetingPlayback();
    }
  }

  void release(NativeVideoController controller) {
    if (identical(_active, controller)) _active = null;
  }
}

final class NativeVideoController {
  NativeVideoController(
    this._adapter, {
    NativeVideoPlaybackCoordinator? playbackCoordinator,
  }) : _playbackCoordinator =
           playbackCoordinator ?? nativeVideoPlaybackCoordinator;

  final NativeVideoAdapter _adapter;
  final NativeVideoPlaybackCoordinator _playbackCoordinator;

  Future<void> open(Uri playlist, {bool play = false}) async {
    await _adapter.setMuted(muted: true);
    await _adapter.open(playlist, play: play);
  }

  Future<void> selectCaption(NativeVideoCaptionTrack? track) =>
      _adapter.selectCaption(track);

  Future<void> didStartPlayback() => _playbackCoordinator.activate(this);

  Future<void> pauseForCompetingPlayback() => _adapter.pause();

  Future<void> setVisible({required bool visible}) async {
    if (!visible) {
      _playbackCoordinator.release(this);
      await _adapter.pause();
    }
  }

  Future<void> didEnterBackground() async {
    _playbackCoordinator.release(this);
    await _adapter.pause();
  }

  Future<void> dispose() async {
    _playbackCoordinator.release(this);
    await _adapter.dispose();
  }
}

final class MediaKitNativeVideoAdapter implements NativeVideoViewAdapter {
  MediaKitNativeVideoAdapter({Player? player}) : _player = player ?? Player();

  final Player _player;
  Orientation? _orientationBeforeFullscreen;

  @override
  Stream<Object> get playbackErrors => _player.stream.error;

  @override
  Stream<bool> get playbackReadyChanges => _player.stream.width
      .where((width) => width != null && width > 0)
      .map((_) => true);

  @override
  Stream<bool> get playingChanges => _player.stream.playing;

  @override
  Widget buildView({required double aspectRatio}) => Builder(
    builder: (context) => Video(
      controller: VideoController(_player),
      aspectRatio: aspectRatio,
      controls: nativeVideoControls,
      onEnterFullscreen: () => _enterFullscreen(
        context,
        isPortraitVideo: aspectRatio < 1,
      ),
      onExitFullscreen: _exitFullscreen,
    ),
  );

  Future<void> _enterFullscreen(
    BuildContext context, {
    required bool isPortraitVideo,
  }) async {
    if (kIsWeb ||
        (defaultTargetPlatform != TargetPlatform.android &&
            defaultTargetPlatform != TargetPlatform.iOS)) {
      return defaultEnterNativeFullscreen();
    }
    try {
      _orientationBeforeFullscreen = MediaQuery.orientationOf(context);
      await Future.wait([
        SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky),
        SystemChrome.setPreferredOrientations(
          nativeVideoFullscreenOrientations(
            isPortraitVideo: isPortraitVideo,
          ),
        ),
      ]);
    } on Object catch (error, stackTrace) {
      debugPrint('$error\n$stackTrace');
    }
  }

  Future<void> _exitFullscreen() async {
    if (kIsWeb ||
        (defaultTargetPlatform != TargetPlatform.android &&
            defaultTargetPlatform != TargetPlatform.iOS)) {
      return defaultExitNativeFullscreen();
    }
    try {
      final orientation = _orientationBeforeFullscreen;
      await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
      if (orientation == null) return;
      await SystemChrome.setPreferredOrientations(
        nativeVideoRestoreOrientations(orientation),
      );
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await SystemChrome.setPreferredOrientations(const []);
    } on Object catch (error, stackTrace) {
      debugPrint('$error\n$stackTrace');
    } finally {
      _orientationBeforeFullscreen = null;
    }
  }

  @override
  Future<void> open(Uri playlist, {required bool play}) =>
      _player.open(Media(playlist.toString()), play: play);

  @override
  Future<void> pause() => _player.pause();

  @override
  Future<void> setMuted({required bool muted}) =>
      _player.setVolume(muted ? 0 : 100);

  @override
  Future<void> selectCaption(NativeVideoCaptionTrack? track) async {
    if (track == null) {
      await _player.setSubtitleTrack(SubtitleTrack.no());
      return;
    }
    await _player.setSubtitleTrack(
      SubtitleTrack.uri(
        track.uri.toString(),
        title: track.label,
        language: track.language,
      ),
    );
  }

  @override
  Future<void> dispose() => _player.dispose();
}
