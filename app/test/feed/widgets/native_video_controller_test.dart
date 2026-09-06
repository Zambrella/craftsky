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

  test('portrait video stays portrait in fullscreen', () {
    expect(
      nativeVideoFullscreenOrientations(isPortraitVideo: true),
      [DeviceOrientation.portraitUp, DeviceOrientation.portraitDown],
    );
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

final class _FakeAdapter implements NativeVideoAdapter {
  bool? playOnOpen;
  NativeVideoCaptionTrack? selectedCaption;
  int pauseCount = 0;
  bool disposed = false;
  bool mutedBeforeOpen = false;
  bool _muted = false;

  @override
  Future<void> open(Uri playlist, {required bool play}) async {
    playOnOpen = play;
    mutedBeforeOpen = _muted;
  }

  @override
  Future<void> pause() async => pauseCount++;

  @override
  Future<void> setMuted({required bool muted}) async => _muted = muted;

  @override
  Future<void> selectCaption(NativeVideoCaptionTrack? track) async =>
      selectedCaption = track;

  @override
  Future<void> dispose() async => disposed = true;
}
