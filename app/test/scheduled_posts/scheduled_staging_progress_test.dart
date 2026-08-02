import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/widgets/scheduled_staging_progress.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-004 announces per-image private staging progress', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    await tester.pumpWidget(
      const MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ScheduledStagingProgress(completed: 1, total: 3),
        ),
      ),
    );

    expect(find.text('Preparing image 2 of 3'), findsOneWidget);
    expect(
      tester
          .getSemantics(find.text('Preparing image 2 of 3'))
          .flagsCollection
          .isLiveRegion,
      isTrue,
    );
    semantics.dispose();
  });
}
