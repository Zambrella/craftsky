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
  String colour = 'terracotta',
  bool reduceMotion = false,
  bool enabled = true,
  bool includeBase = true,
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
            child: includeBase
                ? ProfileHeaderBackground(
                    customisation: ProfileCustomisation(
                      colour: colour,
                      background: background,
                    ),
                  )
                : ProfileShaderBackground(
                    style: profileShaderBackgrounds[background]!,
                    accent: ui.Color(
                      int.parse(
                            profileColourBundles[colour]!.base.substring(1),
                            radix: 16,
                          ) |
                          0xFF000000,
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

  testWidgets(
    'bright reference styles follow the accent and respect reduced motion',
    (tester) async {
      await tester.runAsync(ProfileShaderBackground.prepareAssets);
      await tester.pumpWidget(
        _subject('animated-ribbons', colour: 'cobalt', reduceMotion: true),
      );
      await tester.pump();
      final cobalt = await _pixels(tester);
      await tester.pump(const Duration(seconds: 3));
      expect(await _pixels(tester), orderedEquals(cobalt));
      await tester.pumpWidget(
        _subject('animated-ribbons', colour: 'seafoam', reduceMotion: true),
      );
      await tester.pump();
      expect(await _pixels(tester), isNot(orderedEquals(cobalt)));
      expect(tester.binding.transientCallbackCount, 0);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'translucent pigments follow the selected accent without the base layer',
    (
      tester,
    ) async {
      await tester.runAsync(ProfileShaderBackground.prepareAssets);
      for (final background in profileShaderBackgrounds.keys.take(6)) {
        final samples = <List<int>>[];
        for (final colour in ['cobalt', 'seafoam', 'terracotta', 'ink']) {
          await tester.pumpWidget(
            _subject(
              background,
              colour: colour,
              reduceMotion: true,
              includeBase: false,
            ),
          );
          await tester.pump();
          samples.add(await _pixels(tester));
        }
        for (var i = 1; i < samples.length; i++) {
          expect(
            samples[i],
            isNot(orderedEquals(samples[0])),
            reason: background,
          );
        }
        final neutral = samples.last;
        for (var i = 0; i < neutral.length; i += 4) {
          expect(
            (neutral[i] - neutral[i + 1]).abs(),
            lessThanOrEqualTo(1),
            reason: background,
          );
          expect(
            (neutral[i + 1] - neutral[i + 2]).abs(),
            lessThanOrEqualTo(1),
            reason: background,
          );
        }
      }
      await tester.pumpWidget(const SizedBox());
    },
  );

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
