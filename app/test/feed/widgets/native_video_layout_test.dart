import 'dart:ui';

import 'package:craftsky_app/feed/widgets/native_video_controller.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
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
    expect(style.fontFeatures, [FontFeature.tabularFigures()]);
  });
}
