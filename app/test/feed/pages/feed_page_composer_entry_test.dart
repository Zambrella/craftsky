import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/external_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  testWidgets('IT-014 renders a full external card on the feed surface', (
    tester,
  ) async {
    Uri? launched;
    final post = Post(
      uri: 'at://did:plc:alice/social.craftsky.feed.post/external',
      cid: 'bafyexternal',
      rkey: 'external',
      text: 'Feed post',
      tags: const [],
      createdAt: DateTime.utc(2026),
      indexedAt: DateTime.utc(2026),
      author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
      likeCount: 0,
      repostCount: 0,
      replyCount: 0,
      viewerHasLiked: false,
      viewerHasReposted: false,
      viewerHasSaved: false,
      external: const PostExternal(
        uri: 'https://example.com/feed-pattern?token=final#section',
        title: 'Feed pattern',
        description: 'Feed description',
      ),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          activeLanguagePreferencesProvider.overrideWith(
            (ref) => const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
          ),
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async => TimelinePage(
                items: [TimelineItem(itemKey: 'post:${post.uri}', post: post)],
              ),
            ),
          ),
          externalCardLauncherProvider.overrideWithValue((uri) async {
            launched = uri;
            return true;
          }),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const MediaQuery(
              data: MediaQueryData(size: Size(390, 844)),
              child: FeedPage(),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(ExternalCard), findsOneWidget);
    expect(find.text('Feed pattern'), findsOneWidget);
    expect(find.text('Feed description'), findsOneWidget);
    expect(find.text('example.com'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.tap(find.byType(ExternalCard));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open link'));
    await tester.pumpAndSettle();
    expect(
      launched.toString(),
      'https://example.com/feed-pattern?token=final#section',
    );
  });

  testWidgets('IT-001 feed New post opens chooser and project branch', (
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
          authSessionProvider.overrideWith(SignedInAuthSession.new),
          postRepositoryProvider.overrideWithValue(
            FakePostRepository(
              onListTimeline: ({cursor, limit}) async =>
                  const TimelinePage(items: []),
            ),
          ),
        ],
        child: MessengerScope(
          messenger: RecordingMessenger(),
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const MediaQuery(
              data: MediaQueryData(size: Size(390, 844)),
              child: FeedPage(),
            ),
          ),
        ),
      ),
    );

    await tester.pumpAndSettle();
    await tester.tap(find.text('New post'));
    await tester.pumpAndSettle();

    expect(find.text('Regular post'), findsOneWidget);
    expect(find.text('Project post'), findsOneWidget);

    await tester.tap(find.text('Project post'));
    await tester.pumpAndSettle();

    expect(find.text('Project post'), findsOneWidget);
    expect(find.byKey(const Key('craftType-select-button')), findsOneWidget);
  });
}
