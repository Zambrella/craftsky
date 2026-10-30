import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_taxonomy.g.dart';

abstract final class ProjectOptionCatalogs {
  static const maxFilterValuesPerFamily = 10;
  static const feedDefsPrefix = 'social.craftsky.feed.defs';
  static const projectDefsPrefix = 'social.craftsky.project.defs';
  static const knittingDefsPrefix = 'social.craftsky.project.knitting.defs';
  static const crochetDefsPrefix = 'social.craftsky.project.crochet.defs';
  static const sewingDefsPrefix = 'social.craftsky.project.sewing.defs';
  static const quiltingDefsPrefix = 'social.craftsky.project.quilting.defs';

  static const knittingCraftToken = '$feedDefsPrefix#knitting';
  static const crochetCraftToken = '$feedDefsPrefix#crochet';
  static const sewingCraftToken = '$feedDefsPrefix#sewing';
  static const embroideryCraftToken = '$feedDefsPrefix#embroidery';
  static const quiltingCraftToken = '$feedDefsPrefix#quilting';
  static const finishedStatusToken = '$feedDefsPrefix#finished';
  static const wipStatusToken = '$feedDefsPrefix#wip';

  /// Canonical craft-type tokens used by Flutter on `/v1/*` AppView wires.
  ///
  /// AppView accepts supported bare aliases as compatibility inputs, but
  /// Flutter should prefer these full lexicon tokens for requests and expect
  /// them in responses.
  static const defaultSupportedCraftTokens = <String>[
    knittingCraftToken,
    crochetCraftToken,
    sewingCraftToken,
    embroideryCraftToken,
    quiltingCraftToken,
  ];

  static const knittingCraftFilterToken = CraftTypeFilterToken(
    knittingCraftToken,
  );
  static const crochetCraftFilterToken = CraftTypeFilterToken(
    crochetCraftToken,
  );
  static const garmentProjectTypeFilterToken = ProjectTypeFilterToken(
    '$projectDefsPrefix#garment',
  );
  static const beginnerPatternDifficultyFilterToken =
      PatternDifficultyFilterToken('$feedDefsPrefix#beginner');
  static const stripesDesignTagFilterToken = DesignTagFilterToken(
    '$projectDefsPrefix#stripes',
  );

  static final List<ProjectOption> craftTypes = ProjectTaxonomy.options(
    'craft',
  );

  static List<ProjectOption> craftTypesForPreferences(Iterable<String> crafts) {
    final byId = {
      for (final option in craftTypes) option.value.split('#').last: option,
    };
    final preferred = [
      for (final id in crafts.toSet()) ?byId[id],
    ];
    return [
      ...preferred,
      ...craftTypes.where((option) => !preferred.contains(option)),
    ];
  }

  static final List<ProjectOption> statuses = ProjectTaxonomy.options('status');

  static final List<ProjectOption> patternDifficulties =
      ProjectTaxonomy.options('difficulty');

  static final List<ProjectOption> projectTypes = ProjectTaxonomy.options(
    'projectType',
  );

  static final List<ProjectOption> yarnWeights = ProjectTaxonomy.options(
    'yarnWeight',
  );

  static final List<ProjectOption> needleSizes = <ProjectOption>[
    for (final value in [
      '0.5mm',
      '0.75mm',
      '1.0mm',
      '1.25mm',
      '1.5mm',
      '2.0mm',
      '2.25mm',
      '2.5mm',
      '2.75mm',
      '3.0mm',
      '3.25mm',
      '3.5mm',
      '3.75mm',
      '4.0mm',
      '4.25mm',
      '4.5mm',
      '4.75mm',
      '5.0mm',
      '5.5mm',
      '6.0mm',
      '6.5mm',
      '7.0mm',
      '7.5mm',
      '8.0mm',
      '9.0mm',
      '10.0mm',
      '12.0mm',
      '15.0mm',
      '19.0mm',
      '25.0mm',
    ])
      ProjectOption(value: value, label: value),
  ];

  static final List<ProjectOption> hookSizes = <ProjectOption>[
    for (final value in [
      '0.6mm',
      '0.7mm',
      '0.75mm',
      '0.85mm',
      '0.9mm',
      '1.0mm',
      '1.05mm',
      '1.1mm',
      '1.15mm',
      '1.25mm',
      '1.3mm',
      '1.4mm',
      '1.5mm',
      '1.65mm',
      '1.75mm',
      '1.8mm',
      '1.9mm',
      '2.0mm',
      '2.1mm',
      '2.25mm',
      '2.35mm',
      '2.5mm',
      '2.75mm',
      '3.0mm',
      '3.25mm',
      '3.5mm',
      '3.75mm',
      '4.0mm',
      '4.25mm',
      '4.5mm',
      '5.0mm',
      '5.5mm',
      '6.0mm',
      '6.5mm',
      '7.0mm',
      '7.5mm',
      '8.0mm',
      '9.0mm',
      '10.0mm',
      '11.5mm',
      '12.0mm',
      '15.0mm',
      '15.75mm',
      '19.0mm',
      '25.0mm',
      '40.0mm',
    ])
      ProjectOption(value: value, label: value),
  ];

  static const gaugeUnits = <ProjectOption>[
    ProjectOption(value: 'cm', label: 'cm'),
    ProjectOption(value: 'in', label: 'in'),
  ];

  static final List<ProjectOption> colours = ProjectTaxonomy.options('color');

  static final List<ProjectOption> designTags = ProjectTaxonomy.options(
    'designTag',
  );

  static const quiltingPiecingTechniques = <ProjectOption>[
    ProjectOption(
      value: '$quiltingDefsPrefix#traditional',
      label: 'Traditional',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#foundationPaperPiecing',
      label: 'Foundation paper piecing',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#englishPaperPiecing',
      label: 'English paper piecing',
    ),
    ProjectOption(value: '$quiltingDefsPrefix#improv', label: 'Improv'),
    ProjectOption(value: '$quiltingDefsPrefix#applique', label: 'Appliqué'),
  ];

  static final List<ProjectOption> quiltingMethods = ProjectTaxonomy.options(
    'quiltingMethod',
  );

  static List<ProjectOption> projectTypesForCraft(String craftToken) {
    final allowed = switch (craftToken) {
      quiltingCraftToken => const {
        '$projectDefsPrefix#quilt',
        '$projectDefsPrefix#homeGoods',
        '$projectDefsPrefix#accessory',
        '$projectDefsPrefix#garment',
        '$projectDefsPrefix#toyHobby',
        '$projectDefsPrefix#pet',
        '$projectDefsPrefix#component',
        '$projectDefsPrefix#other',
      },
      sewingCraftToken => const {
        '$projectDefsPrefix#garment',
        '$projectDefsPrefix#accessory',
        '$projectDefsPrefix#homeGoods',
        '$projectDefsPrefix#toyHobby',
        '$projectDefsPrefix#pet',
        '$projectDefsPrefix#medical',
        '$projectDefsPrefix#component',
        '$projectDefsPrefix#alteration',
        '$projectDefsPrefix#costume',
        '$projectDefsPrefix#other',
      },
      knittingCraftToken || crochetCraftToken => const {
        '$projectDefsPrefix#garment',
        '$projectDefsPrefix#accessory',
        '$projectDefsPrefix#homeGoods',
        '$projectDefsPrefix#toyHobby',
        '$projectDefsPrefix#pet',
        '$projectDefsPrefix#medical',
        '$projectDefsPrefix#component',
        '$projectDefsPrefix#costume',
        '$projectDefsPrefix#other',
      },
      _ => const <String>{},
    };
    return projectTypes
        .where((option) => allowed.contains(option.value))
        .toList();
  }

  static List<ProjectOption> projectSubtypesFor({
    required String craftToken,
    required String? projectTypeToken,
  }) {
    if (projectTypeToken == null || projectTypeToken.isEmpty) {
      return const <ProjectOption>[];
    }
    return projectSubtypesForCraft(
      craftToken,
    ).where((option) => option.parentValue == projectTypeToken).toList();
  }

  static List<ProjectOption> projectSubtypesForTypes({
    required String craftToken,
    required Iterable<String> projectTypeTokens,
  }) {
    final parents = projectTypeTokens.toSet();
    return projectSubtypesForCraft(
      craftToken,
    ).where((option) => parents.contains(option.parentValue)).toList();
  }

  static List<String> retainValidSubtypes({
    required String craftToken,
    required Iterable<String> projectTypeTokens,
    required Iterable<String> subtypeTokens,
  }) {
    final valid = projectSubtypesForTypes(
      craftToken: craftToken,
      projectTypeTokens: projectTypeTokens,
    ).map((option) => option.value).toSet();
    return subtypeTokens.where(valid.contains).toList();
  }

  static bool isSubtypeSelectionEnabled({
    required String craftToken,
    required String? projectTypeToken,
  }) {
    return projectSubtypesFor(
      craftToken: craftToken,
      projectTypeToken: projectTypeToken,
    ).isNotEmpty;
  }

  static List<ProjectOption> projectSubtypesForCraft(String craftToken) {
    return switch (craftToken) {
      sewingCraftToken => sewingProjectSubtypes,
      knittingCraftToken => knittingProjectSubtypes,
      crochetCraftToken => crochetProjectSubtypes,
      quiltingCraftToken => quiltingProjectSubtypes,
      _ => const <ProjectOption>[],
    };
  }

  static String? clearInvalidSubtype({
    required String craftToken,
    required String? projectTypeToken,
    required String? subtypeToken,
  }) {
    if (subtypeToken == null) return null;
    final valid = projectSubtypesFor(
      craftToken: craftToken,
      projectTypeToken: projectTypeToken,
    ).any((option) => option.value == subtypeToken);
    return valid ? subtypeToken : null;
  }

  static final List<ProjectOption> sewingProjectSubtypes =
      ProjectTaxonomy.options(
        'projectSubtype',
        craft: sewingCraftToken,
      );

  static final List<ProjectOption> knittingProjectSubtypes =
      ProjectTaxonomy.options(
        'projectSubtype',
        craft: knittingCraftToken,
      );

  static final List<ProjectOption> crochetProjectSubtypes =
      ProjectTaxonomy.options(
        'projectSubtype',
        craft: crochetCraftToken,
      );

  static final List<ProjectOption> quiltingProjectSubtypes =
      ProjectTaxonomy.options(
        'projectSubtype',
        craft: quiltingCraftToken,
      );
}
