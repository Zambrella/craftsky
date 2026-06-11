import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ProjectOptionCatalogs', () {
    test('UT-006 exposes representative known labels and token values', () {
      expect(
        ProjectOptionCatalogs.finishedStatusToken,
        'social.craftsky.feed.defs#finished',
      );
      expect(
        ProjectOptionCatalogs.statuses
            .singleWhere((o) => o.label == 'Finished')
            .value,
        ProjectOptionCatalogs.finishedStatusToken,
      );
      expect(
        ProjectOptionCatalogs.craftTypes
            .singleWhere((o) => o.label == 'Knitting')
            .value,
        'social.craftsky.feed.defs#knitting',
      );
      expect(
        ProjectOptionCatalogs.patternDifficulties
            .singleWhere((o) => o.label == 'Intermediate')
            .value,
        'social.craftsky.feed.defs#intermediate',
      );
      expect(
        ProjectOptionCatalogs.yarnWeights
            .singleWhere((o) => o.label == 'DK')
            .value,
        'social.craftsky.project.defs#dk',
      );
      expect(
        ProjectOptionCatalogs.needleSizes
            .singleWhere((o) => o.value == '4.0mm')
            .label,
        contains('4.0'),
      );
      expect(
        ProjectOptionCatalogs.hookSizes
            .singleWhere((o) => o.value == '5.0mm')
            .label,
        contains('5.0'),
      );
      expect(
        ProjectOptionCatalogs.gaugeUnits
            .singleWhere((o) => o.label == 'in')
            .value,
        'in',
      );
      expect(
        ProjectOptionCatalogs.colours
            .singleWhere((o) => o.value == 'blue')
            .label,
        'Blue',
      );
      expect(
        ProjectOptionCatalogs.designTags
            .singleWhere((o) => o.label == 'Floral')
            .value,
        'social.craftsky.project.defs#floral',
      );
      expect(
        ProjectOptionCatalogs.quiltingPiecingTechniques
            .singleWhere((o) => o.label == 'Improv')
            .value,
        'social.craftsky.project.quilting.defs#improv',
      );
      expect(
        ProjectOptionCatalogs.quiltingMethods
            .singleWhere((o) => o.label == 'Machine quilted')
            .value,
        'social.craftsky.project.quilting.defs#machineQuilted',
      );
    });

    test(
      'UT-006 keeps catalogs UI-only and DTO fields string-backed',
      () {
        const project = Project(
          common: ProjectCommon(
            craftType: ProjectOptionCatalogs.knittingCraftToken,
            status: ProjectOptionCatalogs.finishedStatusToken,
          ),
        );

        expect(project.common.craftType, isA<String>());
        expect(project.common.status, isA<String>());
        expect(project.common.craftType, 'social.craftsky.feed.defs#knitting');
      },
    );
  });
}
