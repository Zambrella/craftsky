import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_payload.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Project composer payload', () {
    test('UT-007 maps common fields, default status and metadata strings', () {
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
      expect(project.common.status, ProjectOptionCatalogs.finishedStatusToken);
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
      expect(result.project!.common.pattern, isA<ProjectPattern>());
      expect(result.project!.common.pattern?.name, 'Hitchhiker');
      expect(result.project!.common.pattern?.url, isNull);
      expect(
        result.project!.common.pattern?.difficulty,
        'social.craftsky.feed.defs#intermediate',
      );
      expect(result.project!.common.pattern?.designer, 'Martina Behm');
      expect(result.project!.common.pattern?.publisher, isNull);
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
      expect(result.project!.common.pattern, isNull);
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
        expect(details.needleSizeMm, '4.0mm');
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
      expect(details.hookSizeMm, '5.0mm');
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
