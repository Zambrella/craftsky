import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Project subtype filtering', () {
    test(
      'UT-013 disables subtype selection until a project type is selected',
      () {
        expect(
          ProjectOptionCatalogs.isSubtypeSelectionEnabled(
            craftToken: ProjectOptionCatalogs.knittingCraftToken,
            projectTypeToken: null,
          ),
          isFalse,
        );
        expect(
          ProjectOptionCatalogs.projectSubtypesFor(
            craftToken: ProjectOptionCatalogs.knittingCraftToken,
            projectTypeToken: null,
          ),
          isEmpty,
        );
      },
    );

    test('UT-013 filters subtypes to the active project type', () {
      final sewingGarments = ProjectOptionCatalogs.projectSubtypesFor(
        craftToken: ProjectOptionCatalogs.sewingCraftToken,
        projectTypeToken: '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
      );

      expect(
        sewingGarments.map((o) => o.value),
        contains('social.craftsky.project.sewing.defs#dress'),
      );
      expect(
        sewingGarments.map((o) => o.value),
        isNot(contains('social.craftsky.project.sewing.defs#bag')),
      );
    });

    test('UT-013 clears an invalid subtype when project type changes', () {
      expect(
        ProjectOptionCatalogs.clearInvalidSubtype(
          craftToken: ProjectOptionCatalogs.crochetCraftToken,
          projectTypeToken:
              '${ProjectOptionCatalogs.projectDefsPrefix}#toyHobby',
          subtypeToken: 'social.craftsky.project.crochet.defs#amigurumi',
        ),
        'social.craftsky.project.crochet.defs#amigurumi',
      );
      expect(
        ProjectOptionCatalogs.clearInvalidSubtype(
          craftToken: ProjectOptionCatalogs.crochetCraftToken,
          projectTypeToken:
              '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
          subtypeToken: 'social.craftsky.project.crochet.defs#amigurumi',
        ),
        isNull,
      );
    });

    test('keeps the subtype union for all selected project types', () {
      final types = [
        '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
        '${ProjectOptionCatalogs.projectDefsPrefix}#toyHobby',
      ];
      final options = ProjectOptionCatalogs.projectSubtypesForTypes(
        craftToken: ProjectOptionCatalogs.crochetCraftToken,
        projectTypeTokens: types,
      );

      expect(
        options.map((option) => option.value),
        containsAll([
          'social.craftsky.project.crochet.defs#sweater',
          'social.craftsky.project.crochet.defs#amigurumi',
        ]),
      );
    });

    test('removes only subtypes invalidated by parent changes', () {
      final retained = ProjectOptionCatalogs.retainValidSubtypes(
        craftToken: ProjectOptionCatalogs.crochetCraftToken,
        projectTypeTokens: [
          '${ProjectOptionCatalogs.projectDefsPrefix}#garment',
        ],
        subtypeTokens: [
          'social.craftsky.project.crochet.defs#sweater',
          'social.craftsky.project.crochet.defs#amigurumi',
        ],
      );

      expect(retained, ['social.craftsky.project.crochet.defs#sweater']);
    });
  });
}
