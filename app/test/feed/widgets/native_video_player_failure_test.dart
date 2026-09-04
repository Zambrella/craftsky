import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-009 missing HLS fails soft with stable layout', (
    tester,
  ) async {
    final theme = ThemeData(extensions: const [RadiusTheme()]);
    await tester.pumpWidget(
      MaterialApp(
        theme: theme,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: NativeVideoPlayer(
            video: PostVideo(
              cid: 'bafyvideo',
              mime: 'video/mp4',
              size: 10,
              alt: 'A wheel spinning yarn',
              aspectRatio: const PostImageAspectRatio(width: 4, height: 3),
            ),
          ),
        ),
      ),
    );

    expect(find.text('Video is unavailable. Try again later.'), findsOneWidget);
    expect(find.text('A wheel spinning yarn'), findsOneWidget);
    expect(tester.getSize(find.byType(AspectRatio)).height, greaterThan(0));
    final outline = tester.widget<DecoratedBox>(
      find.byKey(const Key('native-video-outline')),
    );
    final decoration = outline.decoration as BoxDecoration;
    expect(decoration.borderRadius, BorderRadius.circular(6));
    expect(
      (decoration.border! as Border).top.color,
      theme.colorScheme.outlineVariant,
    );
    expect(
      tester.widget<ClipRRect>(find.byKey(const Key('native-video-clip'))),
      isA<ClipRRect>()
          .having(
            (clip) => clip.borderRadius,
            'borderRadius',
            BorderRadius.circular(6),
          )
          .having((clip) => clip.clipBehavior, 'clipBehavior', Clip.antiAlias),
    );
  });
}
