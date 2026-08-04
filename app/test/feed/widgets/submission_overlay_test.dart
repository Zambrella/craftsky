import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
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
          home: Stack(
            fit: StackFit.expand,
            children: [
              const Scaffold(
                body: TextButton(onPressed: null, child: Text('Behind')),
              ),
              SubmissionBlockingOverlay(scheduling: scenario.scheduling),
            ],
          ),
        ),
      );

      final statusText = find.text(scenario.copy);
      final statusContext = tester.element(statusText);
      final theme = Theme.of(statusContext);

      expect(find.byKey(const Key('submission-modal-barrier')), findsOneWidget);
      expect(find.byType(StitchProgressIndicator), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsNothing);
      expect(
        find.ancestor(of: statusText, matching: find.byType(Material)),
        findsOneWidget,
      );
      expect(
        DefaultTextStyle.of(statusContext).style.color,
        theme.textTheme.bodyMedium?.color,
      );
      expect(statusText, findsOneWidget);
      final semantics = tester.getSemantics(statusText);
      expect(semantics.label, contains(scenario.copy));
    });
  }
}
