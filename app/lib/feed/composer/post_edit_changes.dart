import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';

/// The ordinary payload builder normalizes material text. Carry only the
/// editor's retained indices across that normalization, never raw extensions.
Project restorePublishedProjectEntries({
  required PostEditSeed seed,
  required Project payload,
  required Map<String, dynamic> formValues,
}) {
  final materials = payload.common.materials ?? const <ProjectMaterial>[];
  final source =
      (formValues[ProjectComposerFields.materials] as Iterable?)
          ?.whereType<ProjectMaterial>()
          .where((material) => material.text.trim().isNotEmpty)
          .toList() ??
      const <ProjectMaterial>[];
  if (source.length != materials.length) {
    throw const FormatException('Incomplete project material form');
  }
  final rawCommon =
      (seed.editable['project']! as Map<String, Object?>)['common']!
          as Map<String, Object?>;
  final limit = (rawCommon['materials'] as List? ?? const []).length;
  final restored = <ProjectMaterial>[];
  for (var i = 0; i < materials.length; i++) {
    final index = source[i].extraFields['existingIndex'];
    if (index != null && (index is! int || index < 0 || index >= limit)) {
      throw const FormatException('Invalid retained project material');
    }
    restored.add(
      materials[i].copyWith(
        extraFields: {'existingIndex': ?index},
      ),
    );
  }
  final originalCrafts = rawCommon['additionalCrafts'] as List? ?? const [];
  final craftIndices = <String, int>{};
  for (final raw in originalCrafts.cast<Map<String, Object?>>()) {
    final index = raw['existingIndex'];
    if (index is! int || index < 0 || index >= originalCrafts.length) {
      throw const FormatException('Invalid retained additional craft');
    }
    craftIndices[raw['craftType']! as String] = index;
  }
  final crafts = payload.common.additionalCrafts
      ?.map(
        (craft) => ProjectCraft(
          craftType: craft.craftType,
          extraFields: {
            if (craftIndices.containsKey(craft.craftType))
              'existingIndex': craftIndices[craft.craftType],
          },
        ),
      )
      .toList();
  return payload.copyWith(
    common: payload.common.copyWith(
      materials: payload.common.materials == null ? null : restored,
      additionalCrafts: crafts,
      pattern:
          payload.common.pattern ??
          ((rawCommon['pattern'] as Map<String, Object?>?)?['selfDrafted']
                      is bool &&
                  formValues[ProjectComposerFields.patternSelfDrafted] == false
              ? const ProjectPattern(selfDrafted: false)
              : null),
    ),
  );
}

typedef _ObjectDiff = Map<String, Object?> Function(
  Map<String, Object?>,
  Map<String, Object?>,
);

/// Compare supported form values, not a raw record replacement. Normalized
/// unchanged form defaults must not become new public fields.
Map<String, Object?> buildPublishedProjectPatch({
  required Project initial,
  required Project current,
}) {
  final before = initial.toCreateMap();
  final after = current.toCreateMap();
  final common = _diffKnownObject(
    before['common'] as Map<String, Object?>,
    after['common'] as Map<String, Object?>,
    const [
      'craftType',
      'status',
      'title',
      'duration',
      'colors',
      'designTags',
      'pattern',
    ],
    {
      'pattern': (before, after) => _diffKnownObject(before, after, const [
        'url',
        'name',
        'nameFacets',
        'difficulty',
        'selfDrafted',
        'designer',
        'designerFacets',
        'publisher',
        'publisherFacets',
      ]),
    },
  );
  final beforeMaterials = _knownEntries(
    (before['common'] as Map<String, Object?>)['materials'],
    const ['text', 'facets'],
  );
  final afterMaterials = _knownEntries(
    (after['common'] as Map<String, Object?>)['materials'],
    const ['text', 'facets'],
  );
  if (!_sameEditValue(beforeMaterials, afterMaterials)) {
    common['materials'] = afterMaterials == null
        ? null
        : _retainedEntries(beforeMaterials ?? const [], afterMaterials, const [
            'text',
            'facets',
          ]);
  }
  final beforeCrafts = _knownEntries(
    (before['common'] as Map<String, Object?>)['additionalCrafts'],
    const ['craftType'],
  );
  final afterCrafts = _knownEntries(
    (after['common'] as Map<String, Object?>)['additionalCrafts'],
    const ['craftType'],
  );
  if (!_sameEditValue(beforeCrafts, afterCrafts)) {
    common['additionalCrafts'] = afterCrafts == null
        ? null
        : _retainedEntries(beforeCrafts ?? const [], afterCrafts, const [
            'craftType',
          ]);
  }
  final result = <String, Object?>{if (common.isNotEmpty) 'common': common};
  final beforeDetails = before['details'] as Map<String, Object?>?;
  final afterDetails = after['details'] as Map<String, Object?>?;
  if (beforeDetails == null || _detailKeys(beforeDetails[r'$type']) != null) {
    if (afterDetails == null) {
      if (beforeDetails != null) {
        final sameCraft =
            (before['common'] as Map<String, Object?>)['craftType'] ==
            (after['common'] as Map<String, Object?>)['craftType'];
        if (sameCraft) {
          final details = _diffKnownObject(
            beforeDetails,
            const {},
            _detailKeys(beforeDetails[r'$type'])!
                .where((key) => key != r'$type')
                .toList(),
          );
          if (details.isNotEmpty) result['details'] = details;
        } else {
          result['details'] = null;
        }
      }
    } else {
      final keys = _detailKeys(afterDetails[r'$type']);
      if (keys != null) {
        final sameType = beforeDetails?[r'$type'] == afterDetails[r'$type'];
        final details = _diffKnownObject(
          sameType ? beforeDetails ?? const {} : const {},
          afterDetails,
          keys,
          {
            'gauge': (before, after) => _diffKnownObject(before, after, const [
              'stitches',
              'rows',
              'measurement',
              'unit',
            ]),
          },
        );
        if (details.isNotEmpty) result['details'] = details;
      }
    }
  }
  return result;
}

List<String>? _detailKeys(Object? type) => switch (type) {
  sewingProjectDetailsType => const [
    r'$type',
    'projectType',
    'projectSubtype',
    'sizeMade',
    'fitNotes',
  ],
  knittingProjectDetailsType => const [
    r'$type',
    'projectType',
    'projectSubtype',
    'yarnWeight',
    'needleSizeMm',
    'gauge',
    'finishedSize',
  ],
  crochetProjectDetailsType => const [
    r'$type',
    'projectType',
    'projectSubtype',
    'yarnWeight',
    'hookSizeMm',
    'gauge',
    'finishedSize',
  ],
  quiltingProjectDetailsType => const [
    r'$type',
    'projectType',
    'projectSubtype',
    'size',
    'piecingTechnique',
    'quiltingMethod',
  ],
  _ => null,
};

List<Map<String, Object?>>? _knownEntries(Object? raw, List<String> keys) {
  if (raw is! List) return null;
  return [
    for (final entry in raw.cast<Map<String, Object?>>())
      {
        if (entry.containsKey('existingIndex'))
          'existingIndex': entry['existingIndex'],
        for (final key in keys)
          if (entry.containsKey(key)) key: entry[key],
      },
  ];
}

List<Map<String, Object?>> _retainedEntries(
  List<Map<String, Object?>> before,
  List<Map<String, Object?>> after,
  List<String> keys,
) {
  final originals = {
    for (final entry in before)
      if (entry['existingIndex'] is int) entry['existingIndex']: entry,
  };
  return [
    for (final entry in after)
      {
        if (entry.containsKey('existingIndex'))
          'existingIndex': entry['existingIndex'],
        ..._diffKnownObject(
          originals[entry['existingIndex']] ?? const {},
          entry,
          keys,
        ),
      },
  ];
}

Map<String, Object?> _diffKnownObject(
  Map<String, Object?> before,
  Map<String, Object?> after,
  List<String> keys, [
  Map<String, _ObjectDiff> nested = const {},
]) {
  final result = <String, Object?>{};
  for (final key in keys) {
    // Existing annotations survive unchanged text. Rebuilding an editor is
    // not permission to replace an untouched authored annotation object.
    final text = const {
      'facets': 'text',
      'nameFacets': 'name',
      'designerFacets': 'designer',
      'publisherFacets': 'publisher',
    }[key];
    if (text != null && before[text] == after[text]) continue;
    final old = before[key];
    final value = after[key];
    if (_sameEditValue(old, value)) continue;
    final diff = nested[key];
    if (diff != null &&
        (value is Map<String, Object?> ||
            (key == 'pattern' && old is Map<String, Object?>))) {
      final changes = diff(
        old is Map<String, Object?> ? old : const {},
        value is Map<String, Object?> ? value : const {},
      );
      if (changes.isNotEmpty) result[key] = changes;
    } else {
      result[key] = value;
    }
  }
  return result;
}

bool _sameEditValue(Object? before, Object? after) {
  if (identical(before, after)) return true;
  if (before is List && after is List) {
    return before.length == after.length &&
        [
          for (var i = 0; i < before.length; i++)
            _sameEditValue(before[i], after[i]),
        ].every((same) => same);
  }
  if (before is Map<String, Object?> && after is Map<String, Object?>) {
    return before.length == after.length &&
        before.keys.every(
          (key) =>
              after.containsKey(key) && _sameEditValue(before[key], after[key]),
        );
  }
  return before == after;
}
