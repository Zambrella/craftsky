import 'package:craftsky_app/feed/composer/post_edit_changes.dart';
import 'package:craftsky_app/feed/composer/post_edit_hydrator.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_payload.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-001 project prefill keeps self-drafted pattern and material identities',
    () {
      final post = Post(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/project',
        cid: 'bafybase',
        rkey: 'project',
        text: 'Project caption',
        tags: const [],
        createdAt: DateTime.utc(2026),
        indexedAt: DateTime.utc(2026),
        author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
        likeCount: 4,
        repostCount: 2,
        replyCount: 1,
        viewerHasLiked: false,
        viewerHasReposted: false,
        viewerHasSaved: true,
        sponsored: false,
        project: const Project(
          common: ProjectCommon(craftType: 'social.craftsky.feed.defs#sewing'),
        ),
      );
      final seed = PostEditSeed(
        post: post,
        baseCid: post.cid,
        recordRevision: '3aaaaaaaaaaa2',
        editable: {
          'text': post.text,
          'sponsored': false,
          'project': {
            'common': {
              'craftType': 'social.craftsky.feed.defs#sewing',
              'title': 'Linen shirt',
              'pattern': {'selfDrafted': true, 'name': 'My pattern'},
              'materials': [
                {'existingIndex': 0, 'text': 'Linen'},
                {'existingIndex': 1, 'text': 'Thread'},
              ],
              'additionalCrafts': [
                {
                  'existingIndex': 0,
                  'craftType': 'social.craftsky.feed.defs#embroidery',
                },
              ],
            },
            'details': {
              r'$type': 'social.craftsky.project.sewing#details',
              'sizeMade': 'M',
            },
          },
        },
      );
      final values = hydratePublishedProjectComposer(seed);
      expect(values[ProjectComposerFields.patternSelfDrafted], isTrue);
      expect(values[ProjectComposerFields.title], 'Linen shirt');
      expect(values[ProjectComposerFields.sewingSizeMade], 'M');
      final materials =
          values[ProjectComposerFields.materials] as List<ProjectMaterial>;
      expect(
        materials.map((material) => material.extraFields['existingIndex']),
        [0, 1],
      );
      expect(materials.map((material) => material.text), ['Linen', 'Thread']);
      final payload = buildProjectComposerPayload(formValues: values).project!;
      final restored = restorePublishedProjectEntries(
        seed: seed,
        payload: payload,
        formValues: values,
      );
      expect(
        restored.common.materials!.map(
          (material) => material.extraFields['existingIndex'],
        ),
        [0, 1],
      );
      expect(
        restored.common.additionalCrafts!.single.extraFields['existingIndex'],
        0,
      );
      final clearedPattern = Map<String, dynamic>.of(values)
        ..[ProjectComposerFields.patternName] = ''
        ..[ProjectComposerFields.patternSelfDrafted] = false;
      final clearedPayload = buildProjectComposerPayload(
        formValues: clearedPattern,
      ).project!;
      final cleared = restorePublishedProjectEntries(
        seed: seed,
        payload: clearedPayload,
        formValues: clearedPattern,
      );
      expect(cleared.common.pattern?.selfDrafted, isFalse);
    },
  );
  test('UT-001 published image prefill keeps exact indices, alt and safe '
      'display URLs', () {
    final post = Post(
      uri: 'at://did:plc:alice/social.craftsky.feed.post/base',
      cid: 'bafybase',
      rkey: 'base',
      text: 'Published caption',
      tags: const [],
      createdAt: DateTime.utc(2026),
      indexedAt: DateTime.utc(2026),
      author: PostAuthor(did: 'did:plc:alice', handle: 'alice.example'),
      likeCount: 4,
      repostCount: 2,
      replyCount: 1,
      viewerHasLiked: false,
      viewerHasReposted: false,
      viewerHasSaved: true,
      sponsored: false,
      images: [
        PostImage(
          cid: 'bafyimage',
          mime: 'image/jpeg',
          size: 200,
          alt: 'Original alt',
          fullsize: 'https://cdn.example/image.jpeg',
        ),
      ],
    );
    final seed = PostEditSeed(
      post: post,
      baseCid: Cid.parse('bafybase'),
      recordRevision: '3aaaaaaaaaaa2',
      editable: {
        'text': post.text,
        'sponsored': false,
        'images': [
          {
            'existingIndex': 0,
            'alt': 'Original alt',
            'image': {'mimeType': 'image/jpeg'},
            'aspectRatio': {'width': 3, 'height': 2},
          },
        ],
      },
    );
    final image = hydratePublishedPostImages(seed).single;
    expect(image.altText, 'Original alt');
    expect(image.previewBytes, isNull);
    expect(image.previewAspectRatio?.width, 3);
    expect(image.phase, isA<PublishedImageReady>());
    expect((image.phase as PublishedImageReady).existingIndex, 0);
    expect(
      (image.phase as PublishedImageReady).url,
      'https://cdn.example/image.jpeg',
    );
    expect(image.id, contains('bafybase'));
  });
}
