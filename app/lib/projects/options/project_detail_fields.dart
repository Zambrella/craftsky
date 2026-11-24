import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/options/project_taxonomy.g.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';

enum ProjectDetailInput {
  text,
  decimal,
  count,
  choice,
  choices,
  selection,
  object,
  entries,
}

/// A first-party field backed by a property in a reviewed Project Lexicon.
class ProjectDetailField {
  const new(
    this.key,
    this.label, {
    this.input = ProjectDetailInput.text,
    this.family,
    this.options = const [],
    this.children = const [],
    this.entryKey,
    this.maximum = 10,
    this.limit = const RecordTextLimit(100, 500),
    this.required = false,
    this.method,
    this.explicitEmpty = false,
  });

  final String key;
  final String label;
  final ProjectDetailInput input;
  final String? family;
  final List<ProjectOption> options;
  final List<ProjectDetailField> children;
  final String? entryKey;
  final int maximum;
  final RecordTextLimit limit;
  final bool required;
  final String? method;
  final bool explicitEmpty;

  List<ProjectOption> choicesFor(String craft, {String? parent}) {
    final family = this.family;
    if (family != null &&
        ProjectTaxonomy.projectDetailOptions.containsKey(family)) {
      return ProjectTaxonomy.detailOptionsFor(family);
    }
    return family == null
        ? options
        : ProjectTaxonomy.options(family, craft: craft, parent: parent);
  }
}

abstract final class ProjectDetailFields {
  static String formName(String craft) => 'craftDetails:$craft';

  static String? typeForCraft(String craft) => ProjectTaxonomy.entries
      .where((entry) => entry.family == 'craft' && entry.id == craft)
      .firstOrNull
      ?.detailsType;

  static List<ProjectDetailField> authoringForCraft(String? craft) =>
      _authoringFields[craft] ?? const [];

  static List<ProjectDetailField> forCraft(String? craft) {
    final active = authoringForCraft(craft);
    return [
      ...active,
      for (final field in _fields[craft] ?? const <ProjectDetailField>[])
        if (!active.any((entry) => entry.key == field.key)) field,
    ];
  }

  static Object? authoringValue(
    String craft,
    String key,
    Map<Object?, Object?> details,
  ) {
    if (details.containsKey(key)) return details[key];
    if (key == 'methods' && craft == ProjectOptionCatalogs.feltingCraftToken) {
      final legacy = details['method'];
      if (legacy is String) {
        final known = ProjectTaxonomy.detailOptionsFor('feltingMethod')
            .where((option) => _matchesLegacyToken(legacy, option.value))
            .firstOrNull;
        return [known?.value ?? legacy];
      }
    }
    if (key == 'methods' && craft == ProjectOptionCatalogs.crochetCraftToken) {
      const tunisian =
          '${ProjectOptionCatalogs.crochetDefsPrefix}#tunisianCrochet';
      final legacy = details['techniques'];
      if (legacy is List &&
          legacy.whereType<Map<Object?, Object?>>().any(
            (entry) => _matchesLegacyToken(entry['technique'], tunisian),
          )) {
        return const [tunisian];
      }
    }
    return null;
  }

  static bool _matchesLegacyToken(Object? value, String token) {
    if (value is! String) return false;
    final saved = value.trim().toLowerCase();
    final canonical = token.toLowerCase();
    return saved == canonical || saved == canonical.split('#').last;
  }

  static const _authoringFields = <String, List<ProjectDetailField>>{
    ProjectOptionCatalogs.knittingCraftToken: [
      ProjectDetailField(
        'methods',
        'Knitting methods',
        input: ProjectDetailInput.choices,
        family: 'knittingMethod',
      ),
    ],
    ProjectOptionCatalogs.crochetCraftToken: [
      ProjectDetailField(
        'methods',
        'Crochet methods',
        input: ProjectDetailInput.choices,
        family: 'crochetMethod',
        explicitEmpty: true,
      ),
    ],
    ProjectOptionCatalogs.embroideryCraftToken: [
      ProjectDetailField(
        'method',
        'Embroidery method',
        input: ProjectDetailInput.choice,
        family: 'embroideryMethod',
      ),
      ProjectDetailField(
        'style',
        'Embroidery style',
        input: ProjectDetailInput.choice,
        family: 'embroideryStyle',
      ),
    ],
    ProjectOptionCatalogs.feltingCraftToken: [
      ProjectDetailField(
        'methods',
        'Felting methods',
        input: ProjectDetailInput.choices,
        family: 'feltingMethod',
        explicitEmpty: true,
      ),
    ],
    ProjectOptionCatalogs.weavingCraftToken: [
      ProjectDetailField(
        'loomType',
        'Loom type',
        input: ProjectDetailInput.choice,
        family: 'loomType',
      ),
    ],
  };

  static const _decimal = RecordTextLimit(20, 20);

  static const _length = [
    ProjectOption(value: 'mm', label: 'mm'),
    ProjectOption(value: 'cm', label: 'cm'),
    ProjectOption(value: 'm', label: 'm'),
    ProjectOption(value: 'in', label: 'in'),
    ProjectOption(value: 'ft', label: 'ft'),
    ProjectOption(value: 'yd', label: 'yd'),
  ];

  static const _amount = ProjectDetailField(
    'amount',
    'Amount',
    input: ProjectDetailInput.decimal,
    required: true,
    limit: _decimal,
  );

  static const _lengthFields = <ProjectDetailField>[
    _amount,
    ProjectDetailField(
      'unit',
      'Unit',
      input: ProjectDetailInput.choice,
      options: _length,
      required: true,
    ),
  ];

  static const _fields = <String, List<ProjectDetailField>>{
    ProjectOptionCatalogs.knittingCraftToken: [
      ProjectDetailField(
        'methods',
        'Knitting methods',
        input: ProjectDetailInput.choices,
        family: 'method',
      ),
      ProjectDetailField(
        'machineModel',
        'Machine model',
        method: ProjectOptionCatalogs.knittingMachineMethodToken,
      ),
      ProjectDetailField(
        'machineGaugeMm',
        'Machine gauge (mm)',
        method: ProjectOptionCatalogs.knittingMachineMethodToken,
        input: ProjectDetailInput.decimal,
        limit: _decimal,
      ),
      ProjectDetailField(
        'techniques',
        'Techniques',
        input: ProjectDetailInput.selection,
        family: 'technique',
        entryKey: 'technique',
        maximum: 20,
        children: [
          ProjectDetailField(
            'subtypes',
            'Subtypes',
            input: ProjectDetailInput.selection,
            family: 'techniqueSubtype',
            entryKey: 'subtype',
          ),
        ],
      ),
    ],
    ProjectOptionCatalogs.crochetCraftToken: [
      ProjectDetailField(
        'techniques',
        'Techniques',
        input: ProjectDetailInput.selection,
        family: 'technique',
        entryKey: 'technique',
        maximum: 20,
      ),
    ],
    ProjectOptionCatalogs.sewingCraftToken: [
      ProjectDetailField(
        'techniques',
        'Techniques',
        input: ProjectDetailInput.selection,
        family: 'technique',
        entryKey: 'technique',
        maximum: 20,
      ),
    ],
    ProjectOptionCatalogs.embroideryCraftToken: [
      ProjectDetailField(
        'methods',
        'Embroidery methods',
        input: ProjectDetailInput.selection,
        family: 'method',
        entryKey: 'method',
        children: [
          ProjectDetailField(
            'styles',
            'Styles',
            input: ProjectDetailInput.selection,
            family: 'style',
            entryKey: 'style',
          ),
        ],
      ),
    ],
    ProjectOptionCatalogs.spinningCraftToken: [
      ProjectDetailField(
        'tool',
        'Tool',
        input: ProjectDetailInput.choice,
        family: 'tool',
      ),
      ProjectDetailField(
        'fibrePreparation',
        'Fibre preparation',
        input: ProjectDetailInput.choice,
        family: 'fibrePreparation',
      ),
      ProjectDetailField(
        'draftingTechniques',
        'Drafting techniques',
        input: ProjectDetailInput.selection,
        family: 'draftingTechnique',
        entryKey: 'technique',
      ),
      ProjectDetailField(
        'plyingTechniques',
        'Plying techniques',
        input: ProjectDetailInput.selection,
        family: 'plyingTechnique',
        entryKey: 'technique',
      ),
      ProjectDetailField(
        'plyCount',
        'Ply count',
        input: ProjectDetailInput.count,
      ),
      ProjectDetailField(
        'finishedYarn',
        'Finished yarn',
        input: ProjectDetailInput.object,
        children: [
          ProjectDetailField(
            'mass',
            'Mass',
            input: ProjectDetailInput.object,
            children: [
              _amount,
              ProjectDetailField(
                'unit',
                'Unit',
                input: ProjectDetailInput.choice,
                options: [
                  ProjectOption(value: 'g', label: 'g'),
                  ProjectOption(value: 'kg', label: 'kg'),
                  ProjectOption(value: 'oz', label: 'oz'),
                  ProjectOption(value: 'lb', label: 'lb'),
                ],
                required: true,
              ),
            ],
          ),
          ProjectDetailField(
            'length',
            'Length',
            input: ProjectDetailInput.object,
            children: _lengthFields,
          ),
          ProjectDetailField(
            'wrapsPerInch',
            'Wraps per inch',
            input: ProjectDetailInput.count,
          ),
        ],
      ),
    ],
    ProjectOptionCatalogs.dyeingCraftToken: [
      ProjectDetailField(
        'materialForm',
        'What was dyed',
        input: ProjectDetailInput.choice,
        family: 'materialForm',
      ),
      ProjectDetailField(
        'dyeType',
        'Dye type',
        input: ProjectDetailInput.choice,
        family: 'dyeType',
      ),
      ProjectDetailField(
        'applicationMethod',
        'Application method',
        input: ProjectDetailInput.choice,
        family: 'applicationMethod',
      ),
      ProjectDetailField(
        'colorEffect',
        'Colour effect',
        input: ProjectDetailInput.choice,
        family: 'colorEffect',
      ),
      ProjectDetailField(
        'recipeNotes',
        'Recipe notes',
        limit: RecordTextLimits.fitNotes,
      ),
    ],
    ProjectOptionCatalogs.punchNeedleCraftToken: [
      ProjectDetailField(
        'projectType',
        'Project type',
        input: ProjectDetailInput.choice,
        family: 'projectType',
      ),
      ProjectDetailField('foundationFabric', 'Foundation fabric'),
      ProjectDetailField('needle', 'Needle model or size'),
      ProjectDetailField(
        'loopHeight',
        'Loop height',
        input: ProjectDetailInput.object,
        children: _lengthFields,
      ),
    ],
    ProjectOptionCatalogs.macrameCraftToken: [
      ProjectDetailField(
        'cords',
        'Cords',
        input: ProjectDetailInput.entries,
        children: [
          ProjectDetailField('material', 'Cord material'),
          ProjectDetailField(
            'construction',
            'Cord construction',
            input: ProjectDetailInput.choice,
            family: 'construction',
          ),
          ProjectDetailField(
            'diameterMm',
            'Cord diameter (mm)',
            input: ProjectDetailInput.decimal,
            limit: _decimal,
          ),
        ],
      ),
      ProjectDetailField(
        'knots',
        'Knots',
        input: ProjectDetailInput.selection,
        family: 'knot',
        entryKey: 'knot',
      ),
      ProjectDetailField('finishedSize', 'Finished dimensions'),
    ],
    ProjectOptionCatalogs.weavingCraftToken: [
      ProjectDetailField(
        'loomType',
        'Loom type',
        input: ProjectDetailInput.choice,
        family: 'loomType',
      ),
      ProjectDetailField(
        'structure',
        'Weave structure',
        input: ProjectDetailInput.choice,
        family: 'structure',
      ),
      ProjectDetailField(
        'materials',
        'Warp and weft materials',
        input: ProjectDetailInput.entries,
        children: [
          ProjectDetailField(
            'role',
            'Role',
            input: ProjectDetailInput.choice,
            family: 'materialRole',
            required: true,
          ),
          ProjectDetailField(
            'text',
            'Material',
            limit: RecordTextLimits.material,
            required: true,
          ),
        ],
      ),
      ProjectDetailField(
        'sett',
        'Sett',
        input: ProjectDetailInput.object,
        children: [
          _amount,
          ProjectDetailField(
            'unit',
            'Unit',
            input: ProjectDetailInput.choice,
            required: true,
            options: [
              ProjectOption(value: 'ends-per-inch', label: 'Ends per inch'),
              ProjectOption(
                value: 'ends-per-centimetre',
                label: 'Ends per centimetre',
              ),
            ],
          ),
        ],
      ),
      ProjectDetailField('finishedSize', 'Finished dimensions'),
    ],
    ProjectOptionCatalogs.feltingCraftToken: [
      ProjectDetailField(
        'method',
        'Felting method',
        input: ProjectDetailInput.choice,
        family: 'method',
      ),
      ProjectDetailField(
        'form',
        'Project form',
        input: ProjectDetailInput.choice,
        family: 'form',
      ),
      ProjectDetailField(
        'needleGauges',
        'Needle gauges',
        input: ProjectDetailInput.selection,
        family: 'needleGauge',
        method: ProjectOptionCatalogs.needleFeltingMethodToken,
        entryKey: 'gauge',
      ),
      ProjectDetailField('core', 'Core'),
      ProjectDetailField('armature', 'Armature'),
    ],
  };
}
