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
  });
}
