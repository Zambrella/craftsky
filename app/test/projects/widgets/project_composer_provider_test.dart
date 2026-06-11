import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/create_post_provider.dart';
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
  testWidgets('IT-004 resets provider after successful project create', (
    tester,
  ) async {
    PostReply? submittedReply;
    Object? submittedProject;
    final messenger = RecordingMessenger();
    final container = _container(
      FakePostRepository(
        onCreateWithFacets:
            ({required text, reply, project, images, facets}) async {
              submittedReply = reply;
              submittedProject = project;
              return _post(text);
            },
      ),
    );
    addTearDown(container.dispose);

    await _pumpComposer(tester, container: container, messenger: messenger);
    await _submitValidEmbroideryProject(tester);

    expect(submittedReply, isNull);
    expect(submittedProject, isNotNull);
    expect(container.read(createPostProvider).value, isNull);
    expect(messenger.calls, contains(('info', 'Posted.', null)));
  });

  testWidgets(
    'IT-004 resets provider and allows retry after project create error',
    (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      final container = _container(
        FakePostRepository(
          onCreateWithFacets:
              ({required text, reply, project, images, facets}) async {
                throw StateError('temporary failure');
              },
        ),
      );
      addTearDown(container.dispose);

      await _pumpComposer(tester, container: container, messenger: messenger);
      await _submitValidEmbroideryProject(tester);

      expect(container.read(createPostProvider).value, isNull);
      expect(messenger.calls, contains(('error', "Couldn't post.", null)));
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, 'Post'))
            .onPressed,
        isNotNull,
      );
    },
  );
}

ProviderContainer _container(FakePostRepository repo) {
  return ProviderContainer.test(
    overrides: [
      composerImagesProvider('provider-composer').overrideWithValue(
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
  );
}

Future<void> _pumpComposer(
  WidgetTester tester, {
  required ProviderContainer container,
  required RecordingMessenger messenger,
}) async {
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ProjectComposerSheet(composerId: 'provider-composer'),
        ),
      ),
    ),
  );
}

Future<void> _submitValidEmbroideryProject(WidgetTester tester) async {
  await tester.enterText(find.byType(TextField).first, 'Finished project');
  await tester.tap(find.byType(DropdownButton<String>).first);
  await tester.pumpAndSettle();
  await tester.tap(find.text('Embroidery').last);
  await tester.pumpAndSettle();
  await tester.tap(find.widgetWithText(TextButton, 'Post'));
  await tester.pumpAndSettle();
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
