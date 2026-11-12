import 'dart:ui' as ui;

import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/profile/widgets/profile_shader_background.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:visibility_detector/visibility_detector.dart';

const _captureKey = Key('shader-capture');

Widget _subject(
  String background, {
  bool reduceMotion = false,
  bool enabled = true,
}) => MediaQuery(
  data: MediaQueryData(disableAnimations: reduceMotion),
  child: Directionality(
    textDirection: TextDirection.ltr,
    child: TickerMode(
      enabled: enabled,
      child: Center(
        child: RepaintBoundary(
          key: _captureKey,
          child: SizedBox(
            width: 300,
            height: 120,
            child: ProfileHeaderBackground(
              customisation: ProfileCustomisation(
                colour: 'terracotta',
                background: background,
              ),
            ),
          ),
        ),
      ),
    ),
  ),
);

Future<List<int>> _pixels(WidgetTester tester) async =>
    (await tester.runAsync(() async {
      final boundary = tester.renderObject<RenderRepaintBoundary>(
        find.byKey(_captureKey),
      );
      final image = await boundary.toImage(pixelRatio: 0.5);
      final bytes = await image.toByteData();
      image.dispose();
      return bytes!.buffer.asUint8List().toList();
    }))!;

void main() {
  setUp(() {
    VisibilityDetectorController.instance.updateInterval = Duration.zero;
  });
  tearDown(() {
    VisibilityDetectorController.instance.updateInterval = const Duration(
      milliseconds: 500,
    );
  });

  testWidgets('all compiled gradient styles paint and change over time', (
    tester,
  ) async {
    // Fail on real shader compilation/asset errors, rather than accepting fallback.
    await tester.runAsync(() async {
      final program = await ui.FragmentProgram.fromAsset(
        'shaders/profile_background.frag',
      );
      program.fragmentShader().dispose();
      await ProfileShaderBackground.prepareAssets();
    });
    final stills = <List<int>>[];
    for (final background in profileShaderBackgrounds.keys) {
      await tester.pumpWidget(_subject(background));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      final first = await _pixels(tester);
      stills.add(first);
      await tester.pump(const Duration(seconds: 3));
      expect(
        await _pixels(tester),
        isNot(orderedEquals(first)),
        reason: background,
      );
    }
    for (var i = 0; i < stills.length; i++) {
      for (var j = i + 1; j < stills.length; j++) {
        expect(stills[i], isNot(orderedEquals(stills[j])), reason: '$i and $j');
      }
    }
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('reduced motion and disabled tickers keep a still background', (
    tester,
  ) async {
    await tester.runAsync(ProfileShaderBackground.prepareAssets);
    for (final reduced in [true, false]) {
      await tester.pumpWidget(
        _subject('animated-flow', reduceMotion: reduced, enabled: reduced),
      );
      await tester.pump();
      final first = await _pixels(tester);
      await tester.pump(const Duration(seconds: 3));
      expect(await _pixels(tester), orderedEquals(first));
      expect(tester.binding.transientCallbackCount, 0);
    }
    await tester.pumpWidget(const SizedBox());
  });
}
