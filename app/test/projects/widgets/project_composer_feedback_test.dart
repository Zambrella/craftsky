import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';
import '../../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets(
    'AT-008 confirms before submitting project images without alt text',
    (
      tester,
    ) async {
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
            composerImagesProvider('feedback-composer').overrideWithValue(
              const ComposerImagesState(
                images: [
                  ComposerImageDraft(
                    id: 'image-1',
                    fileName: 'project.jpg',
                    mimeType: 'image/jpeg',
                    altText: '',
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
              home: const ProjectComposerSheet(composerId: 'feedback-composer'),
            ),
          ),
        ),
      );

      await _selectEmbroidery(tester);
      await _goNext(tester);
      await _goNext(tester);
      await tester.enterText(_bodyTextField(), 'Finished project');
      await _pumpUntilPostEnabled(tester);

      await tester.tap(find.widgetWithText(TextButton, 'Post'));
      await tester.pumpAndSettle();

      expect(find.text('Some images do not have alt text'), findsOneWidget);
      expect(find.text('Do you wish to post anyway?'), findsOneWidget);
      expect(createCalls, 0);

      await tester.tap(find.text('Post anyway'));
      await tester.pumpAndSettle();

      expect(createCalls, 1);
    },
  );

  testWidgets('AT-008 disables controls while project create is loading', (
    tester,
  ) async {
    final createGate = Completer<Post>();
    final repo = FakePostRepository(
      onCreateWithFacets: ({required text, reply, project, images, facets}) {
        return createGate.future;
      },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('loading-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'project.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Finished project photo',
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
            home: const ProjectComposerSheet(composerId: 'loading-composer'),
          ),
        ),
      ),
    );

    await _selectEmbroidery(tester);
    await _goNext(tester);
    await _goNext(tester);
    await tester.enterText(_bodyTextField(), 'Finished project');
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pump();

    expect(
      tester
          .widget<TextButton>(find.widgetWithText(TextButton, 'Post'))
          .onPressed,
      isNull,
    );
    expect(
      tester.widget<TextField>(_bodyTextField()).enabled,
      isFalse,
    );
    expect(
      tester
          .widget<InkWell>(
            find.byKey(const Key('composer-add-image'), skipOffstage: false),
          )
          .onTap,
      isNull,
    );
    expect(
      tester
          .widget<BrandTextField>(
            find.byKey(const Key('composer-alt-image-1'), skipOffstage: false),
          )
          .enabled,
      isFalse,
    );
    createGate.complete(_post('Finished project'));
    await tester.pumpAndSettle();
  });

  testWidgets('AT-008 closes and shows success after project create succeeds', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    final repo = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            return _post(text);
          },
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          composerImagesProvider('success-composer').overrideWithValue(
            const ComposerImagesState(
              images: [
                ComposerImageDraft(
                  id: 'image-1',
                  fileName: 'project.jpg',
                  mimeType: 'image/jpeg',
                  altText: 'Finished project photo',
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
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Builder(
              builder: (context) => TextButton(
                onPressed: () {
                  unawaited(
                    Navigator.of(context).push<void>(
                      MaterialPageRoute<void>(
                        builder: (_) => const ProjectComposerSheet(
                          composerId: 'success-composer',
                        ),
                      ),
                    ),
                  );
                },
                child: const Text('Open composer'),
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open composer'));
    await tester.pumpAndSettle();
    await _selectEmbroidery(tester);
    await _goNext(tester);
    await _goNext(tester);
    await tester.enterText(_bodyTextField(), 'Finished project');
    await _pumpUntilPostEnabled(tester);

    await tester.tap(find.widgetWithText(TextButton, 'Post'));
    await tester.pumpAndSettle();

    expect(find.text('Project post'), findsNothing);
    expect(messenger.calls, contains(('info', 'Posted.', null)));
  });

  testWidgets(
    'AT-008 shows error and allows retry after project create fails',
    (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      var createCalls = 0;
      final repo = FakePostRepository(
        onCreateWithFacets:
            ({required text, reply, project, images, facets}) async {
              createCalls += 1;
              if (createCalls == 1) {
                throw StateError('temporary failure');
              }
              return _post(text);
            },
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            composerImagesProvider('error-composer').overrideWithValue(
              const ComposerImagesState(
                images: [
                  ComposerImageDraft(
                    id: 'image-1',
                    fileName: 'project.jpg',
                    mimeType: 'image/jpeg',
                    altText: 'Finished project photo',
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
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Builder(
                builder: (context) => TextButton(
                  onPressed: () {
                    unawaited(
                      Navigator.of(context).push<void>(
                        MaterialPageRoute<void>(
                          builder: (_) => const ProjectComposerSheet(
                            composerId: 'error-composer',
                          ),
                        ),
                      ),
                    );
                  },
                  child: const Text('Open composer'),
                ),
              ),
            ),
          ),
        ),
      );

      await tester.tap(find.text('Open composer'));
      await tester.pumpAndSettle();
      await _selectEmbroidery(tester);
      await _goNext(tester);
      await _goNext(tester);
      await tester.enterText(_bodyTextField(), 'Finished project');
      await _pumpUntilPostEnabled(tester);

      await tester.tap(find.widgetWithText(TextButton, 'Post'));
      await tester.pumpAndSettle();

      expect(find.text('Project post'), findsOneWidget);
      expect(messenger.calls, contains(('error', "Couldn't post.", null)));

      await tester.tap(find.widgetWithText(TextButton, 'Post'));
      await tester.pumpAndSettle();

      expect(createCalls, 2);
      expect(find.text('Project post'), findsNothing);
      expect(messenger.calls, contains(('info', 'Posted.', null)));
    },
  );
}

Finder _bodyTextField() {
  return find.descendant(
    of: find.byKey(const Key('project-composer-body-editor')),
    matching: find.byType(TextField),
  );
}

Future<void> _selectEmbroidery(WidgetTester tester) async {
  final craftDropdown = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(craftDropdown);
  await tester.pumpAndSettle();
  await tester.tap(find.text('Embroidery').last);
  await tester.pumpAndSettle();
}

Future<void> _goNext(WidgetTester tester) async {
  await tester.tap(find.widgetWithText(TextButton, 'Next'));
  await tester.pumpAndSettle();
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
    viewerHasSaved: false,
  );
}
