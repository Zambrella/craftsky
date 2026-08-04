import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('AT-002 scheduling is available for original top-level posts', (
    tester,
  ) async {
    await tester.pumpWidget(
      _postApp(const PostComposerSheet(composerId: 'top')),
    );
    await tester.pumpAndSettle();

    expect(find.text('When'), findsOneWidget);
    await tester.tap(find.text('When'));
    await tester.pumpAndSettle();
    expect(find.byType(SimpleDialog), findsNothing);
    expect(find.byType(BottomSheet), findsOneWidget);
    expect(find.text('Schedule for later'), findsOneWidget);
  });

  testWidgets('AT-002 scheduling is available for project posts', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          composerImagesProvider('project').overrideWithValue(
            _readyProjectImage,
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ProjectComposerSheet(composerId: 'project'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await _advanceProjectComposer(tester);
    expect(find.text('When'), findsOneWidget);
    await tester.tap(find.text('When'));
    await tester.pumpAndSettle();
    expect(find.byType(SimpleDialog), findsNothing);
    expect(find.byType(BottomSheet), findsOneWidget);
    expect(find.text('Schedule for later'), findsOneWidget);
  });

  testWidgets('AT-002 scheduling is unavailable for replies and quotes', (
    tester,
  ) async {
    final target = _post();
    await tester.pumpWidget(
      _postApp(PostComposerSheet(composerId: 'reply', replyTarget: target)),
    );
    await tester.pumpAndSettle();
    expect(find.text('When'), findsNothing);
    expect(find.text('Schedule for later'), findsNothing);

    await tester.pumpWidget(
      _postApp(PostComposerSheet(composerId: 'quote', quoteTarget: target)),
    );
    await tester.pumpAndSettle();
    expect(find.text('When'), findsNothing);
    expect(find.text('Schedule for later'), findsNothing);
  });
}

Widget _postApp(Widget home) => ProviderScope(
  overrides: [
    activeLanguagePreferencesProvider.overrideWith(
      (ref) => const LanguagePreferences(
        primaryLanguage: 'en',
        contentLanguages: ['en'],
      ),
    ),
  ],
  child: MessengerScope(
    messenger: RecordingMessenger(),
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: home,
    ),
  ),
);

const _readyProjectImage = ComposerImagesState(
  images: [
    ComposerImageDraft(
      id: 'project-image',
      fileName: 'project.jpg',
      mimeType: 'image/jpeg',
      altText: 'Finished project',
      phase: ImageUploaded(
        UploadedDraftImage(
          cid: 'bafy-project-image',
          mime: 'image/jpeg',
          size: 100,
        ),
      ),
    ),
  ],
);

Future<void> _advanceProjectComposer(WidgetTester tester) async {
  final craftType = find.byKey(const Key('craftType-select-button'));
  await tester.ensureVisible(craftType);
  await tester.pumpAndSettle();
  await tester.tap(craftType);
  await tester.pumpAndSettle();
  await tester.tap(find.text('Embroidery').last);
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(const Key('project-composer-primary-action')));
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(const Key('project-composer-primary-action')));
  await tester.pumpAndSettle();
}

Post _post() => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/parent',
  cid: 'bafy-parent',
  rkey: 'parent',
  text: 'parent',
  tags: const [],
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  createdAt: DateTime(2026),
  indexedAt: DateTime(2026),
  author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
);
