import 'package:craftsky_app/feed/composer/post_edit_changes.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_hydrator.dart';
import 'package:craftsky_app/projects/composer/project_composer_payload.dart';
import 'package:craftsky_app/projects/composer/project_draft_snapshot_adapter.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('keeps a published machine gauge when changing an unrelated field', () {
    const craft = ProjectOptionCatalogs.knittingCraftToken;
    final fieldName = ProjectDetailFields.formName(craft);
    final original = <String, dynamic>{
      ProjectComposerFields.craftType: craft,
      fieldName: <String, Object?>{
        'methods': [ProjectOptionCatalogs.knittingMachineMethodToken],
        'machineGaugeMm': 'standard gauge',
      },
    };
    final edited = {...original, ProjectComposerFields.title: 'Cardigan'};
    final result = buildProjectComposerPayload(
      formValues: edited,
      originalFormValues: original,
    );
    expect(result.errors, isEmpty);
    expect(
      (result.project!.toCreateMap()['details'] as Map)['machineGaugeMm'],
      'standard gauge',
    );
    expect(buildProjectComposerPayload(formValues: edited).errors, isNotEmpty);
  });

  test(
    'write ordered plural sizes and hydrate a legacy size or explicit clear',
    () {
      for (final (craft, key, scalar, plural) in [
        (
          ProjectOptionCatalogs.knittingCraftToken,
          'knittingNeedleSizes',
          'needleSizeMm',
          'needleSizesMm',
        ),
        (
          ProjectOptionCatalogs.crochetCraftToken,
          'crochetHookSizes',
          'hookSizeMm',
          'hookSizesMm',
        ),
      ]) {
        final result = buildProjectComposerPayload(
          formValues: {
            ProjectComposerFields.craftType: craft,
            key: ['4.5mm', '4.0mm'],
          },
        );
        final details = result.project!.toCreateMap()['details'] as Map;
        expect(details[plural], ['4.5mm', '4.0mm']);
        expect(details.containsKey(scalar), isFalse);

        Map<String, dynamic> hydrate(Map<String, dynamic> details) =>
            hydrateScheduledProjectComposer({
              'common': {'craftType': craft},
              'details': details,
            });

        expect(hydrate({scalar: '4.0mm'})[key], ['4.0mm']);
        expect(hydrate({scalar: '4.0mm', plural: <String>[]})[key], isEmpty);
      }
    },
  );

  test(
    'retain ordered pattern data and identity through hydration and edit',
    () {
      const adapter = ProjectDraftSnapshotAdapter();
      const basic = ProjectPattern(
        name: 'Bodice',
        extraFields: {'existingIndex': 0},
      );
      const sleeve = ProjectPattern(
        name: 'Sleeve',
        designer: 'Mina',
        extraFields: {'existingIndex': 1, 'future': true},
      );
      final form = hydrateScheduledProjectComposer({
        'common': {
          'craftType': ProjectOptionCatalogs.sewingCraftToken,
          'pattern': {'name': 'Suppressed'},
          'patterns': [basic.toMap(), sleeve.toMap()],
        },
      });
      final saved = adapter.decodeKnownFields(adapter.encodeKnownFields(form));
      final original = buildProjectComposerPayload(formValues: saved).project!;
      expect(original.common.effectivePatterns.map((entry) => entry.name), [
        'Bodice',
        'Sleeve',
      ]);
      expect(original.toCreateMap()['common'], isNot(contains('pattern')));

      final current = original.copyWith.common(
        patterns: [
          sleeve.copyWith(name: 'New sleeve'),
        ],
      );
      expect(buildPublishedProjectPatch(initial: original, current: current), {
        'common': {
          'patterns': [
            {'existingIndex': 1, 'name': 'New sleeve'},
          ],
        },
      });
      expect(
        buildPublishedProjectPatch(
          initial: original,
          current: original.copyWith.common(patterns: const []),
        ),
        {
          'common': {'patterns': <Object?>[]},
        },
      );
      final blocked = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.sewingCraftToken,
          ProjectComposerFields.patterns: [
            for (var i = 1; i <= 6; i++) ProjectPattern(name: 'Pattern $i'),
          ],
        },
      );
      expect(
        blocked.errors.single.code,
        ProjectComposerValidationCode.tooManyValues,
      );
    },
  );

  test('a facet budget refresh does not edit an unchanged legacy pattern', () {
    final form = hydrateScheduledProjectComposer({
      'common': {
        'craftType': ProjectOptionCatalogs.sewingCraftToken,
        'pattern': {
          'name': '#quilt',
          'nameFacets': [
            {
              'index': {'byteStart': 0, 'byteEnd': 6},
              'features': [
                {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'quilt'},
              ],
            },
          ],
          'existingIndex': 0,
        },
      },
    });
    final original = buildProjectComposerPayload(formValues: form).project!;
    final refreshed = original.copyWith.common(
      patterns: [
        original.common.effectivePatterns.single.copyWith(nameFacets: null),
      ],
    );
    expect(
      buildPublishedProjectPatch(initial: original, current: refreshed),
      isEmpty,
    );
  });

  test('edition keeps Embroidery styles under their original methods', () {
    const craft = ProjectOptionCatalogs.embroideryCraftToken;
    const type = 'social.craftsky.project.embroidery#details';
    final field = ProjectDetailFields.formName(craft);
    final originalForm = hydrateScheduledProjectComposer({
      'common': {'craftType': craft},
      'details': {
        r'$type': type,
        'methods': [
          {
            'existingIndex': 0,
            'method': 'hand',
            'styles': [
              {'existingIndex': 0, 'style': 'sashiko'},
            ],
          },
          {
            'existingIndex': 1,
            'method': 'machine',
            'styles': [
              {'existingIndex': 0, 'style': '3d-puff', 'future': 12},
            ],
          },
        ],
      },
    });
    final updatedForm = {
      ...originalForm,
      field: {
        'methods': [
          {
            'existingIndex': 1,
            'method': 'machine',
            'styles': [
              {'existingIndex': 0, 'style': '3d-puff', 'future': 12},
            ],
          },
        ],
      },
    };
    final initial = buildProjectComposerPayload(formValues: originalForm)
        .project!;
    final current = buildProjectComposerPayload(formValues: updatedForm)
        .project!;
    expect(
      buildPublishedProjectPatch(initial: initial, current: current),
      {
        'details': {
          'methods': [
            {'existingIndex': 1},
          ],
        },
      },
    );
  });

  test('duration is trimmed, optional, and bounded for project records', () {
    for (final (input, expected, valid) in [
      ('  4 weeks  ', '4 weeks', true),
      ('   ', null, true),
      ('a' * 100, 'a' * 100, true),
      ('a' * 101, null, false),
      ('👨‍👩‍👧‍👦' * 20, '👨‍👩‍👧‍👦' * 20, true),
      ('👨‍👩‍👧‍👦' * 21, null, false),
    ]) {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.sewingCraftToken,
          ProjectComposerFields.duration: input,
        },
      );
      expect(result.project != null, valid);
      if (valid) {
        expect(result.project!.common.duration, expected);
        expect(
          (result.project!.toCreateMap()['common'] as Map)['duration'],
          expected,
        );
      } else {
        expect(result.errors.single.fieldName, ProjectComposerFields.duration);
        expect(
          result.errors.single.code,
          ProjectComposerValidationCode.tooLong,
        );
      }
    }
  });

  test(
    'additional crafts exclude the primary and duplicates in create payload',
    () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.sewingCraftToken,
          ProjectComposerFields.additionalCrafts: [
            ProjectOptionCatalogs.knittingCraftToken,
            ProjectOptionCatalogs.sewingCraftToken,
            ProjectOptionCatalogs.knittingCraftToken,
          ],
        },
      );
      expect(result.errors, isEmpty);
      expect(
        result.project!.toCreateMap()['common'],
        containsPair('additionalCrafts', [
          {'craftType': ProjectOptionCatalogs.knittingCraftToken},
        ]),
      );
    },
  );

  test('restored sewing notes respect grapheme and byte boundaries', () {
    for (final (notes, valid) in [
      ('é' * 600, true),
      ('a' * 601, false),
      ('👨‍👩‍👧‍👦' * 240, true),
      ('👨‍👩‍👧‍👦' * 241, false),
    ]) {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.sewingCraftToken,
          ProjectComposerFields.sewingFitNotes: notes,
        },
      );
      expect(result.project != null, valid);
      if (!valid) {
        expect(
          result.errors.single.fieldName,
          ProjectComposerFields.sewingFitNotes,
        );
        expect(
          result.errors.single.code,
          ProjectComposerValidationCode.tooLong,
        );
      }
    }
  });

  group('Project composer payload', () {
    test('UT-007 maps common fields, cleared status and metadata strings', () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.embroideryCraftToken,
          ProjectComposerFields.status: null,
          ProjectComposerFields.title: '  Hoop sampler  ',
          ProjectComposerFields.materials: const [
            ProjectMaterial(text: ' linen '),
            ProjectMaterial(text: ''),
            ProjectMaterial(text: 'cotton'),
          ],
          ProjectComposerFields.colours: ['blue', 'cream'],
          ProjectComposerFields.designTags: [
            'social.craftsky.project.defs#floral',
          ],
        },
      );

      expect(result.errors, isEmpty);
      final project = result.project!;
      expect(
        project.common.craftType,
        ProjectOptionCatalogs.embroideryCraftToken,
      );
      expect(project.common.status, isNull);
      expect(project.common.title, 'Hoop sampler');
      expect(project.common.materials, const [
        ProjectMaterial(text: 'linen'),
        ProjectMaterial(text: 'cotton'),
      ]);
      expect(project.common.colors, ['blue', 'cream']);
      expect(project.common.designTags, [
        'social.craftsky.project.defs#floral',
      ]);
      expect(project.common.pattern, isNull);
      expect(project.details, isNull);
    });

    test('UT-007 trims and includes only non-empty pattern fields', () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.knittingCraftToken,
          ProjectComposerFields.patternName: '  Hitchhiker ',
          ProjectComposerFields.patternUrl: ' ',
          ProjectComposerFields.patternDifficulty:
              'social.craftsky.feed.defs#intermediate',
          ProjectComposerFields.patternDesigner: ' Martina Behm ',
          ProjectComposerFields.patternPublisher: '',
        },
      );

      expect(result.errors, isEmpty);
      expect(
        result.project!.common.effectivePatterns.firstOrNull,
        isA<ProjectPattern>(),
      );
      expect(
        result.project!.common.effectivePatterns.firstOrNull?.name,
        'Hitchhiker',
      );
      expect(result.project!.common.effectivePatterns.firstOrNull?.url, isNull);
      expect(
        result.project!.common.effectivePatterns.firstOrNull?.difficulty,
        'social.craftsky.feed.defs#intermediate',
      );
      expect(
        result.project!.common.effectivePatterns.firstOrNull?.designer,
        'Martina Behm',
      );
      expect(
        result.project!.common.effectivePatterns.firstOrNull?.publisher,
        isNull,
      );
    });

    test('UT-007 treats a lone pattern hashtag placeholder as empty', () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.knittingCraftToken,
          ProjectComposerFields.patternName: '#',
        },
      );

      expect(result.errors, isEmpty);
      expect(result.project!.common.effectivePatterns.firstOrNull, isNull);
    });

    test('self-drafted alone creates pattern metadata', () {
      final result = buildProjectComposerPayload(
        formValues: const {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.knittingCraftToken,
          ProjectComposerFields.patternSelfDrafted: true,
        },
      );

      expect(result.errors, isEmpty);
      expect(
        result.project!.common.effectivePatterns.firstOrNull?.selfDrafted,
        isTrue,
      );
      expect(
        result.project!.common.effectivePatterns.firstOrNull?.name,
        isNull,
      );
    });

    test('difficulty catalog has the approved four values', () {
      expect(
        ProjectOptionCatalogs.patternDifficulties.map(
          (option) => (option.value, option.label),
        ),
        const [
          ('social.craftsky.feed.defs#beginner', 'Beginner'),
          (
            'social.craftsky.feed.defs#confidentBeginner',
            'Confident beginner',
          ),
          ('social.craftsky.feed.defs#intermediate', 'Intermediate'),
          ('social.craftsky.feed.defs#advanced', 'Advanced'),
        ],
      );
    });

    test('UT-007 reports missing craft type before building a project', () {
      final result = buildProjectComposerPayload(formValues: const {});

      expect(result.project, isNull);
      expect(result.errors.single.fieldName, ProjectComposerFields.craftType);
      expect(result.errors.single.code, ProjectComposerValidationCode.required);
    });

    test('UT-008 builds sewing details only when sewing values exist', () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.sewingCraftToken,
          ProjectComposerFields.sewingProjectType:
              'social.craftsky.project.defs#garment',
          ProjectComposerFields.sewingProjectSubtype:
              'social.craftsky.project.sewing.defs#dress',
          ProjectComposerFields.sewingSizeMade: ' custom ',
          ProjectComposerFields.sewingFitNotes: ' Added length. ',
        },
      );

      expect(result.errors, isEmpty);
      expect(result.project!.details, isA<SewingProjectDetails>());
      final details = result.project!.details! as SewingProjectDetails;
      expect(details.projectType, 'social.craftsky.project.defs#garment');
      expect(
        details.projectSubtype,
        'social.craftsky.project.sewing.defs#dress',
      );
      expect(details.sizeMade, 'custom');
      expect(details.fitNotes, 'Added length.');
    });

    test('UT-008 omits empty sewing details', () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.sewingCraftToken,
          ProjectComposerFields.sewingProjectType: ' ',
          ProjectComposerFields.sewingFitNotes: '',
        },
      );

      expect(result.errors, isEmpty);
      expect(result.project!.details, isNull);
    });

    test(
      'UT-009 builds knitting details with valid gauge and optional rows',
      () {
        final result = buildProjectComposerPayload(
          formValues: {
            ProjectComposerFields.craftType:
                ProjectOptionCatalogs.knittingCraftToken,
            ProjectComposerFields.knittingProjectType:
                'social.craftsky.project.defs#garment',
            ProjectComposerFields.knittingProjectSubtype:
                'social.craftsky.project.knitting.defs#sweater',
            ProjectComposerFields.knittingYarnWeight:
                'social.craftsky.project.defs#dk',
            ProjectComposerFields.knittingNeedleSize: '4.0mm',
            ProjectComposerFields.knittingGaugeStitches: '20',
            ProjectComposerFields.knittingGaugeMeasurement: '4',
            ProjectComposerFields.knittingGaugeUnit: 'in',
            ProjectComposerFields.knittingFinishedSize: ' 42in chest ',
          },
        );

        expect(result.errors, isEmpty);
        expect(result.project!.details, isA<KnittingProjectDetails>());
        final details = result.project!.details! as KnittingProjectDetails;
        expect(
          details.projectSubtype,
          'social.craftsky.project.knitting.defs#sweater',
        );
        expect(details.yarnWeight, 'social.craftsky.project.defs#dk');
        expect(details.needleSizesMm, ['4.0mm']);
        expect(details.finishedSize, '42in chest');
        expect(details.gauge?.stitches, 20);
        expect(details.gauge?.rows, isNull);
        expect(details.gauge?.measurement, 4);
        expect(details.gauge?.unit, 'in');
      },
    );

    test('UT-009 accepts typed numeric gauge values from form fields', () {
      final result = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.knittingCraftToken,
          ProjectComposerFields.knittingGaugeStitches: 22,
          ProjectComposerFields.knittingGaugeRows: 30,
          ProjectComposerFields.knittingGaugeMeasurement: 10,
          ProjectComposerFields.knittingGaugeUnit: 'cm',
        },
      );

      expect(result.errors, isEmpty);
      final details = result.project!.details! as KnittingProjectDetails;
      expect(details.gauge?.stitches, 22);
      expect(details.gauge?.rows, 30);
      expect(details.gauge?.measurement, 10);
      expect(details.gauge?.unit, 'cm');
    });

    test('UT-009 rejects partial and invalid knitting gauge', () {
      for (final values in [
        {ProjectComposerFields.knittingGaugeStitches: '20'},
        {
          ProjectComposerFields.knittingGaugeStitches: '20',
          ProjectComposerFields.knittingGaugeMeasurement: '4',
        },
        {
          ProjectComposerFields.knittingGaugeStitches: '0',
          ProjectComposerFields.knittingGaugeMeasurement: '4',
          ProjectComposerFields.knittingGaugeUnit: 'in',
        },
        {
          ProjectComposerFields.knittingGaugeStitches: '20.5',
          ProjectComposerFields.knittingGaugeMeasurement: '4',
          ProjectComposerFields.knittingGaugeUnit: 'in',
        },
      ]) {
        final result = buildProjectComposerPayload(
          formValues: {
            ProjectComposerFields.craftType:
                ProjectOptionCatalogs.knittingCraftToken,
            ...values,
          },
        );

        expect(result.project, isNull);
        expect(
          result.errors.single.code,
          ProjectComposerValidationCode.invalidGauge,
        );
      }
    });

    test('UT-010 builds crochet details and rejects invalid gauge', () {
      final valid = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.crochetCraftToken,
          ProjectComposerFields.crochetProjectType:
              'social.craftsky.project.defs#toyHobby',
          ProjectComposerFields.crochetProjectSubtype:
              'social.craftsky.project.crochet.defs#amigurumi',
          ProjectComposerFields.crochetYarnWeight:
              'social.craftsky.project.defs#worsted',
          ProjectComposerFields.crochetHookSize: '5.0mm',
          ProjectComposerFields.crochetGaugeStitches: '16',
          ProjectComposerFields.crochetGaugeRows: '20',
          ProjectComposerFields.crochetGaugeMeasurement: '10',
          ProjectComposerFields.crochetGaugeUnit: 'cm',
          ProjectComposerFields.crochetFinishedSize: ' 12cm tall ',
        },
      );

      expect(valid.errors, isEmpty);
      expect(valid.project!.details, isA<CrochetProjectDetails>());
      final details = valid.project!.details! as CrochetProjectDetails;
      expect(
        details.projectSubtype,
        'social.craftsky.project.crochet.defs#amigurumi',
      );
      expect(details.hookSizesMm, ['5.0mm']);
      expect(details.gauge?.rows, 20);
      expect(details.gauge?.unit, 'cm');

      final invalid = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.crochetCraftToken,
          ProjectComposerFields.crochetGaugeMeasurement: '10',
          ProjectComposerFields.crochetGaugeUnit: 'cm',
        },
      );

      expect(invalid.project, isNull);
      expect(
        invalid.errors.single.code,
        ProjectComposerValidationCode.invalidGauge,
      );
    });

    test('UT-011 builds quilting details and omits empty details', () {
      final valid = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.quiltingCraftToken,
          ProjectComposerFields.quiltingProjectType:
              'social.craftsky.project.defs#quilt',
          ProjectComposerFields.quiltingProjectSubtype:
              'social.craftsky.project.quilting.defs#throwQuilt',
          ProjectComposerFields.quiltingSize: ' 50in square ',
          ProjectComposerFields.quiltingPiecingTechnique:
              'social.craftsky.project.quilting.defs#improv',
          ProjectComposerFields.quiltingMethod:
              'social.craftsky.project.quilting.defs#machineQuilted',
        },
      );

      expect(valid.errors, isEmpty);
      expect(valid.project!.details, isA<QuiltingProjectDetails>());
      final details = valid.project!.details! as QuiltingProjectDetails;
      expect(
        details.projectSubtype,
        'social.craftsky.project.quilting.defs#throwQuilt',
      );
      expect(details.size, '50in square');
      expect(
        details.piecingTechnique,
        'social.craftsky.project.quilting.defs#improv',
      );
      expect(
        details.quiltingMethod,
        'social.craftsky.project.quilting.defs#machineQuilted',
      );

      final empty = buildProjectComposerPayload(
        formValues: {
          ProjectComposerFields.craftType:
              ProjectOptionCatalogs.quiltingCraftToken,
        },
      );

      expect(empty.errors, isEmpty);
      expect(empty.project!.details, isNull);
    });

    test('UT-012 common-only and future craft payloads carry no details', () {
      for (final craft in [
        ProjectOptionCatalogs.embroideryCraftToken,
        'social.craftsky.feed.defs#weaving',
      ]) {
        final result = buildProjectComposerPayload(
          formValues: {ProjectComposerFields.craftType: craft},
        );

        expect(result.errors, isEmpty);
        expect(result.project!.common.craftType, craft);
        expect(result.project!.details, isNull);
      }
    });
  });
}
