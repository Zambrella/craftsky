import 'package:craftsky_app/feed/composer/post_edit_presentation.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-002 retained image keeps its serving URL and updated alt', () {
    final post = postEditPresentation(
      _seed(),
      PostEditChanges({
        'images': [
          {'existingIndex': 0, 'alt': 'Updated alt'},
          {'image': _blob('bafynew', 'image/png'), 'alt': 'New image'},
        ],
      }),
    );
    expect(post.images!.first.fullsize, 'https://images.example/retained.jpg');
    expect(post.images!.first.alt, 'Updated alt');
    expect(post.images!.last.cid.value, 'bafynew');
    expect(
      post.images!.last.fullsize,
      'https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:alice/bafynew@png',
    );
    expect(post.likeCount, 7);
    expect(post.createdAt, DateTime.utc(2020));
  });

  test('AT-002 project leaves merge and explicit clears survive', () {
    final post = postEditPresentation(
      _seed(),
      PostEditChanges({
        'project': {
          'common': {
            'title': 'Updated title',
            'pattern': {'name': null, 'selfDrafted': true},
            'materials': [
              {'existingIndex': 1, 'text': 'Updated second'},
            ],
          },
        },
      }),
    );
    final common = post.project!.common;
    expect(common.title, 'Updated title');
    expect(common.craftType, 'social.craftsky.feed.defs#sewing');
    expect(common.pattern!.name, isNull);
    expect(common.pattern!.selfDrafted, isTrue);
    expect(common.materials!.single.text, 'Updated second');
    expect(
      common.materials!.single.extraFields.containsKey('existingIndex'),
      isFalse,
    );
  });

  test('AT-002 replacement video and external cards replace removed media', () {
    final video = postEditPresentation(
      _seed(),
      PostEditChanges({
        'images': null,
        'embed': {
          r'$type': 'app.bsky.embed.video',
          'video': _blob('bafyvideo', 'video/mp4'),
          'alt': 'Video alt',
        },
      }),
    );
    expect(video.images, isNull);
    expect(video.video!.alt, 'Video alt');
    expect(
      video.video!.playlist,
      'https://video.bsky.app/watch/did%3Aplc%3Aalice/bafyvideo/playlist.m3u8',
    );
    final external = postEditPresentation(
      _seed(),
      PostEditChanges({
        'images': null,
        'embed': {
          r'$type': 'app.bsky.embed.external',
          'external': {
            'uri': 'https://example.com',
            'title': 'New card',
            'description': 'Details',
            'thumb': _blob('bafythumb', 'image/jpeg'),
          },
        },
      }),
    );
    expect(external.video, isNull);
    expect(external.external!.title, 'New card');
    expect(
      external.external!.thumb!.url,
      'https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:alice/bafythumb@jpeg',
    );
  });
}

Map<String, Object?> _blob(String cid, String mime) => {
  r'$type': 'blob',
  'ref': {r'$link': cid},
  'mimeType': mime,
  'size': 100,
};

PostEditSeed _seed() {
  final post = Post(
    uri: 'at://did:plc:alice/social.craftsky.feed.post/base',
    cid: 'bafybase',
    rkey: 'base',
    text: 'Original',
    tags: const [],
    createdAt: DateTime.utc(2020),
    indexedAt: DateTime.utc(2020),
    author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
    likeCount: 7,
    repostCount: 2,
    replyCount: 1,
    viewerHasLiked: true,
    viewerHasReposted: false,
    viewerHasSaved: true,
    sponsored: false,
    project: const Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#sewing',
        title: 'Original title',
        pattern: ProjectPattern(name: 'Pattern'),
        materials: [
          ProjectMaterial(text: 'First'),
          ProjectMaterial(text: 'Second'),
        ],
      ),
    ),
    images: [
      PostImage(
        cid: 'bafyold',
        mime: 'image/jpeg',
        size: 100,
        alt: 'Old alt',
        fullsize: 'https://images.example/retained.jpg',
      ),
    ],
  );
  return PostEditSeed(
    post: post,
    baseCid: post.cid,
    recordRevision: null,
    editable: {
      'text': 'Original',
      'sponsored': false,
      'images': [
        {
          'existingIndex': 0,
          'image': _blob('bafyold', 'image/jpeg'),
          'alt': 'Old alt',
        },
      ],
      'project': {
        'common': {
          'craftType': 'social.craftsky.feed.defs#sewing',
          'title': 'Original title',
          'pattern': {'name': 'Pattern'},
          'materials': [
            {'existingIndex': 0, 'text': 'First'},
            {'existingIndex': 1, 'text': 'Second'},
          ],
        },
      },
    },
  );
}
