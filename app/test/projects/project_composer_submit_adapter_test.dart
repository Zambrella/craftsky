import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_submit_adapter.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:flutter_test/flutter_test.dart';

final class _PatternMentionResolver implements MentionResolver {
  const new();

  @override
  Future<String?> didForHandle(String handle) async =>
      handle == 'maker.example' ? 'did:plc:maker' : null;
}

void main() {
  test(
    'IT-005 a saved credit link leaves the last hashtag slot free',
    () async {
      const link = {
        'index': {'byteStart': 0, 'byteEnd': 7},
        'features': [
          {
            r'$type': 'app.bsky.richtext.facet#link',
            'uri': 'https://example.com/pattern-credit',
          },
        ],
      };
      const first = ProjectPattern(
        designer: '#credit',
        designerFacets: [link],
        extraFields: {'existingIndex': 0},
      );
      const original = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#knitting',
          patterns: [first],
        ),
      );
      final result = await buildProjectComposerSubmitArguments(
        text: List.generate(9, (index) => '#tag$index').join(' '),
        langs: const ['en'],
        project: original.copyWith.common(
          patterns: [
            first,
            const ProjectPattern(
              name: 'Later #later',
              publisher: '@maker.example',
            ),
          ],
        ),
        existingProject: original,
        imagesState: const ComposerImagesState(images: []),
        generateFacets: const FacetGenerator(
          mentionResolver: _PatternMentionResolver(),
        ).generate,
      );
      final patterns = result.project.common.patterns!;
      expect(result.facets, hasLength(9));
      expect(patterns.first.designerFacets, [link]);
      expect(patterns.last.nameFacets, [
        {
          'index': {'byteStart': 6, 'byteEnd': 12},
          'features': [
            {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'later'},
          ],
        },
      ]);
      expect(patterns.last.publisherFacets!.single['features'], [
        {r'$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:maker'},
      ]);
    },
  );

  test(
    'one budget covers every project field and refreshes unchanged facets',
    () async {
      final caption = List.generate(8, (index) => '#tag$index').join(' ');
      const project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#sewing',
          tags: ['TAG0', 'structured'],
          pattern: ProjectPattern(name: '#pattern'),
          materials: [ProjectMaterial(text: '#material')],
        ),
      );
      Future<ProjectComposerSubmitArguments> submit(
        String text,
        Project value,
      ) => buildProjectComposerSubmitArguments(
        text: text,
        langs: const ['en'],
        project: value,
        existingProject: value,
        imagesState: const ComposerImagesState(images: []),
        generateFacets: (
          _, {
          includeMentions = true,
          includeLinks = true,
          includeTags = true,
        }) async => [],
      );
      final result = await submit(caption, project);
      expect(result.facets, hasLength(8));
      expect(result.project.common.pattern!.nameFacets, hasLength(1));
      expect(result.project.common.materials!.single.facets, isNull);
      final reopened = await submit('', result.project);
      expect(reopened.project.common.materials!.single.facets, hasLength(1));
      expect(project.common.pattern!.nameFacets, isNull);
    },
  );

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
        text: ' \nHi #craft\n\t ',
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
              {
                'index': {'byteStart': 3, 'byteEnd': 9},
                'features': [
                  {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'craft'},
                ],
              },
            ],
      );

      expect(args.text, 'Hi #craft');
      expect(args.langs, langs);
      expect(args.reply, isNull);
      expect(args.project.toCreateMap(), project.toCreateMap());
      expect(args.images, hasLength(1));
      expect(args.images!.single.alt, 'Finished hoop');
      expect(args.facets, [
        {
          'index': {'byteStart': 3, 'byteEnd': 9},
          'features': [
            {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'craft'},
          ],
        },
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
                {
                  'index': {'byteStart': 0, 'byteEnd': 11},
                  'features': [
                    {
                      r'$type': 'app.bsky.richtext.facet#tag',
                      'tag': 'hitchhiker',
                    },
                  ],
                },
              ];
            }
            if (includeMentions && text == '@alice.craftsky.social') {
              return [
                {
                  'index': {'byteStart': 0, 'byteEnd': 22},
                  'features': [
                    {
                      r'$type': 'app.bsky.richtext.facet#mention',
                      'did': 'did:plc:alice',
                    },
                  ],
                },
              ];
            }
            return [];
          },
    );

    final pattern = args.project.common.pattern!;
    expect(pattern.nameFacets, [
      {
        'index': {'byteStart': 0, 'byteEnd': 11},
        'features': [
          {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'hitchhiker'},
        ],
      },
    ]);
    expect(pattern.designerFacets, [
      {
        'index': {'byteStart': 0, 'byteEnd': 22},
        'features': [
          {r'$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:alice'},
        ],
      },
    ]);
    expect(pattern.publisherFacets, isNull);
  });

  test('AT-007 preserves stored facets for unchanged project text', () async {
    const bodyFacets = [
      {
        'index': {'byteStart': 6, 'byteEnd': 20},
        'features': [
          {r'$type': 'app.bsky.richtext.facet#mention', 'did': 'did:plc:alice'},
        ],
      },
    ];
    const nameFacets = [
      {
        'index': {'byteStart': 0, 'byteEnd': 9},
        'features': [
          {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'cardigan'},
        ],
      },
    ];
    const designerFacets = [
      {
        'index': {'byteStart': 0, 'byteEnd': 17},
        'features': [
          {
            r'$type': 'app.bsky.richtext.facet#mention',
            'did': 'did:plc:designer',
          },
        ],
      },
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
  test(
    'regenerates facets when trimming changes stored text offsets',
    () async {
      String? generatedText;
      const expectedFacets = [
        {
          'index': {'byteStart': 3, 'byteEnd': 9},
          'features': [
            {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'craft'},
          ],
        },
      ];
      final args = await buildProjectComposerSubmitArguments(
        text: ' \nHi #craft\n ',
        langs: langs,
        project: const Project(
          common: ProjectCommon(
            craftType: 'social.craftsky.feed.defs#embroidery',
          ),
        ),
        imagesState: const ComposerImagesState(images: []),
        existingText: ' \nHi #craft\n ',
        existingFacets: const [
          {'stale': true},
        ],
        generateFacets:
            (
              text, {
              includeMentions = true,
              includeLinks = true,
              includeTags = true,
            }) async {
              generatedText = text;
              return expectedFacets;
            },
      );
      expect(args.text, 'Hi #craft');
      expect(generatedText, 'Hi #craft');
      expect(args.facets, expectedFacets);
    },
  );
}
