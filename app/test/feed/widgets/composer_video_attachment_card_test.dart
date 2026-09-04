import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-007 video card edits alt text and exposes explicit actions', (
    tester,
  ) async {
    String? altText;
    var replaced = false;
    var removed = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ComposerVideoAttachmentCard(
            selection: LocalVideoSelection(
              displayName: 'knitting.mp4',
              mimeType: 'video/mp4',
              byteLength: 12,
              duration: null,
              headerBytes: Uint8List(0),
              openRead: () => const Stream<List<int>>.empty(),
            ),
            enabled: true,
            onAltTextChanged: (value) => altText = value,
            onReplace: () => replaced = true,
            onRemove: () => removed = true,
          ),
        ),
      ),
    );

    expect(find.text('knitting.mp4'), findsOneWidget);
    expect(find.byType(BrandTextField), findsOneWidget);
    expect(find.byKey(const Key('composer-video-public-notice')), findsNothing);
    await tester.enterText(
      find.byKey(const Key('composer-video-alt-text')),
      'Hands knitting a blue scarf',
    );
    await tester.tap(find.byKey(const Key('composer-replace-video')));
    await tester.tap(find.byKey(const Key('composer-remove-video')));

    expect(altText, 'Hands knitting a blue scarf');
    expect(replaced, isTrue);
    expect(removed, isTrue);
  });
}
