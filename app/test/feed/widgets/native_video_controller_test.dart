import 'dart:async';

import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-010 opens without autoplay and pauses at lifecycle boundaries',
    () async {
      final adapter = _FakeAdapter();
      final controller = NativeVideoController(adapter);

      await controller.open(
        Uri.parse('https://video.example/playlist.m3u8'),
      );
      await controller.selectCaption(
        NativeVideoCaptionTrack(
          language: 'en',
          label: 'English',
          uri: Uri.parse('file:///tmp/caption.vtt'),
        ),
      );
      await controller.setVisible(visible: false);
      await controller.didEnterBackground();
      await controller.dispose();

      expect(adapter.playOnOpen, isFalse);
      expect(adapter.mutedBeforeOpen, isTrue);
      expect(adapter.selectedCaption?.language, 'en');
      expect(adapter.pauseCount, 2);
      expect(adapter.disposed, isTrue);
    },
  );

  test('user-initiated open starts playback', () async {
    final adapter = _FakeAdapter();
    final controller = NativeVideoController(adapter);

    await controller.open(
      Uri.parse('https://video.example/playlist.m3u8'),
      play: true,
    );

    expect(adapter.playOnOpen, isTrue);
    expect(adapter.mutedBeforeOpen, isTrue);
  });

  test(
    'UT-010 activating one card pauses the previously active card',
    () async {
      final coordinator = NativeVideoPlaybackCoordinator();
      final firstAdapter = _FakeAdapter();
      final secondAdapter = _FakeAdapter();
      final first = NativeVideoController(
        firstAdapter,
        playbackCoordinator: coordinator,
      );
      final second = NativeVideoController(
        secondAdapter,
        playbackCoordinator: coordinator,
      );

      await first.didStartPlayback();
      await second.didStartPlayback();

      expect(firstAdapter.pauseCount, 1);
      expect(secondAdapter.pauseCount, 0);

      await second.setVisible(visible: false);
      expect(secondAdapter.pauseCount, 1);

      await first.didStartPlayback();
      await first.didEnterBackground();
      expect(firstAdapter.pauseCount, 2);
    },
  );

  test('late lifecycle callbacks do not access a disposed player', () async {
    final adapter = _FakeAdapter();
    final controller = NativeVideoController(adapter);
    await controller.dispose();

    await controller.setVisible(visible: false);
    await controller.didEnterBackground();
    await controller.pauseForCompetingPlayback();
    await controller.didStartPlayback();
    await controller.selectCaption(null);
    await controller.open(Uri.parse('https://video.example/playlist.m3u8'));
    await controller.dispose();

    expect(adapter.pauseCount, 0);
    expect(adapter.disposeCount, 1);
    expect(adapter.playOnOpen, isNull);
  });

  test('disposal during mute setup prevents a later open', () async {
    final adapter = _FakeAdapter()..muteGate = Completer<void>();
    final controller = NativeVideoController(adapter);
    final opening = controller.open(
      Uri.parse('https://video.example/playlist.m3u8'),
    );
    await controller.dispose();
    adapter.muteGate!.complete();
    await opening;

    expect(adapter.playOnOpen, isNull);
    expect(adapter.disposeCount, 1);
  });

  test('portrait video stays portrait in fullscreen', () {
    expect(
      nativeVideoFullscreenOrientations(isPortraitVideo: true),
      [DeviceOrientation.portraitUp, DeviceOrientation.portraitDown],
    );
  });

  test('fullscreen keeps playback through visibility changes; '
      'background and competing playback still pause', () async {
    final coordinator = NativeVideoPlaybackCoordinator();
    final adapter = _FakeAdapter()..isFullscreen = true;
    final controller = NativeVideoController(
      adapter,
      playbackCoordinator: coordinator,
    );
    await controller.didStartPlayback();
    await controller.setVisible(visible: false);
    expect(adapter.pauseCount, 0);

    await NativeVideoController(
      _FakeAdapter(),
      playbackCoordinator: coordinator,
    ).didStartPlayback();
    expect(adapter.pauseCount, 1);
    await controller.didEnterBackground();
    expect(adapter.pauseCount, 2);

    adapter.isFullscreen = false;
    await controller.setVisible(visible: false);
    expect(adapter.pauseCount, 3);
  });

  test('landscape video rotates to landscape in fullscreen', () {
    expect(
      nativeVideoFullscreenOrientations(isPortraitVideo: false),
      [DeviceOrientation.landscapeLeft, DeviceOrientation.landscapeRight],
    );
  });

  test('fullscreen exit restores the orientation used on entry', () {
    expect(nativeVideoRestoreOrientations(Orientation.portrait), [
      DeviceOrientation.portraitUp,
      DeviceOrientation.portraitDown,
    ]);
    expect(nativeVideoRestoreOrientations(Orientation.landscape), [
      DeviceOrientation.landscapeLeft,
      DeviceOrientation.landscapeRight,
    ]);
  });
}

final class _FakeAdapter implements NativeVideoFullscreenAdapter {
  @override
  bool isFullscreen = false;
  bool? playOnOpen;
  NativeVideoCaptionTrack? selectedCaption;
  int pauseCount = 0;
  bool disposed = false;
  int disposeCount = 0;
  Completer<void>? muteGate;
  bool mutedBeforeOpen = false;
  bool _muted = false;

  @override
  Future<void> open(Uri playlist, {required bool play}) async {
    if (disposed) throw AssertionError('[Player] has been disposed');
    playOnOpen = play;
    mutedBeforeOpen = _muted;
  }

  @override
  Future<void> pause() async {
    if (disposed) throw AssertionError('[Player] has been disposed');
    pauseCount++;
  }

  @override
  Future<void> setMuted({required bool muted}) async {
    if (disposed) throw AssertionError('[Player] has been disposed');
    _muted = muted;
    await muteGate?.future;
  }

  @override
  Future<void> selectCaption(NativeVideoCaptionTrack? track) async =>
      selectedCaption = track;

  @override
  Future<void> dispose() async {
    disposeCount++;
    disposed = true;
  }
}
