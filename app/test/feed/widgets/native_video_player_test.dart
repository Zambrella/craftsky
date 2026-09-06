import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/caption_uri_resource.dart';
import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:media_kit_video/media_kit_video.dart';
import 'package:visibility_detector/visibility_detector.dart';

void main() {
  testWidgets(
    'IT-015 player uses adapter lifecycle, captions, and soft errors',
    (tester) async {
      tester.view.physicalSize = const Size(800, 1800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final visibilityController = VisibilityDetectorController.instance;
      final previousInterval = visibilityController.updateInterval;
      visibilityController.updateInterval = Duration.zero;
      addTearDown(() => visibilityController.updateInterval = previousInterval);
      final adapter = _FakeViewAdapter();
      final resources = <_FakeCaptionResource>[];
      var adapterCreateCount = 0;

      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: SingleChildScrollView(
              child: NativeVideoPlayer(
                video: PostVideo(
                  cid: 'bafyvideo',
                  mime: 'video/mp4',
                  size: 10,
                  playlist: 'https://video.example/playlist.m3u8',
                  aspectRatio: const PostImageAspectRatio(
                    width: 720,
                    height: 1280,
                  ),
                  captions: const [
                    PostVideoCaption(
                      lang: 'en',
                      name: 'English',
                      uri:
                          '/v1/posts/did:plc:alice/post/video-captions/bafycap',
                    ),
                    PostVideoCaption(
                      lang: 'es',
                      name: 'Español',
                      uri:
                          '/v1/posts/did:plc:alice/post/video-captions/bafycap2',
                    ),
                  ],
                ),
                createAdapter: () {
                  adapterCreateCount++;
                  return adapter;
                },
                loadCaption: (_) async =>
                    'WEBVTT\n\n00:00.000 --> 00:01.000\nHello',
                createCaptionResource: (data) async {
                  final resource = _FakeCaptionResource(
                    Uri.parse('blob:https://app.example/${resources.length}'),
                  );
                  resources.add(resource);
                  return resource;
                },
              ),
            ),
          ),
        ),
      );
      await tester.pump();

      expect(find.byKey(const ValueKey('fake-video-view')), findsNothing);
      expect(
        find.byKey(const Key('native-video-thumbnail-play')),
        findsOneWidget,
      );
      expect(adapterCreateCount, 0);
      expect(adapter.playOnOpen, isNull);
      expect(resources, isEmpty);

      await tester.tap(find.byKey(const Key('native-video-thumbnail-play')));
      await tester.pump();

      expect(find.byKey(const ValueKey('fake-video-view')), findsOneWidget);
      expect(adapterCreateCount, 1);
      expect(adapter.playOnOpen, isTrue);
      expect(adapter.mutedBeforeOpen, isTrue);
      expect(adapter.viewAspectRatio, 9 / 16);
      expect(
        find.byKey(const Key('native-video-caption-menu')),
        findsOneWidget,
      );
      tester
          .widget<PopupMenuButton<int>>(
            find.byKey(const Key('native-video-caption-menu')),
          )
          .onSelected!(1);
      await tester.pump();
      expect(adapter.selectedCaption?.language, 'es');
      expect(adapter.selectedCaption?.uri.scheme, 'blob');

      adapter.errors.add(StateError('HLS failed'));
      await tester.pump();
      expect(find.byKey(const ValueKey('fake-video-view')), findsOneWidget);
      adapter.ready.add(true);
      await tester.pump(const Duration(seconds: 2));
      expect(
        find.text('Video is unavailable. Try again later.'),
        findsNothing,
      );

      adapter.errors.add(StateError('HLS failed after startup'));
      await tester.pump();
      expect(
        find.text('Video is unavailable. Try again later.'),
        findsOneWidget,
      );

      final pausesBeforeBackground = adapter.pauseCount;
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump();
      expect(adapter.pauseCount, pausesBeforeBackground + 1);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpWidget(const SizedBox());
      expect(adapter.disposed, isTrue);
      expect(resources.every((resource) => resource.disposed), isTrue);
      await adapter.close();
    },
  );

  test('AT-008 production view uses shared themed controls', () {
    expect(nativeVideoControls, isNot(same(AdaptiveVideoControls)));
  });
}

final class _FakeViewAdapter implements NativeVideoViewAdapter {
  final errors = StreamController<Object>.broadcast(sync: true);
  final ready = StreamController<bool>.broadcast(sync: true);
  bool? playOnOpen;
  NativeVideoCaptionTrack? selectedCaption;
  int pauseCount = 0;
  bool disposed = false;
  bool mutedBeforeOpen = false;
  double? viewAspectRatio;
  bool _muted = false;

  @override
  Stream<Object> get playbackErrors => errors.stream;

  @override
  Stream<bool> get playbackReadyChanges => ready.stream;

  @override
  Stream<bool> get playingChanges => const Stream.empty();

  @override
  Widget buildView({required double aspectRatio}) {
    viewAspectRatio = aspectRatio;
    return const SizedBox(key: ValueKey('fake-video-view'));
  }

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

  Future<void> close() async {
    await errors.close();
    await ready.close();
  }
}

final class _FakeCaptionResource implements CaptionUriResource {
  _FakeCaptionResource(this.uri);

  @override
  final Uri uri;

  bool disposed = false;

  @override
  Future<void> dispose() async => disposed = true;
}
