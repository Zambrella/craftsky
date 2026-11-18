import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' show PointerDeviceKind;

import 'package:craftsky_app/profile/sharing/share_card_motion.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:sensors_plus/sensors_plus.dart';

void main() {
  testWidgets('opening wobble settles once and does not repeat after sharing', (
    tester,
  ) async {
    late ValueListenable<ShareCardFrame> frames;
    Widget scene({bool enabled = true, bool reduced = false}) => MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reduced),
        child: Center(
          child: ShareCardMotion(
            enabled: enabled,
            gyroscope: () => const Stream.empty(),
            builder: (_, value) {
              frames = value;
              return const SizedBox(width: 200, height: 250);
            },
          ),
        ),
      ),
    );
    await tester.pumpWidget(scene());
    await tester.pump(const Duration(milliseconds: 250));
    await tester.pump(const Duration(milliseconds: 100));
    expect(frames.value.tilt.distance, greaterThan(0));
    expect(
      frames.value.tilt.distance,
      lessThan(math.sqrt(2) * 5 * math.pi / 180),
    );
    await tester.pumpAndSettle();
    expect(frames.value.tilt, Offset.zero);
    expect(tester.binding.hasScheduledFrame, isFalse);
    await tester.pumpWidget(scene(enabled: false));
    await tester.pumpWidget(scene());
    await tester.pump(const Duration(milliseconds: 350));
    expect(frames.value.tilt, Offset.zero);
    await tester.pumpAndSettle();
    expect(tester.binding.hasScheduledFrame, isFalse);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpWidget(scene(reduced: true));
    await tester.pump(const Duration(milliseconds: 350));
    expect(frames.value.tilt, Offset.zero);
    expect(tester.binding.hasScheduledFrame, isFalse);
  });

  for (final kind in [PointerDeviceKind.touch, PointerDeviceKind.mouse]) {
    testWidgets(
      'drag holds tilt, overrides gyro and settles on release ($kind)',
      (tester) async {
        late ValueListenable<ShareCardFrame> frames;
        final sensor = StreamController<GyroscopeEvent>.broadcast();
        addTearDown(sensor.close);
        Widget scene({bool enabled = true, bool reduced = false}) =>
            MaterialApp(
              home: MediaQuery(
                data: MediaQueryData(disableAnimations: reduced),
                child: SingleChildScrollView(
                  child: Column(
                    children: [
                      ShareCardMotion(
                        enabled: enabled,
                        gyroscope: () => sensor.stream,
                        builder: (_, value) {
                          frames = value;
                          return const SizedBox(width: 200, height: 250);
                        },
                      ),
                      const SizedBox(height: 800),
                    ],
                  ),
                ),
              ),
            );
        await tester.pumpWidget(scene());
        final card = find.byKey(const Key('share-card-tilt'));
        final gesture = await tester.startGesture(
          tester.getCenter(card),
          kind: kind,
        );
        await gesture.moveBy(const Offset(30, -30));
        await gesture.moveBy(const Offset(40, -40));
        await tester.pump(const Duration(milliseconds: 100));
        await tester.pump(const Duration(milliseconds: 100));
        expect(frames.value.tilt.dx, greaterThan(0));
        expect(frames.value.tilt.dy, greaterThan(0));
        await tester.pumpAndSettle();
        final held = frames.value.tilt;
        expect(
          held.distance,
          lessThanOrEqualTo(math.sqrt(2) * 5 * math.pi / 180),
        );
        expect(tester.binding.hasScheduledFrame, isFalse);
        final start = DateTime(2026);
        sensor
          ..add(GyroscopeEvent(-100, -100, 0, start))
          ..add(
            GyroscopeEvent(
              -100,
              -100,
              0,
              start.add(const Duration(milliseconds: 32)),
            ),
          );
        await tester.pump(const Duration(milliseconds: 100));
        expect(frames.value.tilt, held);
        expect(tester.getTopLeft(card).dy, closeTo(0, 2));
        await gesture.up();
        await tester.pumpAndSettle();
        expect(frames.value.tilt, Offset.zero);
        expect(tester.binding.hasScheduledFrame, isFalse);
        sensor.add(
          GyroscopeEvent(10, 10, 0, start.add(const Duration(seconds: 5))),
        );
        await tester.pump();
        expect(frames.value.tilt, Offset.zero);
        sensor.add(
          GyroscopeEvent(
            10,
            10,
            0,
            start.add(const Duration(milliseconds: 5032)),
          ),
        );
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));
        await tester.pump(const Duration(milliseconds: 100));
        expect(frames.value.tilt.distance, greaterThan(0));
        await tester.pumpWidget(scene(enabled: false));
        expect(frames.value.tilt, Offset.zero);
        await tester.drag(card, const Offset(40, -40));
        await tester.pumpAndSettle();
        expect(frames.value.tilt, Offset.zero);
        await tester.pumpWidget(scene(reduced: true));
        await tester.drag(card, const Offset(40, -40));
        await tester.pumpAndSettle();
        expect(frames.value.tilt, Offset.zero);
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('tilt is bounded, damped and stops for reduced motion', (
    tester,
  ) async {
    final sensor = StreamController<GyroscopeEvent>.broadcast();
    addTearDown(sensor.close);
    late ValueListenable<ShareCardFrame> frames;
    Widget scene({bool reduced = false}) => MaterialApp(
      home: MediaQuery(
        data: MediaQueryData(disableAnimations: reduced),
        child: ShareCardMotion(
          gyroscope: () => sensor.stream,
          builder: (_, value) {
            frames = value;
            return const SizedBox(width: 100, height: 100);
          },
        ),
      ),
    );
    await tester.pumpWidget(scene());
    await tester.pumpAndSettle();
    expect(sensor.hasListener, isTrue);
    expect(tester.binding.hasScheduledFrame, isFalse);
    final start = DateTime(2026);
    for (var i = 0; i < 12; i++) {
      sensor.add(
        GyroscopeEvent(100, -100, 0, start.add(Duration(milliseconds: i * 32))),
      );
      await tester.pump(const Duration(milliseconds: 32));
    }
    expect(frames.value.tilt.dx, greaterThan(0));
    expect(frames.value.tilt.dy, lessThan(0));
    expect(
      frames.value.tilt.distance,
      lessThanOrEqualTo(math.sqrt(2) * 5 * math.pi / 180),
    );
    final previousTilt = frames.value.tilt.distance;
    for (var i = 0; i < 40; i++) {
      await tester.pump(const Duration(milliseconds: 100));
    }
    expect(frames.value.tilt.distance, lessThan(previousTilt));
    await tester.pumpAndSettle();
    expect(frames.value.tilt, Offset.zero);
    expect(tester.binding.hasScheduledFrame, isFalse);
    await tester.pumpWidget(scene(reduced: true));
    await tester.pumpAndSettle();
    expect(sensor.hasListener, isFalse);
    expect(frames.value.tilt, Offset.zero);
    expect(tester.binding.hasScheduledFrame, isFalse);
  });

  testWidgets(
    'background, covered route, missing sensor and disposal are safe',
    (
      tester,
    ) async {
      final sensor = StreamController<GyroscopeEvent>.broadcast();
      addTearDown(sensor.close);
      final navigator = GlobalKey<NavigatorState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: navigator,
          home: ShareCardMotion(
            gyroscope: () => sensor.stream,
            builder: (_, _) => const Text('Card'),
          ),
        ),
      );
      expect(sensor.hasListener, isTrue);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      await tester.pump();
      expect(sensor.hasListener, isFalse);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump();
      expect(sensor.hasListener, isTrue);
      navigator.currentState!.push<void>(
        MaterialPageRoute(builder: (_) => const Text('Covered')),
      );
      await tester.pumpAndSettle();
      expect(sensor.hasListener, isFalse);
      navigator.currentState!.pop();
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 500));
      expect(sensor.hasListener, isTrue);
      sensor.addError(StateError('Sensor unavailable'));
      await tester.pump();
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      expect(sensor.hasListener, isFalse);
    },
  );
}
