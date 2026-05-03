import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('StitchProgressIndicator', () {
    testWidgets('renders at the requested size', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: Center(child: StitchProgressIndicator(size: 48)),
          ),
        ),
      );

      // Pump a single frame; do NOT use pumpAndSettle (later tasks will add a
      // repeating animation that never settles).
      await tester.pump();

      final size = tester.getSize(find.byType(StitchProgressIndicator));
      expect(size, const Size(48, 48));
    });

    testWidgets('advances rotationTurns over time', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: Center(child: StitchProgressIndicator())),
        ),
      );

      final initialState =
          tester.state<State<StitchProgressIndicator>>(
                find.byType(StitchProgressIndicator),
              )
              as StitchProgressIndicatorStateForTesting;
      expect(initialState.rotationTurns, 0);

      // Advance well past zero but less than a full cycle so we can assert
      // the value has moved without worrying about wrap-around.
      await tester.pump(const Duration(milliseconds: 700));

      expect(initialState.rotationTurns, greaterThan(0));
      expect(initialState.rotationTurns, lessThan(1));
    });

    testWidgets('disposes its ticker when unmounted', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(body: StitchProgressIndicator()),
        ),
      );
      // Replacing the tree should dispose the State; if the AnimationController
      // is leaked, flutter_test will fail the test.
      await tester.pumpWidget(const MaterialApp(home: SizedBox.shrink()));
    });
  });
}
