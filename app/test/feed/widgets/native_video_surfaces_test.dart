import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final surface in [
    'timeline card',
    'profile post list',
    'search result',
    'project list',
    'post detail thread',
  ]) {
    testWidgets('IT-016 $surface forwards canonical video once', (
      tester,
    ) async {
      final received = <PostVideo>[];
      final post = _videoPost();

      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: PostCard(
                post: post,
                videoPlayerBuilder: (video) {
                  received.add(video);
                  return const SizedBox(key: ValueKey('injected-player'));
                },
              ),
            ),
          ),
        ),
      );

      expect(find.byKey(const ValueKey('injected-player')), findsOneWidget);
      expect(received, [same(post.video)]);
    });
  }
}

Post _videoPost() => Post(
  uri: 'at://did:plc:alice/social.craftsky.feed.post/3video',
  cid: 'bafypost',
  rkey: '3video',
  text: 'Spinning wool',
  tags: const [],
  likeCount: 0,
  repostCount: 0,
  replyCount: 0,
  viewerHasLiked: false,
  viewerHasReposted: false,
  viewerHasSaved: false,
  createdAt: DateTime(2026),
  indexedAt: DateTime(2026),
  author: PostAuthor(
    did: 'did:plc:alice',
    handle: 'alice.craftsky.social',
  ),
  video: PostVideo(
    cid: 'bafyvideo',
    mime: 'video/mp4',
    size: 100,
    playlist: 'https://video.example/playlist.m3u8',
    thumbnail: 'https://video.example/thumbnail.jpg',
    aspectRatio: const PostImageAspectRatio(width: 16, height: 9),
  ),
);
