import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_option.dart';

abstract final class ProjectOptionCatalogs {
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

  static const craftTypes = <ProjectOption>[
    ProjectOption(value: knittingCraftToken, label: 'Knitting'),
    ProjectOption(value: crochetCraftToken, label: 'Crochet'),
    ProjectOption(value: sewingCraftToken, label: 'Sewing'),
    ProjectOption(value: embroideryCraftToken, label: 'Embroidery'),
    ProjectOption(value: quiltingCraftToken, label: 'Quilting'),
  ];

  static const statuses = <ProjectOption>[
    ProjectOption(value: finishedStatusToken, label: 'Finished'),
    ProjectOption(value: wipStatusToken, label: 'Work in progress'),
  ];

  static const patternDifficulties = <ProjectOption>[
    ProjectOption(value: '$feedDefsPrefix#beginner', label: 'Beginner'),
    ProjectOption(value: '$feedDefsPrefix#intermediate', label: 'Intermediate'),
    ProjectOption(value: '$feedDefsPrefix#advanced', label: 'Advanced'),
    ProjectOption(value: '$feedDefsPrefix#expert', label: 'Expert'),
  ];

  static const projectTypes = <ProjectOption>[
    ProjectOption(value: '$projectDefsPrefix#garment', label: 'Garment'),
    ProjectOption(value: '$projectDefsPrefix#accessory', label: 'Accessory'),
    ProjectOption(value: '$projectDefsPrefix#homeGoods', label: 'Home goods'),
    ProjectOption(value: '$projectDefsPrefix#toyHobby', label: 'Toy or hobby'),
    ProjectOption(value: '$projectDefsPrefix#pet', label: 'Pet'),
    ProjectOption(
      value: '$projectDefsPrefix#medical',
      label: 'Medical or adaptive',
    ),
    ProjectOption(value: '$projectDefsPrefix#component', label: 'Component'),
    ProjectOption(value: '$projectDefsPrefix#quilt', label: 'Quilt'),
    ProjectOption(value: '$projectDefsPrefix#alteration', label: 'Alteration'),
    ProjectOption(value: '$projectDefsPrefix#costume', label: 'Costume'),
    ProjectOption(value: '$projectDefsPrefix#other', label: 'Other'),
  ];

  static const yarnWeights = <ProjectOption>[
    ProjectOption(value: '$projectDefsPrefix#lace', label: 'Lace'),
    ProjectOption(value: '$projectDefsPrefix#fingering', label: 'Fingering'),
    ProjectOption(value: '$projectDefsPrefix#sport', label: 'Sport'),
    ProjectOption(value: '$projectDefsPrefix#dk', label: 'DK'),
    ProjectOption(value: '$projectDefsPrefix#worsted', label: 'Worsted'),
    ProjectOption(value: '$projectDefsPrefix#aran', label: 'Aran'),
    ProjectOption(value: '$projectDefsPrefix#bulky', label: 'Bulky'),
    ProjectOption(value: '$projectDefsPrefix#superBulky', label: 'Super bulky'),
  ];

  static final needleSizes = <ProjectOption>[
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

  static final hookSizes = <ProjectOption>[
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

  static const colours = <ProjectOption>[
    ProjectOption(value: 'black', label: 'Black'),
    ProjectOption(value: 'white', label: 'White'),
    ProjectOption(value: 'gray', label: 'Grey'),
    ProjectOption(value: 'brown', label: 'Brown'),
    ProjectOption(value: 'beige', label: 'Beige'),
    ProjectOption(value: 'red', label: 'Red'),
    ProjectOption(value: 'orange', label: 'Orange'),
    ProjectOption(value: 'yellow', label: 'Yellow'),
    ProjectOption(value: 'green', label: 'Green'),
    ProjectOption(value: 'blue', label: 'Blue'),
    ProjectOption(value: 'purple', label: 'Purple'),
    ProjectOption(value: 'pink', label: 'Pink'),
    ProjectOption(value: 'cream', label: 'Cream'),
    ProjectOption(value: 'gold', label: 'Gold'),
    ProjectOption(value: 'silver', label: 'Silver'),
    ProjectOption(value: 'multicolor', label: 'Multicolour'),
    ProjectOption(value: 'natural', label: 'Natural'),
  ];

  static const designTags = <ProjectOption>[
    ProjectOption(value: '$projectDefsPrefix#floral', label: 'Floral'),
    ProjectOption(value: '$projectDefsPrefix#botanical', label: 'Botanical'),
    ProjectOption(value: '$projectDefsPrefix#animal', label: 'Animal'),
    ProjectOption(value: '$projectDefsPrefix#geometric', label: 'Geometric'),
    ProjectOption(value: '$projectDefsPrefix#abstract', label: 'Abstract'),
    ProjectOption(value: '$projectDefsPrefix#stripes', label: 'Stripes'),
    ProjectOption(
      value: '$projectDefsPrefix#colorblock',
      label: 'Colour block',
    ),
    ProjectOption(value: '$projectDefsPrefix#plaid', label: 'Plaid'),
    ProjectOption(
      value: '$projectDefsPrefix#checkerboard',
      label: 'Checkerboard',
    ),
    ProjectOption(value: '$projectDefsPrefix#lettering', label: 'Lettering'),
    ProjectOption(value: '$projectDefsPrefix#novelty', label: 'Novelty'),
    ProjectOption(value: '$projectDefsPrefix#holiday', label: 'Holiday'),
    ProjectOption(value: '$projectDefsPrefix#seasonal', label: 'Seasonal'),
    ProjectOption(
      value: '$projectDefsPrefix#traditional',
      label: 'Traditional',
    ),
    ProjectOption(value: '$projectDefsPrefix#modern', label: 'Modern'),
    ProjectOption(value: '$projectDefsPrefix#vintage', label: 'Vintage'),
    ProjectOption(value: '$projectDefsPrefix#minimalist', label: 'Minimalist'),
    ProjectOption(value: '$projectDefsPrefix#maximalist', label: 'Maximalist'),
    ProjectOption(value: '$projectDefsPrefix#whimsical', label: 'Whimsical'),
    ProjectOption(value: '$projectDefsPrefix#folk', label: 'Folk'),
    ProjectOption(value: '$projectDefsPrefix#boho', label: 'Boho'),
    ProjectOption(
      value: '$projectDefsPrefix#cottagecore',
      label: 'Cottagecore',
    ),
    ProjectOption(value: '$projectDefsPrefix#gothic', label: 'Gothic'),
    ProjectOption(value: '$projectDefsPrefix#romantic', label: 'Romantic'),
    ProjectOption(value: '$projectDefsPrefix#nautical', label: 'Nautical'),
    ProjectOption(value: '$projectDefsPrefix#sporty', label: 'Sporty'),
  ];

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

  static const quiltingMethods = <ProjectOption>[
    ProjectOption(
      value: '$quiltingDefsPrefix#handQuilted',
      label: 'Hand quilted',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#machineQuilted',
      label: 'Machine quilted',
    ),
    ProjectOption(value: '$quiltingDefsPrefix#longarm', label: 'Longarm'),
    ProjectOption(value: '$quiltingDefsPrefix#tied', label: 'Tied'),
  ];

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

  static const sewingProjectSubtypes = <ProjectOption>[
    ProjectOption(
      value: '$sewingDefsPrefix#dress',
      label: 'Dress',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#coatJacket',
      label: 'Coat or jacket',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#top',
      label: 'Top',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#bag',
      label: 'Bag',
      parentValue: '$projectDefsPrefix#accessory',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#blanket',
      label: 'Blanket',
      parentValue: '$projectDefsPrefix#homeGoods',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#softToy',
      label: 'Soft toy',
      parentValue: '$projectDefsPrefix#toyHobby',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#petClothing',
      label: 'Pet clothing',
      parentValue: '$projectDefsPrefix#pet',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#mask',
      label: 'Mask',
      parentValue: '$projectDefsPrefix#medical',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#patch',
      label: 'Patch',
      parentValue: '$projectDefsPrefix#component',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#mending',
      label: 'Mending',
      parentValue: '$projectDefsPrefix#alteration',
    ),
    ProjectOption(
      value: '$sewingDefsPrefix#costumePiece',
      label: 'Costume piece',
      parentValue: '$projectDefsPrefix#costume',
    ),
  ];

  static const knittingProjectSubtypes = <ProjectOption>[
    ProjectOption(
      value: '$knittingDefsPrefix#sweater',
      label: 'Sweater',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#cardigan',
      label: 'Cardigan',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#hat',
      label: 'Hat',
      parentValue: '$projectDefsPrefix#accessory',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#blanket',
      label: 'Blanket',
      parentValue: '$projectDefsPrefix#homeGoods',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#softToy',
      label: 'Soft toy',
      parentValue: '$projectDefsPrefix#toyHobby',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#petSweater',
      label: 'Pet sweater',
      parentValue: '$projectDefsPrefix#pet',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#mask',
      label: 'Mask',
      parentValue: '$projectDefsPrefix#medical',
    ),
    ProjectOption(
      value: '$knittingDefsPrefix#swatch',
      label: 'Swatch',
      parentValue: '$projectDefsPrefix#component',
    ),
  ];

  static const crochetProjectSubtypes = <ProjectOption>[
    ProjectOption(
      value: '$crochetDefsPrefix#sweater',
      label: 'Sweater',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$crochetDefsPrefix#bag',
      label: 'Bag',
      parentValue: '$projectDefsPrefix#accessory',
    ),
    ProjectOption(
      value: '$crochetDefsPrefix#blanket',
      label: 'Blanket',
      parentValue: '$projectDefsPrefix#homeGoods',
    ),
    ProjectOption(
      value: '$crochetDefsPrefix#amigurumi',
      label: 'Amigurumi',
      parentValue: '$projectDefsPrefix#toyHobby',
    ),
    ProjectOption(
      value: '$crochetDefsPrefix#petClothing',
      label: 'Pet clothing',
      parentValue: '$projectDefsPrefix#pet',
    ),
    ProjectOption(
      value: '$crochetDefsPrefix#mask',
      label: 'Mask',
      parentValue: '$projectDefsPrefix#medical',
    ),
    ProjectOption(
      value: '$crochetDefsPrefix#grannySquare',
      label: 'Granny square',
      parentValue: '$projectDefsPrefix#component',
    ),
  ];

  static const quiltingProjectSubtypes = <ProjectOption>[
    ProjectOption(
      value: '$quiltingDefsPrefix#throwQuilt',
      label: 'Throw quilt',
      parentValue: '$projectDefsPrefix#quilt',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#bedQuilt',
      label: 'Bed quilt',
      parentValue: '$projectDefsPrefix#quilt',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#tableRunner',
      label: 'Table runner',
      parentValue: '$projectDefsPrefix#homeGoods',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#bag',
      label: 'Bag',
      parentValue: '$projectDefsPrefix#accessory',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#jacket',
      label: 'Jacket',
      parentValue: '$projectDefsPrefix#garment',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#softToy',
      label: 'Soft toy',
      parentValue: '$projectDefsPrefix#toyHobby',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#petBedding',
      label: 'Pet bedding',
      parentValue: '$projectDefsPrefix#pet',
    ),
    ProjectOption(
      value: '$quiltingDefsPrefix#quiltBlock',
      label: 'Quilt block',
      parentValue: '$projectDefsPrefix#component',
    ),
  ];
}
