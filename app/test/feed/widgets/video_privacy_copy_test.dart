import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('selected video card omits publication subtext', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
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
            onReplace: () {},
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
