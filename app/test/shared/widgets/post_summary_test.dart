import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  testWidgets('UT-008 adapts bounded visible and policy summaries', (
    tester,
  ) async {
    final post = _post();
    final data = PostSummaryData.fromPost(post);
    expect(data.state, PostSummaryState.visible);
    expect(data.text, post.text);
    expect(data.projectTitle, 'Hitchhiker shawl');
    expect(data.image, same(post.images!.first));
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
        images: post.images,
        project: post.project,
      ),
    );
    expect(PostSummaryData.fromQuoteView(quote).text, post.text);
    expect(
      PostSummaryData.fromQuoteView(
        const QuoteView(state: 'muted', revealable: true),
      ).state,
      PostSummaryState.muted,
    );
    expect(
      PostSummaryData.fromQuoteView(
        const QuoteView(state: 'blocked'),
      ).state,
      PostSummaryState.blocked,
    );

    var postTaps = 0;
    var authorTaps = 0;
    await tester.pumpWidget(
      MaterialApp(
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
    );
    expect(find.text('Hitchhiker shawl'), findsOneWidget);
    expect(find.text(post.text), findsOneWidget);
    expect(find.byIcon(Icons.bookmark), findsNothing);
    expect(find.byIcon(Icons.favorite_border), findsNothing);
    await tester.tap(find.text(post.text));
    await tester.tap(find.text('@alice.craftsky.social'));
    expect((postTaps, authorTaps), (1, 1));
  });
}

Post _post() => Post(
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
  ),
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: true,
  images: [
    PostImage(cid: 'bafyimage1', mime: 'image/jpeg', size: 1, alt: 'First'),
    PostImage(cid: 'bafyimage2', mime: 'image/jpeg', size: 1, alt: 'Second'),
  ],
  project: const Project(
    common: ProjectCommon(
      craftType: 'knitting',
      title: 'Hitchhiker shawl',
    ),
  ),
);
