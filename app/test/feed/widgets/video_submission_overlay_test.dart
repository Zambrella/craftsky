import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-004 shows video stage, progress, and cancellation', (
    tester,
  ) async {
    var canceled = false;
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: Stack(
            children: [
              SubmissionBlockingOverlay(
                scheduling: false,
                videoProgress: const VideoPublicationProgress(
                  VideoPublicationStage.uploading,
                  fraction: 0.5,
                ),
                onCancelVideo: () => canceled = true,
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.text('Uploading video'), findsOneWidget);
    expect(
      tester
          .widget<LinearProgressIndicator>(
            find.byType(LinearProgressIndicator),
          )
          .value,
      0.5,
    );
    await tester.tap(find.byKey(const Key('cancel-video-publication')));
    expect(canceled, isTrue);
  });
}
