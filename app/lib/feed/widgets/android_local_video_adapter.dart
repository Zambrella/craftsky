import 'dart:async';
import 'dart:io';
import 'dart:math' as math;

import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:flutter/services.dart';
import 'package:material_ui/material_ui.dart';
import 'package:video_player/video_player.dart';

/// Uses Android's native decoder for a local composer preview.
final class AndroidLocalVideoAdapter
    implements NativeVideoViewAdapter, NativeVideoFullscreenAdapter {
  final _errors = StreamController<Object>.broadcast();
  final _ready = StreamController<bool>.broadcast();
  final _playing = StreamController<bool>.broadcast();
  VideoPlayerController? _player;
  bool _initialized = false;
  bool _disposed = false;
  bool _muted = false;
  bool _paused = false;
  bool _isFullscreen = false;
  bool _wasPlaying = false;
  bool _reportedError = false;

  @override
  Stream<Object> get playbackErrors => _errors.stream;

  @override
  Stream<bool> get playbackReadyChanges => _ready.stream;

  @override
  Stream<bool> get playingChanges => _playing.stream;

  @override
  bool get isFullscreen => _isFullscreen;

  @override
  Future<void> open(Uri playlist, {required bool play}) async {
    final player = playlist.scheme == 'content'
        ? VideoPlayerController.contentUri(playlist)
        : VideoPlayerController.file(File.fromUri(playlist));
    _player = player;
    await player.initialize();
    if (_disposed) return;
    _initialized = true;
    player.addListener(_onValueChanged);
    await player.setVolume(_muted ? 0 : 1);
    if (_disposed) return;
    _ready.add(true);
    if (play && !_paused) await player.play();
  }

  void _onValueChanged() {
    final value = _player!.value;
    if (value.isPlaying != _wasPlaying) {
      _wasPlaying = value.isPlaying;
      _playing.add(value.isPlaying);
    }
    if (value.hasError && !_reportedError) {
      _reportedError = true;
      _errors.add(DiagnosticStateError('Android local video playback failed'));
    }
  }

  @override
  Future<void> pause() async {
    _paused = true;
    if (_initialized) await _player?.pause();
  }

  @override
  Future<void> setMuted({required bool muted}) async {
    _muted = muted;
    if (_initialized) await _player?.setVolume(muted ? 0 : 1);
  }

  @override
  Future<void> selectCaption(NativeVideoCaptionTrack? track) async {
    // Composer previews have no external caption tracks.
  }

  @override
  Future<void> dispose() async {
    _disposed = true;
    final player = _player;
    player?.removeListener(_onValueChanged);
    try {
      await player?.dispose();
    } finally {
      unawaited(_errors.close());
      unawaited(_ready.close());
      unawaited(_playing.close());
    }
  }

  @override
  Widget buildView({required double aspectRatio}) {
    final player = _player;
    if (player == null || !_initialized) return const SizedBox.expand();
    return _buildSurface(player, fullscreen: false);
  }

  Widget _buildSurface(
    VideoPlayerController player, {
    required bool fullscreen,
  }) => ValueListenableBuilder<VideoPlayerValue>(
    valueListenable: player,
    builder: (context, value, video) => ColoredBox(
      color: Colors.black,
      child: Stack(
        fit: StackFit.expand,
        children: [
          Center(
            child: AspectRatio(aspectRatio: value.aspectRatio, child: video),
          ),
          NativeVideoControlLayout(
            playing: value.isPlaying,
            completed: value.isCompleted,
            position: value.position,
            duration: value.duration,
            muted: value.volume == 0,
            fullscreen: fullscreen,
            onPlayPause: () =>
                unawaited(value.isPlaying ? player.pause() : player.play()),
            onSeek: (position) => unawaited(
              player.seekTo(
                Duration(
                  milliseconds: math.min(
                    position.round(),
                    value.duration.inMilliseconds,
                  ),
                ),
              ),
            ),
            onMute: () => unawaited(setMuted(muted: value.volume != 0)),
            onFullscreen: () {
              if (fullscreen) {
                Navigator.of(context).pop();
              } else {
                unawaited(_showFullscreen(context, player, value.aspectRatio));
              }
            },
          ),
        ],
      ),
    ),
    child: VideoPlayer(player),
  );

  Future<void> _showFullscreen(
    BuildContext context,
    VideoPlayerController player,
    double aspectRatio,
  ) async {
    if (_isFullscreen) return;
    _isFullscreen = true;
    final navigator = Navigator.of(context, rootNavigator: true);
    try {
      await Future.wait([
        SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky),
        SystemChrome.setPreferredOrientations(
          nativeVideoFullscreenOrientations(isPortraitVideo: aspectRatio < 1),
        ),
      ]);
      if (_disposed || !navigator.mounted) return;
      await navigator.push<void>(
        MaterialPageRoute(
          builder: (_) => _buildSurface(player, fullscreen: true),
        ),
      );
    } finally {
      _isFullscreen = false;
      await Future.wait([
        SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge),
        SystemChrome.setPreferredOrientations(const []),
      ]);
    }
  }
}
