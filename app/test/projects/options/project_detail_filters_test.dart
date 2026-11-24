import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/projects/options/project_detail_filters.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const knitting = ProjectOptionCatalogs.knittingCraftToken;
  const embroidery = ProjectOptionCatalogs.embroideryCraftToken;
  const colourwork = '${ProjectOptionCatalogs.knittingDefsPrefix}#colourwork';
  const cables = '${ProjectOptionCatalogs.knittingDefsPrefix}#cables';
  const fairIsle = '${ProjectOptionCatalogs.knittingDefsPrefix}#fairIsle';

  test('reviewed filters match applicable authoring dropdowns', () {
    final fields = {
      for (final craft in ProjectOptionCatalogs.defaultSupportedCraftTokens)
        craft: {
          for (final filter in ProjectDetailFilters.forCraft(craft))
            if (filter.allOptionsFor(craft).isNotEmpty) filter.family.name,
        },
    };
    expect(fields[knitting], {'knittingMethod'});
    expect(fields[ProjectOptionCatalogs.crochetCraftToken], {'crochetMethod'});
    expect(fields[embroidery], {'embroideryMethod', 'embroideryStyle'});
    expect(fields[ProjectOptionCatalogs.weavingCraftToken], {'loomType'});
    expect(fields[ProjectOptionCatalogs.feltingCraftToken], {'feltingMethod'});
    for (final craft in [
      ProjectOptionCatalogs.sewingCraftToken,
      ProjectOptionCatalogs.quiltingCraftToken,
      ProjectOptionCatalogs.spinningCraftToken,
      ProjectOptionCatalogs.dyeingCraftToken,
      ProjectOptionCatalogs.punchNeedleCraftToken,
      ProjectOptionCatalogs.macrameCraftToken,
    ]) {
      expect(fields[craft], isEmpty);
    }
  });

  test('SIM-AT-05 broad item filters match reviewed narrow values', () {
    const craft = ProjectOptionCatalogs.sewingCraftToken;
    const prefix = ProjectOptionCatalogs.sewingDefsPrefix;
    final project = ProjectMapper.fromMap({
      'common': {'craftType': craft},
      'details': {
        r'$type': sewingProjectDetailsType,
        'projectType': 'social.craftsky.project.defs#accessory',
        'projectSubtype': '$prefix#backpack',
      },
    });
    bool matches(String subtype) => ProjectDetailFilters.matchesEditedProject(
      project,
      ProjectBrowseQuery(
        craftTypes: [craft],
        filters: ProjectBrowseFilters(projectSubtype: [subtype]),
      ),
    );
    expect(matches('$prefix#bag'), isTrue);
    expect(matches('$prefix#backpack'), isTrue);
    expect(matches('$prefix#tote'), isFalse);
  });

  test('combined project item filters match a type or a subtype', () {
    const craft = ProjectOptionCatalogs.sewingCraftToken;
    const types = ProjectOptionCatalogs.projectDefsPrefix;
    const subtypes = ProjectOptionCatalogs.sewingDefsPrefix;
    bool matches(String? type, String? subtype) =>
        ProjectDetailFilters.matchesEditedProject(
          ProjectMapper.fromMap({
            'common': {'craftType': craft},
            'details': {
              r'$type': sewingProjectDetailsType,
              'projectType': ?type,
              'projectSubtype': ?subtype,
            },
          }),
          const ProjectBrowseQuery(
            craftTypes: [craft],
            filters: ProjectBrowseFilters(
              projectItem: ['$types#garment', '$subtypes#bag'],
            ),
          ),
        );
    expect(matches('$types#garment', null), isTrue);
    expect(matches('$types#garment', '$subtypes#dress'), isTrue);
    expect(matches('$types#accessory', '$subtypes#backpack'), isTrue);
    expect(matches('$types#accessory', '$subtypes#hat'), isFalse);
    expect(matches(null, '$subtypes#dress'), isFalse);
    expect(matches(null, '$subtypes#backpack'), isTrue);
  });

  test('combined dyeing item filters prefer the saved project type', () {
    const craft = ProjectOptionCatalogs.dyeingCraftToken;
    const query = ProjectBrowseQuery(
      craftTypes: [craft],
      filters: ProjectBrowseFilters(
        projectItem: ['social.craftsky.project.dyeing#unspunFibre'],
      ),
    );
    bool matches(Map<String, Object?> extra) =>
        ProjectDetailFilters.matchesEditedProject(
          ProjectMapper.fromMap({
            'common': {'craftType': craft},
            'details': {
              r'$type': 'social.craftsky.project.dyeing#details',
              'materialForm': 'fleece',
              ...extra,
            },
          }),
          query,
        );
    expect(matches({}), isTrue);
    expect(matches({'projectType': null}), isFalse);
    expect(
      matches({'projectType': 'social.craftsky.project.dyeing#fabric'}),
      isFalse,
    );
  });

  test('SIM-AT-05 explicit empty methods suppress legacy matches', () {
    const crochet = ProjectOptionCatalogs.crochetCraftToken;
    const tunisian =
        '${ProjectOptionCatalogs.crochetDefsPrefix}#tunisianCrochet';
    const felting = ProjectOptionCatalogs.feltingCraftToken;
    const needle = ProjectOptionCatalogs.needleFeltingMethodToken;
    for (final (craft, type, legacy, family, method) in [
      (
        crochet,
        crochetProjectDetailsType,
        {
          'techniques': [
            {'technique': 'TunisianCrochet'},
          ],
        },
        ProjectBrowseFilterFamily.crochetMethod,
        tunisian,
      ),
      (
        felting,
        'social.craftsky.project.felting#details',
        {'method': 'NEEDLE'},
        ProjectBrowseFilterFamily.feltingMethod,
        needle,
      ),
    ]) {
      final query = ProjectBrowseQuery(
        craftTypes: [craft],
        filters: const ProjectBrowseFilters().withValue(family, method),
      );
      bool matches(List<String>? methods) =>
          ProjectDetailFilters.matchesEditedProject(
            ProjectMapper.fromMap({
              'common': {'craftType': craft},
              'details': {r'$type': type, ...legacy, 'methods': ?methods},
            }),
            query,
          );
      expect(matches(null), isTrue, reason: craft);
      expect(matches([]), isFalse, reason: craft);
      expect(matches([method]), isTrue, reason: craft);
    }
  });

  test(
    'editor normalizes known legacy felting methods without losing others',
    () {
      const craft = ProjectOptionCatalogs.feltingCraftToken;
      expect(
        ProjectDetailFields.authoringValue(craft, 'methods', {
          'method': 'NEEDLE',
        }),
        [ProjectOptionCatalogs.needleFeltingMethodToken],
      );
      expect(
        ProjectDetailFields.authoringValue(craft, 'methods', {'method': 'wet'}),
        ['social.craftsky.project.felting#wet'],
      );
      expect(
        ProjectDetailFields.authoringValue(craft, 'methods', {
          'method': 'custom',
        }),
        ['custom'],
      );
    },
  );

  test('edited projects match a parent and subtype within one saved row', () {
    final filters = const ProjectBrowseFilters()
        .withValues(
          ProjectBrowseFilterFamily.technique,
          [colourwork],
        )
        .withValues(ProjectBrowseFilterFamily.techniqueSubtype, [fairIsle]);
    final query = ProjectBrowseQuery(craftTypes: [knitting], filters: filters);
    Project record(String technique) => ProjectMapper.fromMap({
      'common': {'craftType': knitting},
      'details': {
        r'$type': knittingProjectDetailsType,
        'techniques': [
          {'technique': colourwork},
          {
            'technique': technique,
            'subtypes': [
              {'subtype': fairIsle},
            ],
          },
        ],
      },
    });
    expect(
      ProjectDetailFilters.matchesEditedProject(record(cables), query),
      isFalse,
    );
    expect(
      ProjectDetailFilters.matchesEditedProject(record(colourwork), query),
      isTrue,
    );
  });

  test('edited projects use a legacy size only if the size list is absent', () {
    final query = ProjectBrowseQuery(
      craftTypes: [knitting],
      filters: const ProjectBrowseFilters().withValues(
        ProjectBrowseFilterFamily.needleSizeMm,
        ['4.0mm'],
      ),
    );
    Project record(List<String>? sizes) => ProjectMapper.fromMap({
      'common': {'craftType': knitting},
      'details': {
        r'$type': knittingProjectDetailsType,
        'needleSizeMm': '4.0mm',
        'needleSizesMm': ?sizes,
      },
    });
    expect(
      ProjectDetailFilters.matchesEditedProject(record(null), query),
      isTrue,
    );
    expect(
      ProjectDetailFilters.matchesEditedProject(record([]), query),
      isFalse,
    );
    expect(
      ProjectDetailFilters.matchesEditedProject(
        record(['4.0mm', '5.0mm']),
        query,
      ),
      isTrue,
    );
  });
}
