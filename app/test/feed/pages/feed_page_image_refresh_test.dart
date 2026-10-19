import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/widgets/inline_pinch_zoom.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../../fakes/auth_session_fakes.dart';
import '../../fakes/recording_messenger.dart';
import '../fakes/fake_post_repository.dart';

void main() {
  testWidgets('pinching the first feed image cannot pull to refresh', (
    tester,
  ) async {
    var loads = 0;
    await _pumpFeed(tester, posts: [_photoPost(0)], onLoad: () => loads++);
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

  testWidgets(
    'releasing an inline pinch preserves the scrolled feed position',
    (
      tester,
    ) async {
      var loads = 0;
      await _pumpFeed(
        tester,
        posts: List.generate(12, _photoPost),
        onLoad: () => loads++,
      );
      await tester.pumpAndSettle();
      final scrollView = find.byType(CustomScrollView);
      await tester.drag(scrollView, const Offset(0, -1100));
      await tester.pumpAndSettle();
      final controller = tester
          .widget<CustomScrollView>(scrollView)
          .controller!;
      final offset = controller.offset;
      expect(offset, greaterThan(500));
      final image = find.byType(InlinePinchZoom).hitTestable().first;
      final scrim = Theme.of(
        tester.element(image),
      ).colorScheme.scrim.withValues(alpha: 0.12);
      final zoomBarrier = find.byWidgetPredicate(
        (widget) => widget is ModalBarrier && widget.color == scrim,
      );
      final imageRect = tester.getRect(image);
      final center = imageRect.intersect(tester.getRect(scrollView)).center;
      final first = await tester.startGesture(center - const Offset(25, 0));
      final second = await tester.startGesture(
        center + const Offset(25, 0),
        pointer: 7,
      );
      await first.moveBy(const Offset(-25, 0));
      await second.moveBy(const Offset(25, 0));
      await tester.pump();
      expect(zoomBarrier, findsOneWidget);
      expect(controller.offset, closeTo(offset, 0.01));
      await first.up();
      await second.up();
      await tester.pumpAndSettle();
      expect(zoomBarrier, findsNothing);
      expect(controller.offset, closeTo(offset, 0.01));
      expect(loads, 1);

      // The same feed remains scrollable after the zoom has reset.
      await tester.drag(scrollView, const Offset(0, -200));
      await tester.pumpAndSettle();
      expect(controller.offset, greaterThan(offset));
    },
  );
}

Post _photoPost(int index) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/photo$index',
  cid: 'bafyphoto',
  rkey: 'photo$index',
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

Future<void> _pumpFeed(
  WidgetTester tester, {
  required List<Post> posts,
  required VoidCallback onLoad,
}) async {
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
              onLoad();
              return TimelinePage(
                items: [
                  for (final post in posts)
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
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const MediaQuery(
            data: MediaQueryData(size: Size(390, 844)),
            child: FeedPage(),
          ),
        ),
      ),
    ),
  );
}
