import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';

void main() {
  testWidgets('AT-001 Now submits through the immediate post path', (
    tester,
  ) async {
    var createCalls = 0;
    final repository = FakePostRepository(
      onCreateWithFacets:
          ({required text, reply, project, images, facets}) async {
            createCalls += 1;
            return _post(text);
          },
    );
    final container = ProviderContainer.test(
      overrides: [
        activeLanguagePreferencesProvider.overrideWith(
          (ref) => const LanguagePreferences(
            primaryLanguage: 'en',
            contentLanguages: ['en'],
          ),
        ),
        postRepositoryProvider.overrideWithValue(repository),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const PostComposerSheet(composerId: 'post-now'),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Now'), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Post'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Post'), findsNothing);
    expect(
      find.byKey(const Key('post-composer-bottom-safe-space')),
      findsOneWidget,
    );
    final postButton = find.widgetWithText(ChunkyButton, 'Post');
    expect(
      tester
          .getSize(
            find.byKey(const Key('post-composer-bottom-safe-space')),
          )
          .height,
      greaterThan(tester.getSize(postButton).height),
    );
    expect(tester.getSize(postButton).width, greaterThan(300));
    await tester.enterText(find.byType(TextField).first, 'Post immediately');
    await _pumpUntilEnabled(tester, 'Post');
    await tester.tap(find.byKey(const Key('post-composer-primary-action')));
    await tester.pumpAndSettle();

    expect(createCalls, 1);
  });
}

Future<void> _pumpUntilEnabled(WidgetTester tester, String label) async {
  for (var attempt = 0; attempt < 100; attempt++) {
    await tester.pump(const Duration(milliseconds: 20));
    final button = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, label),
    );
    if (button.onPressed != null) return;
  }
  fail('$label did not become enabled');
}

Post _post(String text) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/now',
  cid: 'bafy-now',
  rkey: 'now',
  text: text,
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
