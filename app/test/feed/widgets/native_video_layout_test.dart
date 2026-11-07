import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:flutter/material.dart' as flutter;
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('fullscreen controls work inside the media-kit Material route', (
    tester,
  ) async {
    var toggles = 0;
    var exits = 0;
    var mutes = 0;
    double? seek;
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        // media-kit creates a Flutter Material, whereas our controls use
        // the standalone material_ui components and their own ink surface.
        home: flutter.Material(
          child: NativeVideoControlLayout(
            playing: false,
            completed: false,
            position: const Duration(seconds: 2),
            duration: const Duration(seconds: 5),
            muted: true,
            fullscreen: true,
            onPlayPause: () => toggles++,
            onSeek: (value) => seek = value,
            onMute: () => mutes++,
            onFullscreen: () => exits++,
          ),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    Finder button(String tooltip) => find.byWidgetPredicate(
      (widget) => widget is IconButton && widget.tooltip == tooltip,
    );
    await tester.tap(button('Play video').first);
    await tester.tap(button('Unmute video'));
    await tester.tap(button('Exit full screen'));
    await tester.drag(find.byType(Slider), const Offset(50, 0));
    await tester.pump();
    expect(toggles, 1);
    expect(mutes, 1);
    expect(exits, 1);
    expect(seek, greaterThan(2000));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  test('UT-016 uses valid declared ratio and stable fallback', () {
    expect(nativeVideoAspectRatio(width: 16, height: 9), 16 / 9);
    expect(nativeVideoAspectRatio(width: 720, height: 1280), 9 / 16);
    expect(nativeVideoAspectRatio(width: null, height: null), 16 / 9);
    expect(nativeVideoAspectRatio(width: 0, height: -1), 16 / 9);
    expect(nativeVideoAspectRatio(width: 1000, height: 1), 2.4);
  });

  testWidgets('vertical drag over inline video scrolls the feed', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ListView(
          children: [
            NativeVideoTapSurface(
              onTap: () {},
              child: const SizedBox(height: 400),
            ),
            const SizedBox(height: 1000),
          ],
        ),
      ),
    );

    await tester.drag(
      find.byType(NativeVideoTapSurface),
      const Offset(0, -300),
    );
    await tester.pumpAndSettle();

    expect(tester.widget<ListView>(find.byType(ListView)).controller, isNull);
    expect(
      tester.getTopLeft(find.byType(NativeVideoTapSurface)).dy,
      lessThan(0),
    );
  });

  testWidgets('inline controls hide after inactivity and return on tap', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: SizedBox.expand(
          child: NativeVideoControlsVisibility(
            hideAfter: Duration(seconds: 1),
            controls: SizedBox.expand(),
          ),
        ),
      ),
    );

    expect(
      tester
          .widget<AnimatedOpacity>(
            find.byKey(const Key('native-video-controls-opacity')),
          )
          .opacity,
      1,
    );

    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(milliseconds: 200));
    expect(
      tester
          .widget<AnimatedOpacity>(
            find.byKey(const Key('native-video-controls-opacity')),
          )
          .opacity,
      0,
    );

    await tester.tap(find.byKey(const Key('native-video-tap-surface')));
    await tester.pump();
    expect(
      tester
          .widget<AnimatedOpacity>(
            find.byKey(const Key('native-video-controls-opacity')),
          )
          .opacity,
      1,
    );
  });

  test('timestamps use monospaced tabular figures', () {
    final style = nativeVideoTimestampStyle(const TextStyle());

    expect(style.fontFamily, 'monospace');
    expect(style.fontFeatures, const [FontFeature.tabularFigures()]);
  });

  testWidgets(
    'completion reveals replay controls and replay restores auto-hide',
    (tester) async {
      var replays = 0;
      Widget player({required bool completed}) => MaterialApp(
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: NativeVideoControlLayout(
          playing: !completed,
          completed: completed,
          position: const Duration(seconds: 5),
          duration: const Duration(seconds: 5),
          muted: true,
          fullscreen: false,
          onPlayPause: () => replays++,
          onSeek: (_) {},
          onMute: () {},
          onFullscreen: () {},
        ),
      );
      double opacity() => tester
          .widget<AnimatedOpacity>(
            find.byKey(const Key('native-video-controls-opacity')),
          )
          .opacity;

      await tester.pumpWidget(player(completed: false));
      await tester.pump(const Duration(seconds: 4));
      expect(opacity(), 0);

      await tester.pumpWidget(player(completed: true));
      await tester.pumpAndSettle();
      expect(opacity(), 1);
      await tester.pump(const Duration(seconds: 5));
      expect(opacity(), 1);
      await tester.tap(find.byIcon(Icons.replay).first);
      expect(replays, 1);
      await tester.tapAt(
        tester.getTopLeft(find.byKey(const Key('native-video-tap-surface'))) +
            const Offset(20, 20),
      );
      await tester.pump();
      expect(opacity(), 1);

      await tester.pumpWidget(player(completed: false));
      await tester.pump(const Duration(seconds: 4));
      expect(opacity(), 0);
      await tester.pumpWidget(const SizedBox());
    },
  );
}
