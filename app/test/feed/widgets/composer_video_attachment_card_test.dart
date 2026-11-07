import 'dart:typed_data';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/feed/widgets/composer_video_details_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('AT-007 video card edits alt text and exposes explicit actions', (
    tester,
  ) async {
    String? altText;
    var removed = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
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
            previewBuilder: (_, _) => const Text('Local player'),
            onAltTextChanged: (value) => altText = value,

            onRemove: () => removed = true,
          ),
        ),
      ),
    );

    expect(find.text('knitting.mp4'), findsOneWidget);
    expect(find.byType(BrandTextField), findsNothing);
    await tester.tap(find.byKey(const Key('composer-inspect-video')));
    await tester.pumpAndSettle();
    expect(find.byType(ComposerVideoDetailsPage), findsOneWidget);
    expect(find.text('Local player'), findsOneWidget);
    expect(find.byType(BrandTextField), findsOneWidget);
    expect(find.byKey(const Key('composer-video-public-notice')), findsNothing);
    await tester.enterText(
      find.byKey(const Key('composer-video-alt-text')),
      'Hands knitting a blue scarf',
    );
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('composer-replace-video')), findsNothing);
    await tester.tap(find.byKey(const Key('composer-remove-video')));

    expect(altText, 'Hands knitting a blue scarf');
    expect(removed, isTrue);
  });
}
