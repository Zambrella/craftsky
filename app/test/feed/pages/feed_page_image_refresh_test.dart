import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/inline_pinch_zoom.dart';
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
  testWidgets('pinching the first feed image cannot pull to refresh', (
    tester,
  ) async {
    var loads = 0;
    final post = Post(
      uri: 'at://did:plc:alice/social.craftsky.feed.post/photo',
      cid: 'bafyphoto',
      rkey: 'photo',
      text: 'Photo',
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
      sponsored: false,
      images: [
        PostImage(cid: 'bafyimage', mime: 'image/jpeg', size: 1, alt: 'Photo'),
      ],
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
              onListTimeline: ({cursor, limit}) async {
                loads++;
                return TimelinePage(
                  items: [
                    TimelineItem(itemKey: 'post:${post.uri}', post: post),
                  ],
                );
              },
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
    expect(loads, 1);
    final image = find.byType(InlinePinchZoom);
    expect(image, findsOneWidget);
    final center = tester.getCenter(image);
    final first = await tester.startGesture(center - const Offset(25, 0));
    await first.moveBy(const Offset(0, 160));
    await tester.pump();
    final second = await tester.startGesture(
      center + const Offset(25, 0),
      pointer: 7,
    );
    await first.moveBy(const Offset(-20, 140));
    await second.moveBy(const Offset(20, 140));
    await tester.pump();
    await first.moveBy(const Offset(0, 140));
    await second.moveBy(const Offset(0, 140));
    await tester.pump();
    await first.up();
    await second.up();
    await tester.pumpAndSettle();
    expect(loads, 1);

    // Pulling with a single finger still refreshes normally.
    await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
    await tester.pumpAndSettle();
    expect(loads, 2);
  });
}
