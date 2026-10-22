import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_submit_adapter.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('scheduled project duration can be changed or cleared', () async {
    const existing = Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#knitting',
        duration: '2 days',
      ),
    );
    for (final duration in ['4 weeks', null]) {
      final args = await buildProjectComposerSubmitArguments(
        text: 'Project',
        langs: const ['en'],
        project: Project(common: existing.common.copyWith(duration: duration)),
        existingProject: existing,
        imagesState: const ComposerImagesState(images: []),
        generateFacets: (
          text, {
          includeMentions = true,
          includeLinks = true,
          includeTags = true,
        }) async => [],
      );
      expect(args.project.common.duration, duration);
    }
  });

  test(
    'scheduled form rebuild retains future metadata and additional crafts',
    () async {
      final existing = ProjectMapper.fromMap({
        'futureRoot': true,
        'common': {
          'craftType': 'social.craftsky.feed.defs#knitting',
          'futureCommon': true,
          'pattern': {'name': 'Aster', 'futurePattern': true},
        },
        'details': {r'$type': knittingProjectDetailsType, 'futureDetail': true},
      });
      final args = await buildProjectComposerSubmitArguments(
        text: 'New caption',
        langs: const ['en'],
        project: const Project(
          common: ProjectCommon(
            craftType: 'social.craftsky.feed.defs#knitting',
            additionalCrafts: [
              ProjectCraft(craftType: 'social.craftsky.feed.defs#sewing'),
            ],
            pattern: ProjectPattern(name: 'Aster'),
          ),
          details: KnittingProjectDetails(),
        ),
        existingProject: existing,
        imagesState: const ComposerImagesState(images: []),
        generateFacets: (
          _, {
          includeMentions = true,
          includeLinks = true,
          includeTags = true,
        }) async => [],
      );
      final map = args.project.toCreateMap();
      expect(map['futureRoot'], isTrue);
      expect((map['common'] as Map)['futureCommon'], isTrue);
      expect(
        ((map['common'] as Map)['pattern'] as Map)['futurePattern'],
        isTrue,
      );
      expect((map['details'] as Map)['futureDetail'], isTrue);
      expect(
        args.project.common.additionalCrafts!.single.craftType,
        'social.craftsky.feed.defs#sewing',
      );
    },
  );

  const langs = ['fr', 'en'];

  test(
    'UT-015 builds create arguments with facets, images, project and no reply',
    () async {
      const project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#embroidery',
        ),
      );
      const images = ComposerImagesState(
        images: [
          ComposerImageDraft(
            id: 'image-1',
            fileName: 'project.jpg',
            mimeType: 'image/jpeg',
            altText: 'Finished hoop',
            phase: ImageUploaded(
              UploadedDraftImage(
                cid: 'bafyimage',
                mime: 'image/jpeg',
                size: 42,
              ),
            ),
          ),
        ],
      );

      final args = await buildProjectComposerSubmitArguments(
        text: 'Hi #craft',
        langs: langs,
        project: project,
        imagesState: images,
        generateFacets:
            (
              text, {
              includeMentions = true,
              includeLinks = true,
              includeTags = true,
            }) async => [
              {'type': 'tag', 'tag': 'craft', 'text': text},
            ],
      );

      expect(args.text, 'Hi #craft');
      expect(args.langs, langs);
      expect(args.reply, isNull);
      expect(args.project, same(project));
      expect(args.images, hasLength(1));
      expect(args.images!.single.alt, 'Finished hoop');
      expect(args.facets, [
        {'type': 'tag', 'tag': 'craft', 'text': 'Hi #craft'},
      ]);
    },
  );

  test('UT-016 attaches scoped pattern field facets', () async {
    const project = Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#knitting',
        pattern: ProjectPattern(
          name: '#hitchhiker',
          designer: '@alice.craftsky.social',
          publisher: 'Plain Publisher',
        ),
      ),
    );

    final args = await buildProjectComposerSubmitArguments(
      text: 'Caption',
      langs: langs,
      project: project,
      imagesState: const ComposerImagesState(images: []),
      generateFacets:
          (
            text, {
            includeMentions = true,
            includeLinks = true,
            includeTags = true,
          }) async {
            if (includeTags && text == '#hitchhiker') {
              return [
                {'feature': 'tag', 'tag': 'hitchhiker'},
              ];
            }
            if (includeMentions && text == '@alice.craftsky.social') {
              return [
                {'feature': 'mention', 'did': 'did:plc:alice'},
              ];
            }
            return [];
          },
    );

    final pattern = args.project.common.pattern!;
    expect(pattern.nameFacets, [
      {'feature': 'tag', 'tag': 'hitchhiker'},
    ]);
    expect(pattern.designerFacets, [
      {'feature': 'mention', 'did': 'did:plc:alice'},
    ]);
    expect(pattern.publisherFacets, isNull);
  });

  test('AT-007 preserves stored facets for unchanged project text', () async {
    const bodyFacets = [
      {'feature': 'mention', 'did': 'did:plc:alice'},
    ];
    const nameFacets = [
      {'feature': 'tag', 'tag': 'cardigan'},
    ];
    const designerFacets = [
      {'feature': 'mention', 'did': 'did:plc:designer'},
    ];
    const project = Project(
      common: ProjectCommon(
        craftType: 'social.craftsky.feed.defs#knitting',
        pattern: ProjectPattern(
          name: '#cardigan',
          nameFacets: nameFacets,
          designer: '@designer.example',
          designerFacets: designerFacets,
        ),
      ),
    );

    final args = await buildProjectComposerSubmitArguments(
      text: 'Hello @alice.example',
      langs: langs,
      project: project,
      imagesState: const ComposerImagesState(images: []),
      existingText: 'Hello @alice.example',
      existingFacets: bodyFacets,
      existingProject: project,
      generateFacets: (
        _, {
        includeMentions = true,
        includeLinks = true,
        includeTags = true,
      }) => throw StateError('unchanged text must not regenerate facets'),
    );

    expect(args.facets, bodyFacets);
    expect(args.project.common.pattern?.nameFacets, nameFacets);
    expect(args.project.common.pattern?.designerFacets, designerFacets);
  });
}
