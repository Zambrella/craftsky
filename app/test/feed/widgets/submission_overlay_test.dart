import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final scenario in [
    (scheduling: false, copy: 'Publishing your post…'),
    (scheduling: true, copy: 'Scheduling your post…'),
  ]) {
    testWidgets('blocks the full surface and shows ${scenario.copy}', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(800, 600);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Stack(
              fit: StackFit.expand,
              children: [
                const TextButton(onPressed: null, child: Text('Behind')),
                SubmissionBlockingOverlay(scheduling: scenario.scheduling),
              ],
            ),
          ),
        ),
      );

      expect(find.byKey(const Key('submission-modal-barrier')), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text(scenario.copy), findsOneWidget);
      final semantics = tester.getSemantics(find.text(scenario.copy));
      expect(semantics.label, contains(scenario.copy));
    });
  }
}
