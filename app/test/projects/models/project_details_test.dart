import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  group('ProjectDetails', () {
    test('UT-003 gauge-like sub-objects parse and round-trip', () {
      final json = {
        'common': {'craftType': 'social.craftsky.feed.defs#knitting'},
        'details': {
          r'$type': knittingProjectDetailsType,
          'gauge': {'stitches': 20, 'rows': 28, 'measurement': 4, 'unit': 'in'},
        },
      };

      final project = ProjectMapper.fromMap(json);
      final details = project.details! as KnittingProjectDetails;

      expect(details.gauge, isA<ProjectGauge>());
      expect(details.gauge?.stitches, 20);
      expect(details.gauge?.rows, 28);
      expect(details.gauge?.measurement, 4);
      expect(details.gauge?.unit, 'in');
      expect(project.toMap(), json);
    });

    test('UT-004 knitting details discriminator maps to knitting variant', () {
      final project = ProjectMapper.fromMap({
        'common': {'craftType': 'social.craftsky.feed.defs#knitting'},
        'details': {
          r'$type': knittingProjectDetailsType,
          'projectType': 'social.craftsky.project.defs#garment',
          'projectSubtype': 'social.craftsky.project.knitting.defs#sweater',
          'yarnWeight': 'social.craftsky.project.defs#dk',
          'needleSizeMm': '4.0mm',
          'finishedSize': '42in chest',
        },
      });

      final details = project.details;
      expect(details, isA<KnittingProjectDetails>());
      expect((details! as KnittingProjectDetails).needleSizeMm, '4.0mm');
      expect(
        project.toMap()['details'],
        containsPair(r'$type', knittingProjectDetailsType),
      );
    });

    test('UT-005 crochet details discriminator maps to crochet variant', () {
      final project = ProjectMapper.fromMap({
        'common': {'craftType': 'social.craftsky.feed.defs#crochet'},
        'details': {
          r'$type': crochetProjectDetailsType,
          'projectType': 'social.craftsky.project.defs#toyHobby',
          'projectSubtype': 'social.craftsky.project.crochet.defs#amigurumi',
          'yarnWeight': 'social.craftsky.project.defs#worsted',
          'hookSizeMm': '5.0mm',
          'finishedSize': '12cm tall',
        },
      });

      final details = project.details;
      expect(details, isA<CrochetProjectDetails>());
      expect((details! as CrochetProjectDetails).hookSizeMm, '5.0mm');
      expect(
        project.toMap()['details'],
        containsPair(r'$type', crochetProjectDetailsType),
      );
    });

    test('UT-006 sewing and quilting discriminators map to variants', () {
      final sewing = ProjectMapper.fromMap({
        'common': {'craftType': 'social.craftsky.feed.defs#sewing'},
        'details': {
          r'$type': sewingProjectDetailsType,
          'projectType': 'social.craftsky.project.defs#garment',
          'projectSubtype': 'social.craftsky.project.sewing.defs#dress',
          'sizeMade': 'custom',
          'fitNotes': 'Added length.',
        },
      });
      final quilting = ProjectMapper.fromMap({
        'common': {'craftType': 'social.craftsky.feed.defs#quilting'},
        'details': {
          r'$type': quiltingProjectDetailsType,
          'projectType': 'social.craftsky.project.defs#quilt',
          'projectSubtype': 'social.craftsky.project.quilting.defs#throwQuilt',
          'size': '50in square',
          'piecingTechnique': 'social.craftsky.project.quilting.defs#improv',
          'quiltingMethod':
              'social.craftsky.project.quilting.defs#machineQuilted',
        },
      });

      expect(sewing.details, isA<SewingProjectDetails>());
      expect((sewing.details! as SewingProjectDetails).sizeMade, 'custom');
      expect(quilting.details, isA<QuiltingProjectDetails>());
      expect(
        (quilting.details! as QuiltingProjectDetails).quiltingMethod,
        'social.craftsky.project.quilting.defs#machineQuilted',
      );
    });

    test('UT-007 and UT-010 unknown details preserve raw data', () {
      final unknown = ProjectMapper.fromMap({
        'common': {'craftType': 'social.craftsky.feed.defs#knitting'},
        'details': {
          r'$type': 'social.craftsky.project.weaving#details',
          'loom': 'rigid heddle',
          'nested': {'width': 24},
          'sequence': [1, 2, 3],
        },
      });
      final missing = ProjectMapper.fromMap({
        'common': {'craftType': 'social.craftsky.feed.defs#knitting'},
        'details': {'loom': 'table loom'},
      });

      expect(unknown.details, isA<UnknownProjectDetails>());
      final details = unknown.details! as UnknownProjectDetails;
      expect(details.type, 'social.craftsky.project.weaving#details');
      expect(details.raw, containsPair('loom', 'rigid heddle'));
      expect(unknown.toMap()['details'], {
        r'$type': 'social.craftsky.project.weaving#details',
        'loom': 'rigid heddle',
        'nested': {'width': 24},
        'sequence': [1, 2, 3],
      });

      expect(missing.details, isA<UnknownProjectDetails>());
      expect((missing.details! as UnknownProjectDetails).type, isNull);
      expect(missing.toMap()['details'], {'loom': 'table loom'});
    });
  });
}
