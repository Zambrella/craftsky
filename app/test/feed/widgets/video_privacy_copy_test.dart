import 'dart:typed_data';

import 'package:craftsky_app/feed/composer/video_failure_message.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/l10n/generated/app_localizations_en.dart';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  test(
    'upload timeout explains that encoding succeeded and the draft is retained',
    () {
      final message = videoPublicationFailureMessage(
        AppLocalizationsEn(),
        const VideoPublicationException(VideoServiceOutcome.uploadTimedOut),
      );
      expect(message, contains('saving it to your account timed out'));
      expect(message, contains('Your draft is safe'));
      expect(message, isNot(contains('allowance')));
    },
  );
  testWidgets('selected video card omits publication subtext', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ComposerVideoAttachmentCard(
            selection: LocalVideoSelection(
              displayName: 'public.mp4',
              mimeType: 'video/mp4',
              byteLength: 12,
              duration: null,
              headerBytes: Uint8List(0),
              openRead: () => const Stream.empty(),
            ),
            enabled: true,
            onAltTextChanged: (_) {},

            onRemove: () {},
          ),
        ),
      ),
    );

    expect(find.byKey(const Key('composer-video-public-notice')), findsNothing);
    expect(
      find.textContaining('Published videos are public'),
      findsNothing,
    );
  });
}
