import 'dart:ui' show Tristate;

import 'package:craftsky_app/feed/models/video_thumbnail_frame.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/composer_video_attachment_card.dart';
import 'package:craftsky_app/feed/widgets/composer_video_details_page.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as image;
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

  testWidgets(
    'automatic frame grid resumes after clearing a custom thumbnail',
    (
      tester,
    ) async {
      final semantics = tester.ensureSemantics();
      const channel = MethodChannel('social.craftsky/local_video');
      final messenger =
          TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
      final frame = image.encodePng(image.Image(width: 2, height: 2));
      final calls = <MethodCall>[];
      messenger.setMockMethodCallHandler(channel, (call) async {
        calls.add(call);
        return frame;
      });
      addTearDown(() => messenger.setMockMethodCallHandler(channel, null));
      final changes = <Uint8List?>[];
      final selections = <VideoThumbnailFrame>[];
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: ComposerVideoDetailsPage(
              selection: LocalVideoSelection(
                displayName: 'knitting.mp4',
                mimeType: 'video/mp4',
                byteLength: 12,
                duration: const Duration(seconds: 9),
                headerBytes: Uint8List(0),
                openRead: () => const Stream<List<int>>.empty(),
                localUri: Uri.file('/tmp/knitting.mp4'),
                thumbnail: frame,
              ),
              onAltTextChanged: (_) {},
              onThumbnailChanged: changes.add,
              onThumbnailFrameChanged: selections.add,
              checkLimits: () async => const VideoUploadLimits(
                canUpload: true,
                customThumbnail: true,
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(calls.map((call) => call.method), List.filled(4, 'midpointFrame'));
      expect(calls.map((call) => call.arguments), [
        for (final timeUs in [0, 2250000, 4500000, 6750000])
          {'path': 'file:///tmp/knitting.mp4', 'timeUs': timeUs},
      ]);
      expect(find.image(MemoryImage(frame)), findsOneWidget);
      expect(changes, isEmpty);

      final removeThumbnail = find.widgetWithText(
        TextButton,
        'Remove custom thumbnail',
      );
      expect(find.byType(OutlinedButton), findsNothing);
      expect(
        tester
            .widget<TextButton>(removeThumbnail)
            .style!
            .foregroundColor!
            .resolve({}),
        AppTheme.lightThemeData.colorScheme.error,
      );
      await tester.ensureVisible(removeThumbnail);
      await tester.tap(removeThumbnail);
      await tester.pumpAndSettle();
      expect(changes, [null]);
      expect(find.byType(Image), findsNWidgets(4));
      Finder frameChoice(int percent) =>
          find.bySemanticsLabel('Frame at $percent%');
      Tristate selected(int percent) =>
          tester.getSemantics(frameChoice(percent)).flagsCollection.isSelected;
      expect(selected(50), Tristate.isTrue);
      for (final percent in [0, 25, 50, 75]) {
        expect(find.text('$percent%'), findsNothing);
      }
      final zero = tester.getRect(frameChoice(0));
      final quarter = tester.getRect(frameChoice(25));
      final midpoint = tester.getRect(frameChoice(50));
      final last = tester.getRect(frameChoice(75));
      expect(zero.top, quarter.top);
      expect(midpoint.top, last.top);
      expect(midpoint.top, greaterThan(zero.top));
      await tester.ensureVisible(frameChoice(75));
      await tester.tap(frameChoice(75));
      await tester.pumpAndSettle();
      expect(selected(75), Tristate.isTrue);
      expect(selected(50), Tristate.isFalse);
      expect(selections, [VideoThumbnailFrame.threeQuarters]);
      expect(changes, [null]);
      final addThumbnail = find.widgetWithText(
        OutlinedButton,
        'Add custom thumbnail',
      );
      expect(addThumbnail, findsOneWidget);
      final addRect = tester.getRect(addThumbnail);
      expect(
        addRect.width,
        closeTo(tester.getSize(find.text('Thumbnail')).width, 0.01),
      );
      expect(quarter.right - zero.left, lessThan(addRect.width));
      expect(
        (quarter.right + zero.left) / 2,
        closeTo(addRect.center.dx, 0.01),
      );
      semantics.dispose();
    },
  );
}
