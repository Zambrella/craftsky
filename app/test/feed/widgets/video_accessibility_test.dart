import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-011 exposes video alt text as semantics and context', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();

    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: NativeVideoPlayer(
            video: PostVideo(
              cid: 'bafyvideo',
              mime: 'video/mp4',
              size: 10,
              alt: 'A wheel spinning blue wool',
            ),
          ),
        ),
      ),
    );

    expect(
      tester.getSemantics(find.byType(NativeVideoPlayer)).label,
      contains('A wheel spinning blue wool'),
    );
    expect(find.text('A wheel spinning blue wool'), findsOneWidget);
    semantics.dispose();
  });
}
