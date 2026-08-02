import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_hydrator.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('AT-007 hydrates every nested knitting project field', () {
    final values = hydrateScheduledProjectComposer({
      'common': {
        'craftType': ProjectOptionCatalogs.knittingCraftToken,
        'status': ProjectOptionCatalogs.finishedStatusToken,
        'title': 'Cardigan',
        'colors': ['blue'],
        'designTags': ['cables'],
        'materials': [
          {'text': 'Wool'},
        ],
        'pattern': {
          'url': 'https://example.com/pattern',
          'name': 'Warm cardigan',
          'difficulty': 'intermediate',
          'designer': 'A. Maker',
          'publisher': 'Patterns Ltd',
        },
      },
      'details': {
        r'$type': 'social.craftsky.project.knitting#details',
        'projectType': 'garment',
        'projectSubtype': 'cardigan',
        'yarnWeight': 'dk',
        'needleSizeMm': '4',
        'gauge': {
          'stitches': 20,
          'rows': 28,
          'measurement': 10,
          'unit': 'cm',
        },
        'finishedSize': 'M',
      },
    });

    expect(values[ProjectComposerFields.title], 'Cardigan');
    expect(values[ProjectComposerFields.materials], [
      const ProjectMaterial(text: 'Wool'),
    ]);
    expect(values[ProjectComposerFields.patternDesigner], 'A. Maker');
    expect(values[ProjectComposerFields.knittingProjectType], 'garment');
    expect(values[ProjectComposerFields.knittingNeedleSize], '4');
    expect(values[ProjectComposerFields.knittingGaugeStitches], 20);
    expect(values[ProjectComposerFields.knittingGaugeRows], 28);
    expect(values[ProjectComposerFields.knittingGaugeMeasurement], 10);
    expect(values[ProjectComposerFields.knittingGaugeUnit], 'cm');
    expect(values[ProjectComposerFields.knittingFinishedSize], 'M');
  });
}
