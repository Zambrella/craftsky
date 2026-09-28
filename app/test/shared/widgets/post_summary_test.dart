import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/external_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('UT-008 adapts bounded visible and policy summaries', (
    tester,
  ) async {
    final post = _post(sponsored: true);
    final data = PostSummaryData.fromPost(post);
    expect(data.state, PostSummaryState.visible);
    expect(data.text, post.text);
    expect(data.projectTitle, 'Hitchhiker shawl');
    expect(data.image, same(post.images!.first));
    expect(data.externalImport, same(post.externalImport));
    expect(data.sponsored, isFalse);
    expect(data.image, isNot(same(post.images!.last)));
    expect(data.copyWith(text: null).text, isNull);
    expect(data.copyWith(), data);
    expect(data.toString(), isNot(contains(post.text)));

    final quote = QuoteView(
      state: 'visible',
      post: QuotePreviewPost(
        uri: post.uri.toString(),
        cid: post.cid.toString(),
        text: post.text,
        author: post.author,
        createdAt: post.createdAt,
        sponsored: true,
        images: post.images,
        project: post.project,
        externalImport: post.externalImport,
      ),
    );
    expect(PostSummaryData.fromQuoteView(quote).text, post.text);
    expect(PostSummaryData.fromQuoteView(quote).sponsored, isTrue);
    expect(
      PostSummaryData.fromQuoteView(quote).externalImport,
      same(post.externalImport),
    );
    expect(
      PostSummaryData.fromQuoteView(
        const QuoteView(state: 'muted', revealable: true),
      ).state,
      PostSummaryState.muted,
    );
    expect(
      PostSummaryData.fromQuoteView(const QuoteView(state: 'blocked')).state,
      PostSummaryState.blocked,
    );

    var postTaps = 0;
    var authorTaps = 0;
    final fakeCache = FakeBaseCacheManager();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          feedImageCacheManagerProvider.overrideWithValue(fakeCache),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: PostSummary(
              data: data,
              onTap: () => postTaps++,
              onAuthorTap: () => authorTaps++,
            ),
          ),
        ),
      ),
    );
    expect(find.text('Hitchhiker shawl'), findsOneWidget);
    expect(find.text('Imported from Instagram'), findsOneWidget);
    expect(find.bySemanticsLabel('Sponsored'), findsNothing);
    expect(find.text(post.text), findsOneWidget);
    final postTime = find.byType(RelativeTimeText);
    expect(postTime, findsOneWidget);
    expect(tester.widget<RelativeTimeText>(postTime).timestamp, post.createdAt);
    expect(
      tester.getTopRight(postTime).dx,
      greaterThan(tester.getTopRight(find.text('@alice.craftsky.social')).dx),
    );
    expect(
      tester.getTopLeft(postTime).dy,
      lessThan(tester.getTopLeft(find.text(post.text)).dy),
    );
    expect(find.byIcon(CraftskyIcons.saved), findsNothing);
    expect(find.byIcon(CraftskyIcons.like), findsNothing);
    expect(find.byKey(const Key('post-summary-image')), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) => widget is Semantics && widget.properties.label == 'First',
      ),
      findsOneWidget,
    );
    final image = tester.widget<CachedNetworkImage>(
      find.byType(CachedNetworkImage),
    );
    expect(image.imageUrl, 'https://cdn.example.com/first-thumb.jpg');
    expect(image.cacheManager, same(fakeCache));
    final avatar = tester.widget<ProfileAvatar>(find.byType(ProfileAvatar));
    expect(avatar.customisation.colour, 'lime');
    await tester.tap(find.text(post.text));
    await tester.tap(find.text('@alice.craftsky.social'));
    expect((postTaps, authorTaps), (1, 1));
  });

  testWidgets('IT-011 compact quote carries external under images-win', (
    tester,
  ) async {
    Uri? launched;
    const external = PostExternal(
      uri: 'https://example.com/pattern?token=final#section',
      title: 'Quoted pattern',
      description: 'Description',
    );
    final quote = QuoteView(
      state: 'visible',
      post: QuotePreviewPost(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/quote',
        cid: 'bafyquote',
        text: 'Quoted text',
        author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
        createdAt: DateTime.utc(2026),
        sponsored: false,
        external: external,
      ),
    );
    final data = PostSummaryData.fromQuoteView(quote);
    expect(data.external, same(external));

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          externalCardLauncherProvider.overrideWithValue((uri) async {
            launched = uri;
            return true;
          }),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: PostSummary(data: data)),
        ),
      ),
    );

    expect(find.byType(ExternalCard), findsOneWidget);
    expect(find.text('Quoted pattern'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.tap(find.byType(ExternalCard));
    await tester.pump();
    await tester.tap(find.text('Open link'));
    await tester.pump();
    expect(
      launched.toString(),
      'https://example.com/pattern?token=final#section',
    );

    final imagesWin = PostSummaryData.fromQuoteView(
      QuoteView(
        state: 'visible',
        post: QuotePreviewPost(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/images',
          cid: 'bafyimages',
          text: 'Images',
          author: PostAuthor(did: 'did:plc:alice', handle: 'alice.test'),
          createdAt: DateTime.utc(2026),
          sponsored: false,
          images: [
            PostImage(cid: 'bafyimage', mime: 'image/jpeg', size: 1, alt: ''),
          ],
          external: external,
        ),
      ),
    );
    expect(imagesWin.external, isNull);

    for (final state in ['hidden', 'unavailable']) {
      final hidden = PostSummaryData.fromQuoteView(QuoteView(state: state));
      expect(hidden.state, isNot(PostSummaryState.visible));
      expect(hidden.external, isNull);
    }
  });

  testWidgets('renders policy placeholders and reveals muted summaries', (
    tester,
  ) async {
    var reveals = 0;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: Column(
            children: [
              PostSummary(
                data: const PostSummaryData(
                  state: PostSummaryState.muted,
                  revealable: true,
                ),
                onReveal: () => reveals++,
              ),
              const PostSummary(
                data: PostSummaryData(state: PostSummaryState.hidden),
              ),
              const PostSummary(
                data: PostSummaryData(state: PostSummaryState.blocked),
              ),
              const PostSummary(
                data: PostSummaryData(state: PostSummaryState.unavailable),
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.text('Post from a muted account'), findsOneWidget);
    expect(find.text('Quoted post hidden'), findsOneWidget);
    expect(find.text('Post unavailable'), findsOneWidget);
    expect(find.text('Quoted post unavailable'), findsOneWidget);
    await tester.tap(find.text('Show post'));
    expect(reveals, 1);
  });

  testWidgets('owns quote provenance and sponsored presentation', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    final quote = QuoteView(
      state: 'visible',
      post: QuotePreviewPost(
        uri: 'at://did:plc:bob/social.craftsky.feed.post/target',
        cid: 'bafyquote',
        text: 'Original imported post',
        author: PostAuthor(
          did: 'did:plc:bob',
          handle: 'bob.craftsky.social',
        ),
        createdAt: DateTime.utc(2026),
        sponsored: true,
        externalImport: const ExternalImport(source: 'instagram'),
      ),
    );
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: PostSummary(data: PostSummaryData.fromQuoteView(quote)),
        ),
      ),
    );

    expect(find.bySemanticsLabel('Imported from Instagram'), findsOneWidget);
    expect(find.bySemanticsLabel('Sponsored'), findsOneWidget);

    for (final provenance in <ExternalImport?>[
      null,
      const ExternalImport(source: 'future-service'),
    ]) {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: PostSummary(
              data: PostSummaryData.fromQuoteView(
                QuoteView(
                  state: 'visible',
                  post: quote.post!.copyWith(
                    sponsored: false,
                    externalImport: provenance,
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      expect(find.text('Imported from Instagram'), findsNothing);
    }
    semantics.dispose();
  });
}

Post _post({bool sponsored = false}) => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/summary',
  cid: 'bafysummary',
  rkey: 'summary',
  text: 'A long compact summary.',
  tags: const [],
  createdAt: DateTime.utc(2026, 7, 21),
  indexedAt: DateTime.utc(2026, 7, 21),
  author: PostAuthor(
    did: 'did:plc:alice',
    handle: 'alice.craftsky.social',
    customisation: const ProfileCustomisation(colour: 'lime'),
  ),
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: true,
  sponsored: sponsored,
  externalImport: const ExternalImport(source: 'instagram'),
  images: [
    PostImage(
      cid: 'bafyimage1',
      mime: 'image/jpeg',
      size: 1,
      alt: 'First',
      thumb: 'https://cdn.example.com/first-thumb.jpg',
    ),
    PostImage(
      cid: 'bafyimage2',
      mime: 'image/jpeg',
      size: 1,
      alt: 'Second',
      thumb: 'https://cdn.example.com/second-thumb.jpg',
    ),
  ],
  project: const Project(
    common: ProjectCommon(craftType: 'knitting', title: 'Hitchhiker shawl'),
  ),
);
