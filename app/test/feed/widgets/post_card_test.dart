import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/models/moderation_metadata.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_action_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

Post _post({
  String text = 'Cast on for the Hitchhiker shawl tonight.',
  List<Map<String, dynamic>>? facets,
  String? displayName,
  int likeCount = 0,
  int repostCount = 0,
  int quoteCount = 0,
  int replyCount = 0,
  bool viewerHasLiked = false,
  bool viewerHasReposted = false,
  bool viewerHasReplied = false,
  List<PostImage>? images,
  DateTime? createdAt,
  ModerationMetadata? moderation,
  Project? project,
  PostReply? reply,
  QuoteView? quoteView,
}) {
  return Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
    cid: 'bafy123',
    rkey: '3lf2abc',
    text: text,
    facets: facets,
    tags: const [],
    likeCount: likeCount,
    repostCount: repostCount,
    quoteCount: quoteCount,
    replyCount: replyCount,
    viewerHasLiked: viewerHasLiked,
    viewerHasReposted: viewerHasReposted,
    viewerHasReplied: viewerHasReplied,
    reply: reply,
    images: images,
    createdAt: createdAt ?? DateTime.now().subtract(const Duration(minutes: 3)),
    indexedAt: DateTime.now().subtract(const Duration(minutes: 2)),
    author: PostAuthor(
      did: 'did:plc:alice',
      handle: 'alice.craftsky.social',
      displayName: displayName,
    ),
    moderation: moderation,
    project: project,
    quoteView: quoteView,
  );
}

Future<void> _pump(
  WidgetTester tester,
  Widget child, {
  EdgeInsets viewPadding = EdgeInsets.zero,
  List<dynamic> overrides = const [],
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: List.from(overrides),
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, routeChild) {
          final mediaQuery = MediaQuery.of(context);
          return MediaQuery(
            data: mediaQuery.copyWith(viewPadding: viewPadding),
            child: routeChild!,
          );
        },
        home: Scaffold(body: child),
      ),
    ),
  );
}

void main() {
  group('PostCard', () {
    testWidgets('renders author, handle, body, and relative time', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post(displayName: 'Alice')));

      expect(find.text('Alice'), findsOneWidget);
      expect(find.textContaining('@alice.craftsky.social'), findsOneWidget);
      expect(
        find.text('Cast on for the Hitchhiker shawl tonight.'),
        findsOneWidget,
      );
      expect(find.textContaining('3m'), findsOneWidget);
      expect(find.byIcon(Icons.more_horiz), findsOneWidget);
      expect(find.byIcon(Icons.favorite_border), findsOneWidget);
      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.text('Reply'), findsNothing);
      expect(find.byIcon(Icons.repeat), findsOneWidget);
      expect(find.text('0'), findsNothing);

      final replyIcon = tester.widget<Icon>(
        find.byIcon(Icons.chat_bubble_outline),
      );
      final likeIcon = tester.widget<Icon>(find.byIcon(Icons.favorite_border));
      final repostIcon = tester.widget<Icon>(find.byIcon(Icons.repeat));
      expect(replyIcon.color, BrandColors.ink2);
      expect(likeIcon.color, BrandColors.ink2);
      expect(repostIcon.color, BrandColors.ink2);
    });

    testWidgets('AT-005 styles valid post facets with theme primary color', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            text: 'Hi @alice #Lace',
            facets: [
              _facet(3, 9, {
                r'$type': 'app.bsky.richtext.facet#mention',
                'did': 'did:plc:alice',
              }),
              _facet(10, 15, {
                r'$type': 'app.bsky.richtext.facet#tag',
                'tag': 'Lace',
              }),
            ],
          ),
        ),
      );

      final body = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text &&
              widget.textSpan?.toPlainText() == 'Hi @alice #Lace',
        ),
      );
      final spans = _leafTextSpans(body.textSpan! as TextSpan);

      expect(spans.map((span) => span.text), ['Hi ', '@alice', ' ', '#Lace']);
      expect(spans[1].style?.color, BrandColors.cobalt);
      expect(spans[3].style?.color, BrandColors.cobalt);
    });

    testWidgets('renders post body links with the shared display label', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            text: 'See https://example.com/patterns/top?utm_source=feed',
            facets: [
              _facet(4, 52, {
                r'$type': 'app.bsky.richtext.facet#link',
                'uri': 'https://example.com/patterns/top?utm_source=feed',
              }),
            ],
          ),
        ),
      );

      final body = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text &&
              widget.textSpan?.toPlainText() == 'See example.com/patterns/top',
        ),
      );
      final spans = _leafTextSpans(body.textSpan! as TextSpan);

      expect(spans.map((span) => span.text), [
        'See ',
        'example.com/patterns/top',
      ]);
      expect(spans[1].style?.color, BrandColors.cobalt);
    });

    testWidgets('post body links confirm before opening', (tester) async {
      Uri? launched;
      await _pump(
        tester,
        PostCard(
          post: _post(
            text: 'https://example.com/patterns/top?utm_source=feed',
            facets: [
              _facet(0, 48, {
                r'$type': 'app.bsky.richtext.facet#link',
                'uri': 'https://example.com/patterns/top?utm_source=feed',
              }),
            ],
          ),
        ),
        overrides: [
          facetUrlLauncherProvider.overrideWithValue((uri) async {
            launched = uri;
            return true;
          }),
        ],
      );

      await tester.tap(find.text('example.com/patterns/top'));
      await tester.pumpAndSettle();

      expect(find.text('Open link?'), findsOneWidget);
      expect(
        find.text('https://example.com/patterns/top?utm_source=feed'),
        findsOneWidget,
      );
      expect(launched, isNull);

      await tester.tap(find.text('Open link'));
      await tester.pumpAndSettle();

      expect(
        launched,
        Uri.parse('https://example.com/patterns/top?utm_source=feed'),
      );
    });

    testWidgets('renders engagement counts and selected colours', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            likeCount: 5,
            repostCount: 2,
            replyCount: 3,
            viewerHasLiked: true,
            viewerHasReposted: true,
            viewerHasReplied: true,
          ),
        ),
      );

      expect(find.byIcon(Icons.favorite), findsOneWidget);
      expect(find.text('5'), findsOneWidget);
      expect(find.text('2'), findsOneWidget);
      expect(find.text('3'), findsOneWidget);

      final replyIcon = tester.widget<Icon>(
        find.byIcon(Icons.chat_bubble_outline),
      );
      final likeIcon = tester.widget<Icon>(find.byIcon(Icons.favorite));
      final repostIcon = tester.widget<Icon>(find.byIcon(Icons.repeat));
      final replyCount = tester.widget<Text>(find.text('3'));
      final likeCount = tester.widget<Text>(find.text('5'));
      final repostCount = tester.widget<Text>(find.text('2'));

      expect(replyIcon.color, BrandColors.clay);
      expect(likeIcon.color, BrandColors.red);
      expect(repostIcon.color, BrandColors.moss);
      expect(replyCount.style?.color, BrandColors.clay);
      expect(likeCount.style?.color, BrandColors.red);
      expect(repostCount.style?.color, BrandColors.moss);
    });

    testWidgets('renders combined share count from reposts and quotes', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(post: _post(repostCount: 2, quoteCount: 3)),
      );

      expect(find.text('5'), findsOneWidget);
      expect(find.text('2'), findsNothing);
      expect(find.text('3'), findsNothing);
    });

    testWidgets('renders tappable repost attribution inside the card', (
      tester,
    ) async {
      var reposterTaps = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(),
          repostReason: RepostReason(
            type: RepostReasonType.repost,
            by: PostAuthor(
              did: 'did:plc:bob',
              handle: 'bob.craftsky.social',
              displayName: 'Bob',
            ),
            uri: 'at://did:plc:bob/social.craftsky.feed.repost/repost-target',
            createdAt: DateTime(2026, 5, 22, 13),
            indexedAt: DateTime(2026, 5, 22, 13),
          ),
          onReposterTap: () => reposterTaps++,
        ),
      );

      final attribution = find.text('Reposted by Bob');
      expect(attribution, findsOneWidget);
      expect(
        find.ancestor(of: attribution, matching: find.byType(CraftskyCard)),
        findsOneWidget,
      );

      await tester.tap(attribution);
      expect(reposterTaps, 1);
    });

    testWidgets('renders visible quote preview', (tester) async {
      var quotedPostTaps = 0;
      var quotedAuthorTaps = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(
            text: 'My take on this pattern.',
            quoteView: QuoteView(
              state: 'visible',
              post: QuotePreviewPost(
                uri: 'at://did:plc:bob/social.craftsky.feed.post/target',
                cid: 'bafyquote',
                text: 'Original quoted post',
                author: PostAuthor(
                  did: 'did:plc:bob',
                  handle: 'bob.craftsky.social',
                  displayName: 'Bob',
                  avatar: 'https://cdn.example.com/bob.jpg',
                ),
                createdAt: DateTime(2026, 5, 22, 12),
              ),
            ),
          ),
          onQuotedPostTap: () => quotedPostTaps++,
          onQuotedAuthorTap: () => quotedAuthorTaps++,
        ),
      );

      expect(find.text('My take on this pattern.'), findsOneWidget);
      expect(find.text('Original quoted post'), findsOneWidget);
      expect(find.text('Bob'), findsOneWidget);
      expect(find.text('@bob.craftsky.social'), findsOneWidget);
      expect(find.byType(ProfileAvatar), findsNWidgets(2));
      expect(
        tester.widget<ProfileAvatar>(find.byType(ProfileAvatar).last).avatarUrl,
        'https://cdn.example.com/bob.jpg',
      );

      await tester.tap(find.text('Bob'));
      expect(quotedAuthorTaps, 1);
      expect(quotedPostTaps, 0);

      await tester.tap(find.text('Original quoted post'));
      expect(quotedPostTaps, 1);
    });

    testWidgets('renders only the first image from a quoted normal post', (
      tester,
    ) async {
      final fakeCache = FakeBaseCacheManager();
      await _pump(
        tester,
        PostCard(
          post: _post(
            quoteView: QuoteView(
              state: 'visible',
              post: QuotePreviewPost(
                uri: 'at://did:plc:bob/social.craftsky.feed.post/target',
                cid: 'bafyquote',
                text: 'Original quoted post',
                author: PostAuthor(
                  did: 'did:plc:bob',
                  handle: 'bob.craftsky.social',
                ),
                images: [
                  PostImage(
                    cid: 'bafkfirst',
                    mime: 'image/jpeg',
                    size: 10,
                    alt: 'First quoted image',
                    thumb: 'https://cdn.example.com/first-thumb.jpg',
                    fullsize: 'https://cdn.example.com/first-full.jpg',
                  ),
                  PostImage(
                    cid: 'bafksecond',
                    mime: 'image/jpeg',
                    size: 10,
                    alt: 'Second quoted image',
                    thumb: 'https://cdn.example.com/second-thumb.jpg',
                  ),
                ],
                createdAt: DateTime(2026, 5, 22, 12),
              ),
            ),
          ),
        ),
        overrides: [
          feedImageCacheManagerProvider.overrideWith((ref) => fakeCache),
        ],
      );
      await tester.pump();

      expect(find.byKey(const Key('quote-preview-image')), findsOneWidget);
      final image = tester.widget<CachedNetworkImage>(
        find.byType(CachedNetworkImage),
      );
      expect(image.imageUrl, 'https://cdn.example.com/first-thumb.jpg');
      expect(image.cacheManager, same(fakeCache));
      expect(
        find.byWidgetPredicate(
          (widget) =>
              widget is Semantics &&
              widget.properties.label == 'First quoted image',
        ),
        findsOneWidget,
      );
      expect(
        find.byWidgetPredicate(
          (widget) =>
              widget is Semantics &&
              widget.properties.label == 'Second quoted image',
        ),
        findsNothing,
      );
    });

    testWidgets(
      'renders the first image and project name for a quoted project',
      (
        tester,
      ) async {
        final fakeCache = FakeBaseCacheManager();
        await _pump(
          tester,
          PostCard(
            post: _post(
              quoteView: QuoteView(
                state: 'visible',
                post: QuotePreviewPost(
                  uri: 'at://did:plc:bob/social.craftsky.feed.post/project',
                  cid: 'bafyproject',
                  text: 'A finished project.',
                  author: PostAuthor(
                    did: 'did:plc:bob',
                    handle: 'bob.craftsky.social',
                  ),
                  images: [
                    PostImage(
                      cid: 'bafkproject',
                      mime: 'image/jpeg',
                      size: 10,
                      alt: 'Finished blue shawl',
                      thumb: 'https://cdn.example.com/project-thumb.jpg',
                    ),
                  ],
                  project: const Project(
                    common: ProjectCommon(
                      craftType: ProjectOptionCatalogs.knittingCraftToken,
                      title: 'Hitchhiker Shawl',
                    ),
                  ),
                  createdAt: DateTime(2026, 5, 22, 12),
                ),
              ),
            ),
          ),
          overrides: [
            feedImageCacheManagerProvider.overrideWith((ref) => fakeCache),
          ],
        );
        await tester.pump();

        expect(find.byKey(const Key('quote-preview-image')), findsOneWidget);
        expect(find.text('Hitchhiker Shawl'), findsOneWidget);
        expect(
          find.byWidgetPredicate(
            (widget) =>
                widget is Semantics &&
                widget.properties.label == 'Finished blue shawl',
          ),
          findsOneWidget,
        );
      },
    );

    testWidgets('renders quote preview placeholders', (tester) async {
      await _pump(
        tester,
        Column(
          children: [
            PostCard(
              post: _post(quoteView: const QuoteView(state: 'hidden')),
            ),
            PostCard(
              post: _post(quoteView: const QuoteView(state: 'unavailable')),
            ),
          ],
        ),
      );

      expect(find.text('Quoted post hidden'), findsOneWidget);
      expect(find.text('Quoted post unavailable'), findsOneWidget);
    });

    testWidgets('colours reply label when viewer has replied', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(viewerHasReplied: true),
          showReplyCount: false,
          showReplyLabel: true,
        ),
      );

      final replyIcon = tester.widget<Icon>(
        find.byIcon(Icons.chat_bubble_outline),
      );
      final replyLabel = tester.widget<Text>(find.text('Reply'));

      expect(replyIcon.color, BrandColors.clay);
      expect(replyLabel.style?.color, BrandColors.clay);
    });

    testWidgets('formats large engagement counts compactly', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(likeCount: 1234, repostCount: 15000, replyCount: 2000000),
        ),
      );

      expect(find.text('1.2k'), findsOneWidget);
      expect(find.text('15k'), findsOneWidget);
      expect(find.text('2m'), findsOneWidget);
      expect(find.text('1234'), findsNothing);
    });

    testWidgets('can hide reply count while keeping reply action', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(replyCount: 3),
          showReplyCount: false,
          showReplyLabel: true,
        ),
      );

      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.text('3'), findsNothing);
      expect(find.text('Reply'), findsOneWidget);
    });

    testWidgets('can hide reply label while keeping reply count', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(post: _post(replyCount: 3)),
      );

      expect(find.byIcon(Icons.chat_bubble_outline), findsOneWidget);
      expect(find.text('3'), findsOneWidget);
      expect(find.text('Reply'), findsNothing);
    });

    testWidgets('flat style does not use the card surface', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(),
          style: PostCardStyle.flat,
          showReplyLabel: true,
        ),
      );

      expect(find.byType(CraftskyCard), findsNothing);
      expect(find.text('Reply'), findsOneWidget);
    });

    testWidgets('renders project summary before body text', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            text: 'Process shots, swipe through.',
            images: [
              PostImage(
                cid: 'bafkprojectimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Indigo jacket on a hanger',
                aspectRatio: const PostImageAspectRatio(width: 4, height: 1),
                thumb: 'https://cdn.example.com/project-thumb.jpg',
                fullsize: 'https://cdn.example.com/project-full.jpg',
              ),
            ],
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.sewingCraftToken,
                status: ProjectOptionCatalogs.finishedStatusToken,
                title: 'Wiksten Haori in indigo linen',
                pattern: ProjectPattern(
                  name: 'Wiksten Haori',
                  designer: 'Jenny Gordy',
                ),
              ),
              details: SewingProjectDetails(sizeMade: 'Medium'),
            ),
          ),
        ),
      );

      expect(find.text('Wiksten Haori in indigo linen'), findsOneWidget);
      expect(find.text('Finished'), findsOneWidget);
      expect(find.text('Sewing'), findsOneWidget);
      expect(find.text('PATTERN'), findsOneWidget);
      expect(find.text('Wiksten Haori'), findsOneWidget);
      expect(find.text('Jenny Gordy'), findsOneWidget);
      expect(find.text('SIZE'), findsOneWidget);
      expect(find.text('Medium'), findsOneWidget);
      expect(find.text('Process shots, swipe through.'), findsOneWidget);

      final title = tester.widget<Text>(
        find.text('Wiksten Haori in indigo linen'),
      );
      final theme = Theme.of(tester.element(find.byType(PostCard)));
      expect(title.style?.fontFamily, theme.textTheme.displaySmall?.fontFamily);
      expect(title.style?.fontSize, theme.textTheme.headlineSmall?.fontSize);

      expect(
        tester.getCenter(find.text('PATTERN')).dy,
        moreOrLessEquals(
          tester.getCenter(find.text('Wiksten Haori')).dy,
          epsilon: 1,
        ),
      );
      expect(
        tester.getCenter(find.text('SIZE')).dy,
        moreOrLessEquals(tester.getCenter(find.text('Medium')).dy, epsilon: 1),
      );

      expect(
        tester.getBottomLeft(find.byKey(const Key('post-image-carousel'))).dy,
        lessThan(
          tester.getTopLeft(find.text('Wiksten Haori in indigo linen')).dy,
        ),
      );
      expect(
        tester.getTopLeft(find.text('Wiksten Haori in indigo linen')).dy,
        lessThan(
          tester.getTopLeft(find.text('Process shots, swipe through.')).dy,
        ),
      );
    });

    testWidgets('renders clickable facets in project pattern metadata', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            project: Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.knittingCraftToken,
                pattern: ProjectPattern(
                  name: '#hitchhiker',
                  nameFacets: [
                    _facet(0, 11, {
                      r'$type': 'app.bsky.richtext.facet#tag',
                      'tag': 'hitchhiker',
                    }),
                  ],
                  designer: '@alice.craftsky.social',
                  designerFacets: [
                    _facet(0, 22, {
                      r'$type': 'app.bsky.richtext.facet#mention',
                      'did': 'did:plc:alice',
                    }),
                  ],
                ),
              ),
            ),
          ),
        ),
      );

      final patternName = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text && widget.textSpan?.toPlainText() == '#hitchhiker',
        ),
      );
      final designer = tester.widget<Text>(
        find.byWidgetPredicate(
          (widget) =>
              widget is Text &&
              widget.textSpan?.toPlainText() == '@alice.craftsky.social',
        ),
      );
      final patternNameSpan = _leafTextSpans(
        patternName.textSpan! as TextSpan,
      ).single;
      final designerSpan = _leafTextSpans(
        designer.textSpan! as TextSpan,
      ).single;

      expect(patternNameSpan.style?.color, BrandColors.cobalt);
      expect(patternNameSpan.recognizer, isNotNull);
      expect(designerSpan.style?.color, BrandColors.cobalt);
      expect(designerSpan.recognizer, isNotNull);
    });

    testWidgets('renders partial pattern metadata without empty rows', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.knittingCraftToken,
                pattern: ProjectPattern(name: 'Hitchhiker'),
              ),
            ),
          ),
        ),
      );

      expect(find.text('PATTERN'), findsOneWidget);
      expect(find.text('Hitchhiker'), findsOneWidget);
      expect(find.textContaining(' by '), findsNothing);

      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.knittingCraftToken,
                pattern: ProjectPattern(designer: 'Martina Behm'),
              ),
            ),
          ),
        ),
      );

      expect(find.text('PATTERN'), findsOneWidget);
      expect(find.text('Martina Behm'), findsOneWidget);

      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.knittingCraftToken,
                pattern: ProjectPattern(),
              ),
            ),
          ),
        ),
      );

      expect(find.text('PATTERN'), findsNothing);
    });

    testWidgets('renders craft-specific size metadata', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.knittingCraftToken,
              ),
              details: KnittingProjectDetails(finishedSize: '40 in bust'),
            ),
          ),
        ),
      );
      expect(find.text('FINISHED SIZE'), findsOneWidget);
      expect(find.text('40 in bust'), findsOneWidget);

      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.crochetCraftToken,
              ),
              details: CrochetProjectDetails(finishedSize: 'Baby blanket'),
            ),
          ),
        ),
      );
      expect(find.text('FINISHED SIZE'), findsOneWidget);
      expect(find.text('Baby blanket'), findsOneWidget);

      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.quiltingCraftToken,
              ),
              details: QuiltingProjectDetails(size: '60 x 72 in'),
            ),
          ),
        ),
      );
      expect(find.text('SIZE'), findsOneWidget);
      expect(find.text('60 x 72 in'), findsOneWidget);

      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: ProjectOptionCatalogs.embroideryCraftToken,
              ),
            ),
          ),
        ),
      );
      expect(find.text('SIZE'), findsNothing);
      expect(find.text('FINISHED SIZE'), findsNothing);
    });

    testWidgets('falls back to readable labels for unknown project tokens', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            project: const Project(
              common: ProjectCommon(
                craftType: 'social.craftsky.feed.defs#machineKnitting',
                status: 'social.craftsky.feed.defs#blockedOut',
              ),
            ),
          ),
        ),
      );

      expect(find.text('Machine Knitting'), findsOneWidget);
      expect(find.text('Blocked Out'), findsOneWidget);
      expect(find.textContaining('social.craftsky'), findsNothing);
    });

    testWidgets('uses stronger treatment when highlighted', (tester) async {
      await _pump(tester, PostCard(post: _post(), isHighlighted: true));

      final highlighted = tester.widget<AnimatedContainer>(
        find.byWidgetPredicate(
          (widget) =>
              widget is AnimatedContainer && widget.decoration is BoxDecoration,
        ),
      );
      final decoration = highlighted.decoration! as BoxDecoration;
      final border = decoration.border! as Border;

      expect(decoration.color, BrandColors.sky.withValues(alpha: 0.32));
      expect(border.left.color, BrandColors.cobalt);
      expect(border.left.width, 6);
    });

    testWidgets('shows full timestamp tooltip on relative time', (
      tester,
    ) async {
      final createdAt = DateTime(2026, 5, 15, 14, 30);
      await _pump(tester, PostCard(post: _post(createdAt: createdAt)));

      final tooltip = tester.widget<Tooltip>(find.byType(Tooltip).first);
      expect(tooltip.message, contains('2026'));
      expect(tooltip.message, contains('2:30'));
    });

    testWidgets('lays out interaction actions left-aligned at natural widths', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(post: _post(likeCount: 5, repostCount: 2, replyCount: 3)),
      );

      expect(find.byType(TextButton), findsNWidgets(3));
      final likeRect = tester.getRect(find.byType(TextButton).at(0));
      final replyRect = tester.getRect(find.byType(TextButton).at(1));
      final repostRect = tester.getRect(find.byType(TextButton).at(2));

      expect(likeRect.left, lessThan(replyRect.left));
      expect(replyRect.left, lessThan(repostRect.left));
      expect(replyRect.width, likeRect.width);

      final replyCountRect = tester.getRect(find.text('3'));
      final likeCountRect = tester.getRect(find.text('5'));
      final repostCountRect = tester.getRect(find.text('2'));

      expect(replyCountRect.right, lessThan(replyRect.right));
      expect(likeCountRect.right, lessThan(likeRect.right));
      expect(repostCountRect.right, lessThan(repostRect.right));

      final replyCount = tester.widget<Text>(find.text('3'));
      expect(
        replyCount.style?.fontFeatures,
        contains(const FontFeature.tabularFigures()),
      );
    });

    testWidgets('opens share menu with repost and quote choices', (
      tester,
    ) async {
      var reposts = 0;
      var quotes = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(),
          onRepost: () => reposts++,
          onQuote: () => quotes++,
        ),
      );

      await tester.tap(find.byIcon(Icons.repeat));
      await tester.pumpAndSettle();

      expect(find.text('Repost'), findsOneWidget);
      expect(find.text('Quote'), findsOneWidget);

      await tester.tap(find.text('Quote'));
      await tester.pumpAndSettle();

      expect(reposts, 0);
      expect(quotes, 1);
    });

    testWidgets('share count opens the same menu', (tester) async {
      await _pump(
        tester,
        PostCard(post: _post(repostCount: 2), onRepost: () {}, onQuote: () {}),
      );

      await tester.tap(find.text('2'));
      await tester.pumpAndSettle();

      expect(find.text('Repost'), findsOneWidget);
      expect(find.text('Quote'), findsOneWidget);
    });

    testWidgets('hides share action for reply posts', (tester) async {
      final root = PostRef(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
        cid: 'bafyroot',
      );
      final parent = PostRef(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/parent',
        cid: 'bafyparent',
      );

      await _pump(
        tester,
        PostCard(
          post: _post(
            reply: PostReply(root: root, parent: parent),
          ),
          onRepost: () {},
          onQuote: () {},
        ),
      );

      expect(find.byIcon(Icons.repeat), findsNothing);
    });

    testWidgets('invokes interaction callbacks', (tester) async {
      var replies = 0;
      var likes = 0;
      var reposts = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(),
          onReply: () => replies++,
          onLike: () => likes++,
          onRepost: () => reposts++,
        ),
      );

      await tester.tap(find.byIcon(Icons.favorite_border));
      await tester.tap(find.byIcon(Icons.chat_bubble_outline));
      await tester.tap(find.byIcon(Icons.repeat));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Repost'));

      expect(replies, 1);
      expect(likes, 1);
      expect(reposts, 1);
    });

    testWidgets('invokes card tap from body', (tester) async {
      var taps = 0;
      await _pump(tester, PostCard(post: _post(), onTap: () => taps++));

      await tester.tap(find.text('Cast on for the Hitchhiker shawl tonight.'));

      expect(taps, 1);
    });

    testWidgets('reply tap does not invoke card tap', (tester) async {
      var replies = 0;
      var taps = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(),
          onTap: () => taps++,
          onReply: () => replies++,
        ),
      );

      await tester.tap(find.byIcon(Icons.chat_bubble_outline));

      expect(replies, 1);
      expect(taps, 0);
    });

    testWidgets('falls back to handle when display name is absent', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post()));

      expect(find.text('alice.craftsky.social'), findsOneWidget);
    });

    testWidgets('opens empty menu when delete callback is absent', (
      tester,
    ) async {
      await _pump(tester, PostCard(post: _post()));

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.text('Delete post'), findsNothing);
    });

    testWidgets('shows delete action when callback is supplied', (
      tester,
    ) async {
      var tapped = false;
      await _pump(
        tester,
        PostCard(post: _post(), onDelete: () => tapped = true),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete post'));
      expect(tapped, isTrue);
    });

    testWidgets('shows report action only when callback is supplied', (
      tester,
    ) async {
      var reports = 0;
      await _pump(
        tester,
        PostCard(post: _post(), onReport: () => reports++),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Report post'));

      expect(reports, 1);

      await _pump(tester, PostCard(post: _post()));
      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.text('Report post'), findsNothing);
    });

    testWidgets('renders generic warning copy without raw reason text', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            moderation: const ModerationMetadata(warningKind: 'author'),
          ),
        ),
      );

      expect(
        find.text('This author may not follow CraftSky community guidelines.'),
        findsOneWidget,
      );
      expect(find.textContaining('raw unsafe reason fixture'), findsNothing);
    });

    testWidgets('uses custom delete label when supplied', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(),
          deleteLabel: 'Delete comment',
          onDelete: () {},
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.text('Delete comment'), findsOneWidget);
      expect(find.text('Delete post'), findsNothing);
    });

    testWidgets('renders single post image without multi-image indicators', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Blue shawl drying flat',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
            ],
          ),
        ),
      );

      expect(find.byKey(const Key('post-image-carousel')), findsOneWidget);
      expect(find.byKey(const Key('post-image-count')), findsNothing);
      expect(find.byKey(const Key('post-image-dots')), findsNothing);
      expect(find.bySemanticsLabel('Blue shawl drying flat'), findsOneWidget);
      expect(find.byType(InteractiveViewer), findsWidgets);
    });

    testWidgets('renders multi-image indicators and count', (tester) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Image one',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
              PostImage(
                cid: 'bafkimage2',
                mime: 'image/png',
                size: 11,
                alt: 'Image two',
                thumb: 'https://cdn.example.com/thumb2.jpg',
                fullsize: 'https://cdn.example.com/full2.jpg',
              ),
            ],
          ),
        ),
      );

      expect(find.byKey(const Key('post-image-count')), findsOneWidget);
      expect(find.byKey(const Key('post-image-dots')), findsOneWidget);
      expect(find.text('1/2'), findsOneWidget);
    });

    testWidgets('horizontal paging updates image index without card tap', (
      tester,
    ) async {
      var cardTaps = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Image one',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
              PostImage(
                cid: 'bafkimage2',
                mime: 'image/png',
                size: 11,
                alt: 'Image two',
                thumb: 'https://cdn.example.com/thumb2.jpg',
                fullsize: 'https://cdn.example.com/full2.jpg',
              ),
            ],
          ),
          onTap: () => cardTaps++,
        ),
      );

      expect(find.text('1/2'), findsOneWidget);

      await tester.drag(
        find.byKey(const Key('post-image-carousel')),
        const Offset(-500, 0),
      );
      await tester.pumpAndSettle();

      expect(find.text('2/2'), findsOneWidget);
      expect(cardTaps, 0);
    });

    testWidgets(
      'tapping image opens gallery while non-image tap keeps card routing',
      (
        tester,
      ) async {
        var cardTaps = 0;
        await _pump(
          tester,
          PostCard(
            post: _post(
              images: [
                PostImage(
                  cid: 'bafkimage1',
                  mime: 'image/jpeg',
                  size: 10,
                  alt: 'Image one',
                  thumb: 'https://cdn.example.com/thumb1.jpg',
                  fullsize: 'https://cdn.example.com/full1.jpg',
                ),
                PostImage(
                  cid: 'bafkimage2',
                  mime: 'image/png',
                  size: 11,
                  alt: 'Image two',
                  thumb: 'https://cdn.example.com/thumb2.jpg',
                  fullsize: 'https://cdn.example.com/full2.jpg',
                ),
              ],
            ),
            onTap: () => cardTaps++,
          ),
        );

        await tester.tap(find.byKey(const Key('post-image-carousel')));
        await tester.pump();
        await tester.pump(const Duration(seconds: 1));

        expect(find.byType(PostImageGallery), findsOneWidget);
        expect(find.byType(AppBar), findsNothing);
        expect(find.byType(CloseButton), findsOneWidget);
        expect(
          find.byKey(const Key('post-image-gallery-close-background')),
          findsOneWidget,
        );
        expect(
          find.ancestor(
            of: find.byType(CloseButton),
            matching: find.byType(SafeArea),
          ),
          findsNothing,
        );
        expect(
          find.byKey(const Key('post-image-gallery-count')),
          findsOneWidget,
        );
        expect(
          find.byKey(const Key('post-image-gallery-dots')),
          findsOneWidget,
        );
        expect(find.text('1/2'), findsOneWidget);
        expect(cardTaps, 0);

        tester.state<NavigatorState>(find.byType(Navigator).first).pop();
        await tester.pumpAndSettle();

        await tester.tap(
          find.text('Cast on for the Hitchhiker shawl tonight.'),
        );
        expect(cardTaps, 1);
      },
    );

    testWidgets('gallery close button accounts for media view padding', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Image one',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
            ],
          ),
        ),
        viewPadding: const EdgeInsets.only(left: 11, top: 23),
      );

      await tester.tap(find.byKey(const Key('post-image-carousel')));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      final closeBackground = find.byKey(
        const Key('post-image-gallery-close-background'),
      );
      expect(tester.getTopLeft(closeBackground), const Offset(19, 31));
      final decoratedBox = tester.widget<DecoratedBox>(closeBackground);
      final decoration = decoratedBox.decoration as BoxDecoration;
      expect(decoration.shape, BoxShape.circle);
      expect(decoration.color, isNotNull);
    });

    testWidgets('opens gallery at currently visible tapped image index', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Image one',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
              PostImage(
                cid: 'bafkimage2',
                mime: 'image/png',
                size: 11,
                alt: 'Image two',
                thumb: 'https://cdn.example.com/thumb2.jpg',
                fullsize: 'https://cdn.example.com/full2.jpg',
              ),
            ],
          ),
        ),
      );

      await tester.drag(
        find.byKey(const Key('post-image-carousel')),
        const Offset(-500, 0),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('post-image-carousel')));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(PostImageGallery), findsOneWidget);
      expect(find.text('Image two'), findsOneWidget);
      expect(find.text('2/2'), findsOneWidget);
      expect(find.text('Image one'), findsNothing);
    });

    testWidgets('opens gallery without hero animations', (
      tester,
    ) async {
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Image one',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
            ],
          ),
        ),
      );

      await tester.tap(find.byKey(const Key('post-image-carousel')));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(PostImageGallery), findsOneWidget);
      expect(find.byType(Hero), findsNothing);
    });

    testWidgets('image-card action taps do not open gallery', (tester) async {
      var replies = 0;
      await _pump(
        tester,
        PostCard(
          post: _post(
            images: [
              PostImage(
                cid: 'bafkimage1',
                mime: 'image/jpeg',
                size: 10,
                alt: 'Image one',
                thumb: 'https://cdn.example.com/thumb1.jpg',
                fullsize: 'https://cdn.example.com/full1.jpg',
              ),
            ],
          ),
          onReply: () => replies++,
        ),
      );

      await tester.tap(find.byIcon(Icons.chat_bubble_outline));
      await tester.pumpAndSettle();

      expect(replies, 1);
      expect(find.byType(PostImageGallery), findsNothing);
    });
  });
}

List<TextSpan> _leafTextSpans(TextSpan root) {
  final leaves = <TextSpan>[];

  void visit(TextSpan span) {
    final children = span.children;
    if (children == null || children.isEmpty) {
      leaves.add(span);
      return;
    }
    for (final child in children) {
      if (child is TextSpan) visit(child);
    }
  }

  visit(root);
  return leaves;
}

Map<String, dynamic> _facet(
  int byteStart,
  int byteEnd,
  Map<String, dynamic> feature,
) {
  return {
    'index': {'byteStart': byteStart, 'byteEnd': byteEnd},
    'features': [feature],
  };
}
