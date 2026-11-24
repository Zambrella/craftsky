import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/options/project_taxonomy.g.dart';
import 'package:flutter/foundation.dart';

class ProjectDetailFilter {
  const new(
    this.family,
    this.path, {
    this.label,
    this.options,
    this.parent,
    this.legacyField,
  });

  final ProjectBrowseFilterFamily family;
  final List<String> path;
  final String? label;
  final List<ProjectOption>? options;
  final ProjectBrowseFilterFamily? parent;
  final String? legacyField;

  ProjectDetailField _editorField(String craft) {
    var fields = ProjectDetailFields.forCraft(craft);
    late ProjectDetailField field;
    for (final key in path) {
      field = fields.firstWhere((field) => field.key == key);
      fields = field.children;
    }
    return field;
  }

  String labelFor(String craft) => label ?? _editorField(craft).label;

  List<ProjectOption> allOptionsFor(String craft) =>
      options ?? _editorField(craft).choicesFor(craft);

  List<ProjectOption> optionsFor(String craft, ProjectBrowseFilters filters) {
    final parent = this.parent;
    if (parent == null) return allOptionsFor(craft);
    final selectedParents = filters.valuesFor(parent);
    if (selectedParents.isEmpty) return const [];
    final requiredMethod = options == null ? _editorField(craft).method : null;
    return allOptionsFor(craft).where((option) {
      final requiredParent = option.parentValue ?? requiredMethod;
      return requiredParent == null || selectedParents.contains(requiredParent);
    }).toList();
  }

  Iterable<String> _savedValues(
    String craft,
    Map<Object?, Object?> details, {
    bool withinGroup = false,
  }) {
    if (family == ProjectBrowseFilterFamily.embroideryMethod ||
        family == ProjectBrowseFilterFamily.embroideryStyle) {
      if (withinGroup || details.containsKey(path.first)) {
        return _valuesAtPath(details, path);
      }
      return _valuesAtPath(details, [
        'methods',
        if (family == ProjectBrowseFilterFamily.embroideryMethod)
          'method'
        else
          'styles',
      ]);
    }
    if (family == ProjectBrowseFilterFamily.crochetMethod ||
        family == ProjectBrowseFilterFamily.feltingMethod) {
      return _valuesAtPath(
        ProjectDetailFields.authoringValue(craft, path.first, details),
        const [],
      );
    }
    final legacy = legacyField;
    if (legacy != null && !details.containsKey(path.first)) {
      return _valuesAtPath(details, [legacy]);
    }
    final entryKey = options == null ? _editorField(craft).entryKey : null;
    return _valuesAtPath(details, [
      ...(withinGroup ? path.skip(1) : path),
      ?entryKey,
    ]);
  }
}

abstract final class ProjectDetailFilters {
  static List<ProjectDetailFilter> forCraft(String craft) =>
      ProjectOptionCatalogs.defaultSupportedCraftTokens.contains(craft)
      ? _authoringFilters[craft] ?? const []
      : const [];

  static Iterable<ProjectDetailFilter> _supportedFilters(String craft) sync* {
    final current = forCraft(craft);
    yield* current;
    for (final legacy in _filters[craft] ?? const <ProjectDetailFilter>[]) {
      if (!current.any((filter) => filter.family == legacy.family)) {
        yield legacy;
      }
    }
  }

  static ProjectBrowseFilters retainCompatibleSelections(
    String craft,
    ProjectBrowseFilters previous,
    ProjectBrowseFilters requested,
  ) {
    var next = requested;
    if (!listEquals(previous.projectType, next.projectType)) {
      next = next.withValues(
        ProjectBrowseFilterFamily.projectSubtype,
        ProjectOptionCatalogs.retainValidSubtypes(
          craftToken: craft,
          projectTypeTokens: next.projectType,
          subtypeTokens: next.projectSubtype,
        ),
      );
    }
    for (final filter in _supportedFilters(craft)) {
      final parent = filter.parent;
      if (parent == null ||
          listEquals(previous.valuesFor(parent), next.valuesFor(parent))) {
        continue;
      }
      final selected = next.valuesFor(filter.family);
      if (selected.isEmpty) continue;
      final allowed = filter
          .optionsFor(craft, next)
          .map((option) => option.value);
      final retained = selected.where(allowed.contains).toList();
      if (selected.length != retained.length) {
        next = next.withValues(filter.family, retained);
      }
    }
    return next;
  }

  /// An accepted edit may change a project already visible in this feed.
  /// Fresh results remain authoritative and do not need local filtering.
  static bool matchesEditedProject(Project? project, ProjectBrowseQuery query) {
    final filters = query.filters;
    final detailCount = ProjectBrowseFilterFamily.values
        .where(
          (family) => family.isDetail && filters.valuesFor(family).isNotEmpty,
        )
        .length;
    if (detailCount == 0 &&
        filters.projectType.isEmpty &&
        filters.projectSubtype.isEmpty &&
        filters.projectItem.isEmpty &&
        filters.knittingMethod.isEmpty &&
        filters.feltingMethod.isEmpty) {
      return true;
    }
    if (project == null) return false;
    final craft = project.common.craftType;
    if (query.craftTypes.isNotEmpty && !query.craftTypes.contains(craft)) {
      return false;
    }
    final active = {
      for (final filter in _supportedFilters(craft))
        if (filters.valuesFor(filter.family).isNotEmpty) filter.family: filter,
    };
    final details = project.toMap()['details'];
    if (details is! Map<Object?, Object?> ||
        details[r'$type'] != ProjectDetailFields.typeForCraft(craft) ||
        active.length !=
            detailCount +
                (filters.knittingMethod.isEmpty ? 0 : 1) +
                (filters.feltingMethod.isEmpty ? 0 : 1)) {
      return false;
    }
    final legacyDyeingItem = details['materialForm'];
    final type =
        craft == ProjectOptionCatalogs.dyeingCraftToken &&
            !details.containsKey('projectType') &&
            legacyDyeingItem is String
        ? _legacyDyeingItemId(legacyDyeingItem)
        : details['projectType'];
    final subtype = details['projectSubtype'];
    if (!_matchesItem(type, filters.projectType) ||
        !_matchesItem(subtype, filters.projectSubtype) ||
        !_matchesSelectedItem(type, subtype, filters.projectItem)) {
      return false;
    }
    if (craft == ProjectOptionCatalogs.embroideryCraftToken &&
        active.containsKey(ProjectBrowseFilterFamily.embroideryMethod) &&
        active.containsKey(ProjectBrowseFilterFamily.embroideryStyle) &&
        !details.containsKey('method') &&
        !details.containsKey('style')) {
      final methods = details['methods'];
      if (methods is! List ||
          !methods.whereType<Map<Object?, Object?>>().any(
            (entry) =>
                _overlaps(
                  _valuesAtPath(entry, ['method']),
                  filters.valuesFor(ProjectBrowseFilterFamily.embroideryMethod),
                ) &&
                _overlaps(
                  _valuesAtPath(entry, ['styles']),
                  filters.valuesFor(ProjectBrowseFilterFamily.embroideryStyle),
                ),
          )) {
        return false;
      }
    }
    for (final filter in active.values) {
      final selected = filters.valuesFor(filter.family);
      final parent = active[filter.parent];
      if (parent != null && parent.path.first == filter.path.first) {
        final entries = details[filter.path.first];
        if (entries is! List ||
            !entries.whereType<Map<Object?, Object?>>().any(
              (entry) =>
                  _overlaps(
                    parent._savedValues(craft, entry, withinGroup: true),
                    filters.valuesFor(parent.family),
                  ) &&
                  _overlaps(
                    filter._savedValues(craft, entry, withinGroup: true),
                    selected,
                  ),
            )) {
          return false;
        }
      } else if (!_overlaps(filter._savedValues(craft, details), selected)) {
        return false;
      }
    }
    return true;
  }

  static bool _matchesItem(Object? actual, List<String> selected) =>
      selected.isEmpty ||
      (actual is String &&
          selected.any(
            (value) => ProjectTaxonomy.browseMatches(actual, value),
          ));

  static bool _matchesSelectedItem(
    Object? type,
    Object? subtype,
    List<String> selected,
  ) {
    if (selected.isEmpty) return true;
    return selected.any((value) {
      final normalized = value.trim().toLowerCase();
      return ProjectTaxonomy.entries.any((entry) {
        if (entry.id.toLowerCase() != normalized) return false;
        if (entry.family == 'projectType') {
          return _matchesItem(type, [entry.id]) ||
              (type != null && _matchesItem(subtype, [entry.id]));
        }
        return entry.family == 'projectSubtype' &&
            _matchesItem(subtype, [entry.id]);
      });
    });
  }

  static String _legacyDyeingItemId(String value) {
    return RegExp(r'^[a-zA-Z0-9]+$').hasMatch(value)
        ? 'social.craftsky.project.dyeing#$value'
        : value;
  }

  static const _authoringFilters = <String, List<ProjectDetailFilter>>{
    ProjectOptionCatalogs.knittingCraftToken: [
      ProjectDetailFilter(ProjectBrowseFilterFamily.knittingMethod, [
        'methods',
      ]),
    ],
    ProjectOptionCatalogs.crochetCraftToken: [
      ProjectDetailFilter(ProjectBrowseFilterFamily.crochetMethod, ['methods']),
    ],
    ProjectOptionCatalogs.embroideryCraftToken: [
      ProjectDetailFilter(ProjectBrowseFilterFamily.embroideryMethod, [
        'method',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.embroideryStyle, ['style']),
    ],
    ProjectOptionCatalogs.weavingCraftToken: [
      ProjectDetailFilter(ProjectBrowseFilterFamily.loomType, ['loomType']),
    ],
    ProjectOptionCatalogs.feltingCraftToken: [
      ProjectDetailFilter(ProjectBrowseFilterFamily.feltingMethod, ['methods']),
    ],
  };

  static const _technique = ProjectDetailFilter(
    ProjectBrowseFilterFamily.technique,
    ['techniques'],
  );

  static const _gauge = ProjectDetailFilter(
    ProjectBrowseFilterFamily.gaugeUnit,
    ['gauge', 'unit'],
    label: 'Gauge unit',
    options: ProjectOptionCatalogs.gaugeUnits,
  );

  static final _filters = <String, List<ProjectDetailFilter>>{
    ProjectOptionCatalogs.knittingCraftToken: [
      const ProjectDetailFilter(
        ProjectBrowseFilterFamily.knittingMethod,
        ['methods'],
        label: 'Knitting method',
      ),
      const ProjectDetailFilter(
        ProjectBrowseFilterFamily.technique,
        ['techniques'],
      ),
      const ProjectDetailFilter(
        ProjectBrowseFilterFamily.techniqueSubtype,
        ['techniques', 'subtypes'],
        label: 'Technique subtypes',
        parent: ProjectBrowseFilterFamily.technique,
      ),
      ProjectDetailFilter(
        ProjectBrowseFilterFamily.needleSizeMm,
        ['needleSizesMm'],
        label: 'Needle sizes',
        options: ProjectOptionCatalogs.needleSizes,
        legacyField: 'needleSizeMm',
      ),
      _gauge,
    ],
    ProjectOptionCatalogs.crochetCraftToken: [
      _technique,
      ProjectDetailFilter(
        ProjectBrowseFilterFamily.hookSizeMm,
        ['hookSizesMm'],
        label: 'Hook sizes',
        options: ProjectOptionCatalogs.hookSizes,
        legacyField: 'hookSizeMm',
      ),
      _gauge,
    ],
    ProjectOptionCatalogs.sewingCraftToken: const [_technique],
    ProjectOptionCatalogs.embroideryCraftToken: const [
      ProjectDetailFilter(
        ProjectBrowseFilterFamily.embroideryMethod,
        ['methods'],
        label: 'Embroidery method',
      ),
      ProjectDetailFilter(
        ProjectBrowseFilterFamily.embroideryStyle,
        ['methods', 'styles'],
        label: 'Embroidery style',
        parent: ProjectBrowseFilterFamily.embroideryMethod,
      ),
    ],
    ProjectOptionCatalogs.spinningCraftToken: const [
      ProjectDetailFilter(ProjectBrowseFilterFamily.spinningTool, [
        'tool',
      ], label: 'Spinning tool'),
      ProjectDetailFilter(ProjectBrowseFilterFamily.fibrePreparation, [
        'fibrePreparation',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.draftingTechnique, [
        'draftingTechniques',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.plyingTechnique, [
        'plyingTechniques',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.finishedYarnMassUnit, [
        'finishedYarn',
        'mass',
        'unit',
      ], label: 'Finished yarn mass unit'),
      ProjectDetailFilter(ProjectBrowseFilterFamily.finishedYarnLengthUnit, [
        'finishedYarn',
        'length',
        'unit',
      ], label: 'Finished yarn length unit'),
    ],
    ProjectOptionCatalogs.dyeingCraftToken: const [
      ProjectDetailFilter(ProjectBrowseFilterFamily.materialForm, [
        'materialForm',
      ], label: 'Material form'),
      ProjectDetailFilter(ProjectBrowseFilterFamily.dyeType, ['dyeType']),
      ProjectDetailFilter(ProjectBrowseFilterFamily.applicationMethod, [
        'applicationMethod',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.colorEffect, [
        'colorEffect',
      ]),
    ],
    ProjectOptionCatalogs.punchNeedleCraftToken: const [
      ProjectDetailFilter(ProjectBrowseFilterFamily.loopHeightUnit, [
        'loopHeight',
        'unit',
      ], label: 'Loop height unit'),
    ],
    ProjectOptionCatalogs.macrameCraftToken: const [
      ProjectDetailFilter(ProjectBrowseFilterFamily.cordConstruction, [
        'cords',
        'construction',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.knot, ['knots']),
    ],
    ProjectOptionCatalogs.weavingCraftToken: const [
      ProjectDetailFilter(ProjectBrowseFilterFamily.loomType, ['loomType']),
      ProjectDetailFilter(ProjectBrowseFilterFamily.weaveStructure, [
        'structure',
      ]),
      ProjectDetailFilter(ProjectBrowseFilterFamily.materialRole, [
        'materials',
        'role',
      ], label: 'Material role'),
      ProjectDetailFilter(ProjectBrowseFilterFamily.settUnit, [
        'sett',
        'unit',
      ], label: 'Sett unit'),
    ],
    ProjectOptionCatalogs.feltingCraftToken: const [
      ProjectDetailFilter(ProjectBrowseFilterFamily.feltingMethod, ['method']),
      ProjectDetailFilter(ProjectBrowseFilterFamily.feltingForm, ['form']),
      ProjectDetailFilter(ProjectBrowseFilterFamily.needleGauge, [
        'needleGauges',
      ], parent: ProjectBrowseFilterFamily.feltingMethod),
    ],
  };
}

Iterable<String> _valuesAtPath(Object? value, List<String> path) sync* {
  if (value is List) {
    for (final entry in value) {
      yield* _valuesAtPath(entry, path);
    }
  } else if (value is Map && path.isNotEmpty) {
    yield* _valuesAtPath(value[path.first], path.sublist(1));
  } else if (value is String && path.isEmpty) {
    yield value;
  }
}

bool _overlaps(Iterable<String> actual, List<String> selected) {
  final expected = selected.map((value) => value.toLowerCase()).toSet();
  return actual.any((value) {
    final item = value.toLowerCase();
    if (expected.contains(item)) return true;
    // Older records may store the bare value for the same catalog token.
    return !item.contains('#') &&
        expected.any((option) => option.endsWith('#$item'));
  });
}
