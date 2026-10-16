import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/publishing_progress_indicator.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:rive/rive.dart' as rive;

void main() {
  testWidgets('pending or failed asset keeps a visible fallback', (
    tester,
  ) async {
    final pending = Completer<rive.File>();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          publishingAnimationFileProvider.overrideWith((ref) => pending.future),
        ],
        child: const MaterialApp(
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Center(child: PublishingProgressIndicator()),
        ),
      ),
    );
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(
      tester.getSize(find.byType(PublishingProgressIndicator)),
      const Size(48, 48),
    );
    pending.completeError(const FormatException('Broken test animation'));
    await tester.pump();
    await tester.pump();
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'bundled animation plays, freezes for reduced motion, and resumes',
    (tester) async {
      final file = await tester.runAsync(
        () => rive.File.asset(
          'assets/animations/publishing.riv',
          riveFactory: rive.Factory.flutter,
        ),
      );
      expect(
        file,
        isNotNull,
        reason: 'The shipped asset must decode in Flutter',
      );
      addTearDown(() => file!.dispose());
      final container = ProviderContainer(
        overrides: [
          publishingAnimationFileProvider.overrideWith((ref) async => file!),
        ],
      );
      addTearDown(container.dispose);
      await container.read(publishingAnimationFileProvider.future);

      const captureKey = Key('animation-capture');
      Future<Uint8List> pixels() async => (await tester.runAsync(() async {
        final boundary = tester.renderObject<RenderRepaintBoundary>(
          find.byKey(captureKey),
        );
        final image = await boundary.toImage(pixelRatio: 3);
        final data = await image.toByteData();
        image.dispose();
        return data!.buffer.asUint8List();
      }))!;
      bool hasColor(Uint8List bytes, Color color) {
        final argb = color.toARGB32();
        for (var i = 0; i < bytes.length; i += 4) {
          if (bytes[i] == (argb >> 16 & 255) &&
              bytes[i + 1] == (argb >> 8 & 255) &&
              bytes[i + 2] == (argb & 255) &&
              bytes[i + 3] == 255) {
            return true;
          }
        }
        return false;
      }

      Future<void> show({
        required bool reduceMotion,
        Color primary = BrandColors.cobalt,
      }) => tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: ThemeData(colorScheme: ColorScheme.light(primary: primary)),
            home: MediaQuery(
              data: MediaQueryData(disableAnimations: reduceMotion),
              child: const Center(
                child: RepaintBoundary(
                  key: captureKey,
                  child: PublishingProgressIndicator(),
                ),
              ),
            ),
          ),
        ),
      );

      await show(reduceMotion: false);
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.byType(rive.RiveWidget), findsOneWidget);
      expect(find.byType(StitchProgressIndicator), findsNothing);
      var controller = tester
          .widget<rive.RiveWidget>(find.byType(rive.RiveWidget))
          .controller;
      expect(controller.artboard.name, 'Publishing');
      expect(controller.active, isTrue);
      final firstFrame = await pixels();
      expect(
        firstFrame[3],
        0,
        reason: 'The artboard background is transparent',
      );
      expect(hasColor(firstFrame, BrandColors.cobalt), isTrue);
      expect(hasColor(firstFrame, BrandColors.red), isTrue);
      await tester.pump(const Duration(milliseconds: 300));
      expect(await pixels(), isNot(orderedEquals(firstFrame)));

      await show(reduceMotion: true);
      controller = tester
          .widget<rive.RiveWidget>(find.byType(rive.RiveWidget))
          .controller;
      expect(controller.active, isFalse);
      expect(
        TickerMode.valuesOf(
          tester.element(find.byType(rive.RiveWidget)),
        ).enabled,
        isFalse,
      );
      final frozenFrame = await pixels();
      await tester.pump(const Duration(seconds: 1));
      expect(await pixels(), orderedEquals(frozenFrame));
      expect(tester.binding.hasScheduledFrame, isFalse);

      const darkCobalt = Color(0xFF7890FF);
      await show(reduceMotion: true, primary: darkCobalt);
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
      expect(hasColor(await pixels(), darkCobalt), isTrue);
      expect(hasColor(await pixels(), BrandColors.red), isTrue);

      await show(reduceMotion: false);
      controller = tester
          .widget<rive.RiveWidget>(find.byType(rive.RiveWidget))
          .controller;
      expect(controller.active, isTrue);
      await tester.pumpWidget(const SizedBox());
      expect(tester.takeException(), isNull);
    },
  );
}
