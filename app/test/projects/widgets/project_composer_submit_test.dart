import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-004 submits a valid common-only embroidery project', (
    tester,
  ) async {
    String? capturedText;
    PostReply? capturedReply;
    Project? capturedProject;
    List<CreatePostImage>? capturedImages;
    List<Map<String, dynamic>>? capturedFacets;
    final repo = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            capturedText = text;
            capturedReply = reply;
            capturedProject = project;
            capturedImages = images;
            capturedFacets = facets;
            return _post(text: text, project: project);
          },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('project-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'hoop.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Finished embroidery hoop on a table',
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
            home: const ProjectComposerSheet(composerId: 'project-composer'),
          ),
        ),
      ),
    );

    await tester.enterText(
      _bodyTextField(),
      'Finished my hoop #embroidery',
    );
    await _selectCraft(tester, 'Embroidery');
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(capturedText, 'Finished my hoop #embroidery');
    expect(capturedReply, isNull);
    expect(capturedImages, hasLength(1));
    expect(capturedImages!.single.alt, 'Finished embroidery hoop on a table');
    expect(capturedProject, isNotNull);
    expect(
      capturedProject!.common.craftType,
      ProjectOptionCatalogs.embroideryCraftToken,
    );
    expect(capturedProject!.details, isNull);
    expect(capturedProject!.common.title, isNull);
    expect(capturedProject!.common.materials, isNull);
    expect(capturedProject!.common.colors, isNull);
    expect(capturedProject!.common.designTags, isNull);
    expect(capturedFacets, isNotNull);
    expect(
      capturedFacets!.expand((facet) => facet['features']! as List<dynamic>),
      contains(
        predicate<Object?>(
          (feature) =>
              feature is Map<String, dynamic> &&
              feature[r'$type'] == 'app.bsky.richtext.facet#tag' &&
              feature['tag'] == 'embroidery',
          'tag facet for #embroidery',
        ),
      ),
    );
  });

  testWidgets('AT-007 submits non-empty metadata and pattern fields', (
    tester,
  ) async {
    Project? capturedProject;
    final repo = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            capturedProject = project;
            return _post(text: text, project: project);
          },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('metadata-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'dress.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Blue handmade dress',
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
            home: const ProjectComposerSheet(composerId: 'metadata-composer'),
          ),
        ),
      ),
    );

    await tester.enterText(_bodyTextField(), 'Finished a dress');
    await _selectCraft(tester, 'Sewing');

    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.materials}-custom-input')),
    );
    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.materials}-custom-input')),
      'Cotton lawn',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.materials}-add-custom')),
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('${ProjectComposerFields.materials}-add-custom')),
    );
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.colours}-search-input')),
      'Blue',
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(const Key('${ProjectComposerFields.colours}-option-blue')),
    );
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('${ProjectComposerFields.designTags}-search-input')),
    );
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('${ProjectComposerFields.designTags}-search-input')),
      'Floral',
    );
    await tester.pumpAndSettle();
    await tester.tap(
      find.byKey(
        const Key(
          '${ProjectComposerFields.designTags}-option-'
          'social.craftsky.project.defs#floral',
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Pattern'));
    await tester.tap(find.text('Pattern'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('pattern-name-input')),
      'Garden dress',
    );
    await tester.enterText(
      find.byKey(const Key('pattern-url-input')),
      'https://patterns.example/garden-dress',
    );

    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(capturedProject, isNotNull);
    final common = capturedProject!.common;
    expect(common.materials, ['Cotton lawn']);
    expect(common.colors, ['blue']);
    expect(common.designTags, ['social.craftsky.project.defs#floral']);
    expect(common.pattern, isNotNull);
    expect(common.pattern!.name, 'Garden dress');
    expect(common.pattern!.url, 'https://patterns.example/garden-dress');
  });
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

Finder _bodyTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-body-editor')),
    matching: find.byType(TextField),
  );
}

Future<void> _selectCraft(WidgetTester tester, String craftLabel) async {
  final craftDropdown = find.byType(DropdownButton<String>).first;
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text(craftLabel).last);
  await tester.pumpAndSettle();
}

Post _post({required String text, Project? project}) {
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
    project: project,
  );
}
