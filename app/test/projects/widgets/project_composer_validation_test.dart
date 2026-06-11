import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-006 blocks empty required project submission', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(),
          ),
        ),
      ),
    );

    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(find.text('Add body text.'), findsOneWidget);
    expect(find.text('Choose a craft type.'), findsOneWidget);
    expect(find.text('Add at least one photo.'), findsOneWidget);
  });

  testWidgets('AT-006 blocks partial knitting gauge input', (tester) async {
    var createCalls = 0;
    final repo = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            createCalls += 1;
            return _post(text);
          },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('validation-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'knit.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Knitted swatch',
                  phase: ImageUploaded(
                    UploadedDraftImage(
                      cid: 'bafkimage',
                      mime: 'image/jpeg',
                      size: 123,
                    ),
                  ),
                ),
              ],
            ),
          ),
          postRepositoryProvider.overrideWithValue(repo),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'validation-composer'),
          ),
        ),
      ),
    );

    await tester.enterText(find.byType(TextField).first, 'Finished swatch');
    await tester.tap(find.byType(DropdownButton<String>).first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Knitting').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('More project details'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('More project details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('knitting-gauge-stitches-input')),
    );
    await tester.enterText(
      find.byKey(const Key('knitting-gauge-stitches-input')),
      '20',
    );
    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(find.text('Complete the gauge or clear it.'), findsOneWidget);
    expect(createCalls, 0);
  });
}

Post _post(String text) {
  final now = DateTime.utc(2026, 6, 11);
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafyreibazjzrzibga2jwt5co2yus7j2w6p3n3cb6nn4njvkzcxwrlfvula',
    rkey: '3lf2abc',
    text: text,
    tags: const [],
    createdAt: now,
    indexedAt: now,
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
    likeCount: 0,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
  );
}
