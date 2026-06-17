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
  testWidgets('AT-006 blocks page one next until required fields are filled', (
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

    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();

    expect(find.text('Choose a craft type.'), findsOneWidget);
    expect(find.text('Add at least one photo.'), findsOneWidget);
    expect(find.text('Materials'), findsNothing);
  });

  testWidgets('AT-006 blocks final submission without caption text', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('body-validation-composer').overrideWithValue(
            _readyImagesState,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(
              composerId: 'body-validation-composer',
            ),
          ),
        ),
      ),
    );

    await _selectCraft(tester, 'Embroidery');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(find.text('Add body text.'), findsOneWidget);
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

    await _selectCraft(tester, 'Knitting');
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('knitting-gauge-stitches-input')),
    );
    await tester.enterText(
      find.byKey(const Key('knitting-gauge-stitches-input')),
      '20',
    );
    await tester.tap(find.widgetWithText(TextButton, 'Next'));
    await tester.pumpAndSettle();
    await tester.enterText(_bodyTextField(), 'Finished swatch');
    await _pumpUntilPostEnabled(tester);
    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(find.text('Complete the gauge or clear it.'), findsOneWidget);
    expect(createCalls, 0);
  });
}

const _readyImagesState = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'image-1',
      fileName: 'project.jpg',
      mimeType: 'image/jpeg',
      altText: 'Finished project photo',
      phase: ImageUploaded(
        UploadedDraftImage(cid: 'bafkimage', mime: 'image/jpeg', size: 123),
      ),
    ),
  ],
);

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

Finder _bodyTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-body-editor')),
    matching: find.byType(TextField),
  );
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

Future<void> _pumpUntilPostEnabled(WidgetTester tester) async {
  for (var i = 0; i < 200; i += 1) {
    await tester.pump(const Duration(milliseconds: 20));
    final button = tester.widget<TextButton>(
      find.widgetWithText(TextButton, 'Post'),
    );
    if (button.onPressed != null) return;
  }
  fail('Timed out waiting for Post button to be enabled');
}
