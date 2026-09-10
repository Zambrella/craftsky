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
    ProjectOption(
      value: '$feedDefsPrefix#confidentBeginner',
      label: 'Confident beginner',
    ),
    ProjectOption(value: '$feedDefsPrefix#intermediate', label: 'Intermediate'),
    ProjectOption(value: '$feedDefsPrefix#advanced', label: 'Advanced'),
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

  static ProjectOption _subtype(
    String prefix,
    String token,
    String label,
    String parent,
  ) => ProjectOption(
    value: '$prefix#$token',
    label: label,
    parentValue: '$projectDefsPrefix#$parent',
  );

  static final sewingProjectSubtypes = List<ProjectOption>.unmodifiable([
    _subtype(sewingDefsPrefix, 'coatJacket', 'Coat or jacket', 'garment'),
    _subtype(sewingDefsPrefix, 'dress', 'Dress', 'garment'),
    _subtype(sewingDefsPrefix, 'top', 'Top', 'garment'),
    _subtype(sewingDefsPrefix, 'shirt', 'Shirt', 'garment'),
    _subtype(sewingDefsPrefix, 'blouse', 'Blouse', 'garment'),
    _subtype(sewingDefsPrefix, 'tunic', 'Tunic', 'garment'),
    _subtype(sewingDefsPrefix, 'pants', 'Pants or trousers', 'garment'),
    _subtype(sewingDefsPrefix, 'shorts', 'Shorts', 'garment'),
    _subtype(sewingDefsPrefix, 'skirt', 'Skirt', 'garment'),
    _subtype(sewingDefsPrefix, 'leggings', 'Leggings', 'garment'),
    _subtype(sewingDefsPrefix, 'jumpsuit', 'Jumpsuit or romper', 'garment'),
    _subtype(sewingDefsPrefix, 'robe', 'Robe', 'garment'),
    _subtype(sewingDefsPrefix, 'sleepwear', 'Sleepwear', 'garment'),
    _subtype(sewingDefsPrefix, 'swimwear', 'Swimwear', 'garment'),
    _subtype(
      sewingDefsPrefix,
      'intimateApparel',
      'Intimate apparel',
      'garment',
    ),
    _subtype(sewingDefsPrefix, 'vest', 'Vest', 'garment'),
    _subtype(sewingDefsPrefix, 'babyOnesie', 'Baby onesie', 'garment'),
    _subtype(sewingDefsPrefix, 'shrugBolero', 'Shrug or bolero', 'garment'),
    _subtype(sewingDefsPrefix, 'bag', 'Bag', 'accessory'),
    _subtype(sewingDefsPrefix, 'tote', 'Tote bag', 'accessory'),
    _subtype(sewingDefsPrefix, 'pouch', 'Pouch or small case', 'accessory'),
    _subtype(sewingDefsPrefix, 'wallet', 'Wallet', 'accessory'),
    _subtype(sewingDefsPrefix, 'belt', 'Belt', 'accessory'),
    _subtype(sewingDefsPrefix, 'hat', 'Hat or cap', 'accessory'),
    _subtype(sewingDefsPrefix, 'headband', 'Headband', 'accessory'),
    _subtype(sewingDefsPrefix, 'scarf', 'Scarf', 'accessory'),
    _subtype(sewingDefsPrefix, 'jewelry', 'Jewellery', 'accessory'),
    _subtype(
      sewingDefsPrefix,
      'glovesMittens',
      'Gloves or mittens',
      'accessory',
    ),
    _subtype(sewingDefsPrefix, 'shoeCover', 'Shoe cover', 'accessory'),
    _subtype(sewingDefsPrefix, 'blanket', 'Blanket', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'bedding', 'Bedding', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'pillow', 'Pillow or cushion', 'homeGoods'),
    _subtype(
      sewingDefsPrefix,
      'curtain',
      'Curtain or window covering',
      'homeGoods',
    ),
    _subtype(sewingDefsPrefix, 'tableRunner', 'Table runner', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'placemat', 'Placemat', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'napkin', 'Napkin', 'homeGoods'),
    _subtype(
      sewingDefsPrefix,
      'potholder',
      'Potholder or hot pad',
      'homeGoods',
    ),
    _subtype(sewingDefsPrefix, 'coaster', 'Coaster', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'rug', 'Rug or mat', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'basket', 'Basket', 'homeGoods'),
    _subtype(
      sewingDefsPrefix,
      'container',
      'Container or storage item',
      'homeGoods',
    ),
    _subtype(sewingDefsPrefix, 'lampshade', 'Lampshade', 'homeGoods'),
    _subtype(sewingDefsPrefix, 'sachet', 'Sachet', 'homeGoods'),
    _subtype(
      sewingDefsPrefix,
      'decorative',
      'Decorative home item',
      'homeGoods',
    ),
    _subtype(sewingDefsPrefix, 'softToy', 'Soft toy', 'toyHobby'),
    _subtype(sewingDefsPrefix, 'dollClothes', 'Doll clothes', 'toyHobby'),
    _subtype(sewingDefsPrefix, 'puppet', 'Puppet', 'toyHobby'),
    _subtype(sewingDefsPrefix, 'costumePiece', 'Costume piece', 'costume'),
    _subtype(sewingDefsPrefix, 'playFood', 'Play food', 'toyHobby'),
    _subtype(sewingDefsPrefix, 'game', 'Game or game piece', 'toyHobby'),
    _subtype(sewingDefsPrefix, 'mobile', 'Mobile or hanging toy', 'toyHobby'),
    _subtype(
      sewingDefsPrefix,
      'craftSupply',
      'Craft supply or organiser',
      'toyHobby',
    ),
    _subtype(sewingDefsPrefix, 'petClothing', 'Pet clothing', 'pet'),
    _subtype(sewingDefsPrefix, 'petBedding', 'Pet bedding', 'pet'),
    _subtype(sewingDefsPrefix, 'petToy', 'Pet toy', 'pet'),
    _subtype(sewingDefsPrefix, 'petAccessory', 'Pet accessory', 'pet'),
    _subtype(sewingDefsPrefix, 'mask', 'Mask or face covering', 'medical'),
    _subtype(sewingDefsPrefix, 'heatColdPack', 'Heat or cold pack', 'medical'),
    _subtype(
      sewingDefsPrefix,
      'adaptiveClothing',
      'Adaptive clothing',
      'medical',
    ),
    _subtype(
      sewingDefsPrefix,
      'braceSupport',
      'Brace, support or wrap',
      'medical',
    ),
    _subtype(
      sewingDefsPrefix,
      'wheelchairAccessory',
      'Wheelchair accessory',
      'medical',
    ),
    _subtype(sewingDefsPrefix, 'patch', 'Patch', 'component'),
    _subtype(sewingDefsPrefix, 'applique', 'Applique', 'component'),
    _subtype(sewingDefsPrefix, 'pocket', 'Pocket', 'component'),
    _subtype(sewingDefsPrefix, 'strap', 'Strap', 'component'),
    _subtype(sewingDefsPrefix, 'lining', 'Lining', 'component'),
    _subtype(sewingDefsPrefix, 'trim', 'Trim', 'component'),
    _subtype(sewingDefsPrefix, 'mending', 'Mending', 'alteration'),
    _subtype(sewingDefsPrefix, 'refashion', 'Refashion', 'alteration'),
    _subtype(sewingDefsPrefix, 'hemming', 'Hemming', 'alteration'),
    _subtype(sewingDefsPrefix, 'repair', 'Repair', 'alteration'),
  ]);

  static final knittingProjectSubtypes = List<ProjectOption>.unmodifiable([
    _subtype(knittingDefsPrefix, 'sweater', 'Sweater', 'garment'),
    _subtype(knittingDefsPrefix, 'cardigan', 'Cardigan', 'garment'),
    _subtype(knittingDefsPrefix, 'pullover', 'Pullover', 'garment'),
    _subtype(knittingDefsPrefix, 'top', 'Top', 'garment'),
    _subtype(knittingDefsPrefix, 'tee', 'Tee', 'garment'),
    _subtype(knittingDefsPrefix, 'tank', 'Tank', 'garment'),
    _subtype(knittingDefsPrefix, 'dress', 'Dress', 'garment'),
    _subtype(knittingDefsPrefix, 'skirt', 'Skirt', 'garment'),
    _subtype(knittingDefsPrefix, 'vest', 'Vest', 'garment'),
    _subtype(knittingDefsPrefix, 'shrugBolero', 'Shrug or bolero', 'garment'),
    _subtype(knittingDefsPrefix, 'poncho', 'Poncho', 'garment'),
    _subtype(knittingDefsPrefix, 'coatJacket', 'Coat or jacket', 'garment'),
    _subtype(knittingDefsPrefix, 'babyOnesie', 'Baby onesie', 'garment'),
    _subtype(knittingDefsPrefix, 'hat', 'Hat', 'accessory'),
    _subtype(knittingDefsPrefix, 'beanie', 'Beanie', 'accessory'),
    _subtype(knittingDefsPrefix, 'headband', 'Headband', 'accessory'),
    _subtype(knittingDefsPrefix, 'scarf', 'Scarf', 'accessory'),
    _subtype(knittingDefsPrefix, 'cowl', 'Cowl', 'accessory'),
    _subtype(knittingDefsPrefix, 'shawlWrap', 'Shawl or wrap', 'accessory'),
    _subtype(
      knittingDefsPrefix,
      'neckTorso',
      'Neck or torso accessory',
      'accessory',
    ),
    _subtype(knittingDefsPrefix, 'mittens', 'Mittens', 'accessory'),
    _subtype(knittingDefsPrefix, 'gloves', 'Gloves', 'accessory'),
    _subtype(
      knittingDefsPrefix,
      'fingerlessMitts',
      'Fingerless mitts',
      'accessory',
    ),
    _subtype(knittingDefsPrefix, 'socks', 'Socks', 'accessory'),
    _subtype(knittingDefsPrefix, 'slippers', 'Slippers', 'accessory'),
    _subtype(knittingDefsPrefix, 'bag', 'Bag', 'accessory'),
    _subtype(knittingDefsPrefix, 'belt', 'Belt', 'accessory'),
    _subtype(knittingDefsPrefix, 'jewelry', 'Jewellery', 'accessory'),
    _subtype(knittingDefsPrefix, 'blanket', 'Blanket', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'afghan', 'Afghan', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'babyBlanket', 'Baby blanket', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'pillow', 'Pillow', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'rug', 'Rug', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'washcloth', 'Washcloth', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'dishcloth', 'Dishcloth', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'teaTowel', 'Tea towel', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'cozy', 'Cosy', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'decorative', 'Decorative item', 'homeGoods'),
    _subtype(knittingDefsPrefix, 'softToy', 'Soft toy', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'doll', 'Doll', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'dollClothes', 'Doll clothes', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'puppet', 'Puppet', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'mobile', 'Mobile', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'playFood', 'Play food', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'ball', 'Ball', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'game', 'Game', 'toyHobby'),
    _subtype(knittingDefsPrefix, 'petSweater', 'Pet sweater', 'pet'),
    _subtype(knittingDefsPrefix, 'petBedding', 'Pet bedding', 'pet'),
    _subtype(knittingDefsPrefix, 'petToy', 'Pet toy', 'pet'),
    _subtype(knittingDefsPrefix, 'petAccessory', 'Pet accessory', 'pet'),
    _subtype(knittingDefsPrefix, 'mask', 'Mask', 'medical'),
    _subtype(
      knittingDefsPrefix,
      'heatColdPack',
      'Heat or cold pack',
      'medical',
    ),
    _subtype(
      knittingDefsPrefix,
      'adaptiveClothing',
      'Adaptive clothing',
      'medical',
    ),
    _subtype(knittingDefsPrefix, 'compression', 'Compression item', 'medical'),
    _subtype(knittingDefsPrefix, 'braceSupport', 'Brace or support', 'medical'),
    _subtype(knittingDefsPrefix, 'swatch', 'Swatch', 'component'),
    _subtype(knittingDefsPrefix, 'square', 'Square', 'component'),
    _subtype(knittingDefsPrefix, 'motif', 'Motif', 'component'),
    _subtype(knittingDefsPrefix, 'patch', 'Patch', 'component'),
    _subtype(knittingDefsPrefix, 'trim', 'Trim', 'component'),
    _subtype(knittingDefsPrefix, 'cord', 'Cord', 'component'),
  ]);

  static final crochetProjectSubtypes = List<ProjectOption>.unmodifiable([
    _subtype(crochetDefsPrefix, 'sweater', 'Sweater', 'garment'),
    _subtype(crochetDefsPrefix, 'cardigan', 'Cardigan', 'garment'),
    _subtype(crochetDefsPrefix, 'pullover', 'Pullover', 'garment'),
    _subtype(crochetDefsPrefix, 'top', 'Top', 'garment'),
    _subtype(crochetDefsPrefix, 'tank', 'Tank', 'garment'),
    _subtype(crochetDefsPrefix, 'tee', 'Tee', 'garment'),
    _subtype(crochetDefsPrefix, 'dress', 'Dress', 'garment'),
    _subtype(crochetDefsPrefix, 'skirt', 'Skirt', 'garment'),
    _subtype(crochetDefsPrefix, 'vest', 'Vest', 'garment'),
    _subtype(crochetDefsPrefix, 'shrugBolero', 'Shrug or bolero', 'garment'),
    _subtype(crochetDefsPrefix, 'poncho', 'Poncho', 'garment'),
    _subtype(crochetDefsPrefix, 'swimwear', 'Swimwear', 'garment'),
    _subtype(crochetDefsPrefix, 'babyOnesie', 'Baby onesie', 'garment'),
    _subtype(crochetDefsPrefix, 'bag', 'Bag', 'accessory'),
    _subtype(crochetDefsPrefix, 'tote', 'Tote', 'accessory'),
    _subtype(crochetDefsPrefix, 'pouch', 'Pouch', 'accessory'),
    _subtype(crochetDefsPrefix, 'hat', 'Hat', 'accessory'),
    _subtype(crochetDefsPrefix, 'headband', 'Headband', 'accessory'),
    _subtype(crochetDefsPrefix, 'scarf', 'Scarf', 'accessory'),
    _subtype(crochetDefsPrefix, 'cowl', 'Cowl', 'accessory'),
    _subtype(crochetDefsPrefix, 'shawlWrap', 'Shawl or wrap', 'accessory'),
    _subtype(
      crochetDefsPrefix,
      'neckTorso',
      'Neck or torso accessory',
      'accessory',
    ),
    _subtype(
      crochetDefsPrefix,
      'glovesMittens',
      'Gloves or mittens',
      'accessory',
    ),
    _subtype(
      crochetDefsPrefix,
      'socksSlippers',
      'Socks or slippers',
      'accessory',
    ),
    _subtype(crochetDefsPrefix, 'belt', 'Belt', 'accessory'),
    _subtype(crochetDefsPrefix, 'jewelry', 'Jewellery', 'accessory'),
    _subtype(crochetDefsPrefix, 'blanket', 'Blanket', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'afghan', 'Afghan', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'babyBlanket', 'Baby blanket', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'pillow', 'Pillow', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'rug', 'Rug', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'basket', 'Basket', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'container', 'Container', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'coaster', 'Coaster', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'potholder', 'Potholder', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'doily', 'Doily', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'tableRunner', 'Table runner', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'curtain', 'Curtain', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'plantHanger', 'Plant hanger', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'cozy', 'Cosy', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'decorative', 'Decorative item', 'homeGoods'),
    _subtype(crochetDefsPrefix, 'amigurumi', 'Amigurumi', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'softToy', 'Soft toy', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'doll', 'Doll', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'dollClothes', 'Doll clothes', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'puppet', 'Puppet', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'mobile', 'Mobile', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'playFood', 'Play food', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'game', 'Game', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'ball', 'Ball', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'blocks', 'Blocks', 'toyHobby'),
    _subtype(crochetDefsPrefix, 'costumePiece', 'Costume piece', 'costume'),
    _subtype(crochetDefsPrefix, 'petClothing', 'Pet clothing', 'pet'),
    _subtype(crochetDefsPrefix, 'petBedding', 'Pet bedding', 'pet'),
    _subtype(crochetDefsPrefix, 'petToy', 'Pet toy', 'pet'),
    _subtype(crochetDefsPrefix, 'petAccessory', 'Pet accessory', 'pet'),
    _subtype(crochetDefsPrefix, 'mask', 'Mask', 'medical'),
    _subtype(crochetDefsPrefix, 'heatColdPack', 'Heat or cold pack', 'medical'),
    _subtype(
      crochetDefsPrefix,
      'adaptiveClothing',
      'Adaptive clothing',
      'medical',
    ),
    _subtype(crochetDefsPrefix, 'grannySquare', 'Granny square', 'component'),
    _subtype(crochetDefsPrefix, 'motif', 'Motif', 'component'),
    _subtype(crochetDefsPrefix, 'applique', 'Applique', 'component'),
    _subtype(crochetDefsPrefix, 'swatch', 'Swatch', 'component'),
    _subtype(crochetDefsPrefix, 'trim', 'Trim', 'component'),
    _subtype(crochetDefsPrefix, 'cord', 'Cord', 'component'),
    _subtype(crochetDefsPrefix, 'flower', 'Flower', 'component'),
  ]);

  static final quiltingProjectSubtypes = List<ProjectOption>.unmodifiable([
    _subtype(quiltingDefsPrefix, 'bedQuilt', 'Bed quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'babyQuilt', 'Baby quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'throwQuilt', 'Throw quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'wallHanging', 'Wall hanging', 'quilt'),
    _subtype(quiltingDefsPrefix, 'miniQuilt', 'Mini quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'artQuilt', 'Art quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'memoryQuilt', 'Memory quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'samplerQuilt', 'Sampler quilt', 'quilt'),
    _subtype(quiltingDefsPrefix, 'medallionQuilt', 'Medallion quilt', 'quilt'),
    _subtype(
      quiltingDefsPrefix,
      'wholeclothQuilt',
      'Wholecloth quilt',
      'quilt',
    ),
    _subtype(quiltingDefsPrefix, 'tableRunner', 'Table runner', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'placemat', 'Placemat', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'pillow', 'Pillow', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'potholder', 'Potholder', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'coaster', 'Coaster', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'rug', 'Rug', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'curtain', 'Curtain', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'bedding', 'Bedding', 'homeGoods'),
    _subtype(quiltingDefsPrefix, 'bag', 'Bag', 'accessory'),
    _subtype(quiltingDefsPrefix, 'tote', 'Tote', 'accessory'),
    _subtype(quiltingDefsPrefix, 'pouch', 'Pouch', 'accessory'),
    _subtype(quiltingDefsPrefix, 'jacket', 'Jacket', 'garment'),
    _subtype(quiltingDefsPrefix, 'vest', 'Vest', 'garment'),
    _subtype(quiltingDefsPrefix, 'softToy', 'Soft toy', 'toyHobby'),
    _subtype(quiltingDefsPrefix, 'playMat', 'Play mat', 'toyHobby'),
    _subtype(quiltingDefsPrefix, 'petBedding', 'Pet bedding', 'pet'),
    _subtype(quiltingDefsPrefix, 'quiltBlock', 'Quilt block', 'component'),
    _subtype(quiltingDefsPrefix, 'applique', 'Applique', 'component'),
    _subtype(quiltingDefsPrefix, 'patch', 'Patch', 'component'),
    _subtype(quiltingDefsPrefix, 'binding', 'Binding', 'component'),
    _subtype(quiltingDefsPrefix, 'quiltTop', 'Quilt top', 'component'),
    _subtype(quiltingDefsPrefix, 'sampleBlock', 'Sample block', 'component'),
  ]);
}
