import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_detail_fields.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:craftsky_app/shared/rich_text/hashtag_budget.dart';

typedef ProjectFacetGenerator = Future<List<Map<String, dynamic>>> Function(
  String text, {
  bool includeMentions,
  bool includeLinks,
  bool includeTags,
});

class ProjectComposerSubmitArguments {
  const new({
    required this.text,
    required this.langs,
    required this.project,
    required this.reply,
    required this.images,
    required this.facets,
  });

  final String text;
  final List<String> langs;
  final Project project;
  final PostReply? reply;
  final List<CreatePostImage>? images;
  final List<Map<String, dynamic>>? facets;
}

Future<ProjectComposerSubmitArguments> buildProjectComposerSubmitArguments({
  required String text,
  required List<String> langs,
  required Project project,
  required ComposerImagesState imagesState,
  required ProjectFacetGenerator generateFacets,
  String? existingText,
  List<Map<String, dynamic>>? existingFacets,
  Project? existingProject,
  bool materializeImagesFromState = true,
  bool preserveSavedFacets = false,
}) async {
  final submittedText = preserveSavedFacets && text == existingText
      ? text
      : text.trim();
  final sameText = existingText == submittedText;
  final facets = sameText
      ? existingFacets
      : await generateFacets(submittedText);
  final preserved = _preserveProjectExtensions(project, existingProject);
  final common = preserved.common;

  final accepted = <String>{};
  bool acceptTag(String raw) {
    final tag = raw.trim().toLowerCase();
    return tag.isNotEmpty &&
        (accepted.contains(tag) ||
            (accepted.length < maxPostHashtags && accepted.add(tag)));
  }

  List<Map<String, dynamic>>? fieldFacets(
    String text,
    List<Map<String, dynamic>>? facets, {
    bool unchanged = false,
  }) {
    // An unchanged saved field keeps its authored facets. New tags can only
    // spend a slot after overlap with other facets is ruled out.
    final retained = preserveSavedFacets && unchanged;
    final eligible = retained
        ? facets ?? const <Map<String, dynamic>>[]
        : budgetedFacets(
            text,
            facets,
            hashtagsInText(text).map((tag) => tag.toLowerCase()).toSet(),
          );
    final limited = <Map<String, dynamic>>[];
    for (final facet in eligible) {
      final rawFeatures = facet['features'];
      if (rawFeatures is! List) continue;
      final features = rawFeatures.where((feature) {
        if (feature is! Map || feature[r'$type'] != FacetFeatureType.tag) {
          return true;
        }
        final tag = feature['tag'];
        return tag is String && acceptTag(tag);
      }).toList();
      if (features.isNotEmpty) limited.add({...facet, 'features': features});
    }
    if (retained) return facets;
    return limited.isEmpty ? null : limited;
  }

  final limitedFacets = fieldFacets(submittedText, facets, unchanged: sameText);
  final tags = common.tags?.where(acceptTag).toList();
  final existingCommon = existingProject?.common;
  final retainedPatterns = {
    for (final pattern
        in existingCommon?.effectivePatterns ?? const <ProjectPattern>[])
      if (pattern.extraFields['existingIndex'] is int)
        pattern.extraFields['existingIndex']: pattern,
  };

  Future<ProjectPattern> submitPattern(
    ProjectPattern pattern,
    ProjectPattern? previous,
  ) async {
    final annotated = await _patternWithFacets(
      pattern,
      generateFacets,
      previous,
    );
    return annotated.copyWith(
      nameFacets: fieldFacets(
        annotated.name ?? '',
        annotated.nameFacets,
        unchanged: annotated.name == previous?.name,
      ),
      designerFacets: fieldFacets(
        annotated.designer ?? '',
        annotated.designerFacets,
        unchanged: annotated.designer == previous?.designer,
      ),
      publisherFacets: fieldFacets(
        annotated.publisher ?? '',
        annotated.publisherFacets,
        unchanged: annotated.publisher == previous?.publisher,
      ),
    );
  }

  final patterns = common.patterns == null
      ? null
      : [
          for (final pattern in common.patterns!)
            await submitPattern(
              pattern,
              retainedPatterns[pattern.extraFields['existingIndex']],
            ),
        ];
  final pattern = patterns == null && common.pattern != null
      ? await submitPattern(common.pattern!, existingCommon?.pattern)
      : common.pattern;
  final materials = _preserveUnchangedMaterialFacets(
    common.materials,
    existingCommon?.materials,
  );
  final projectWithFacets = preserved.copyWith.common(
    tags: tags,
    pattern: pattern,
    patterns: patterns,
    materials: materials
        ?.map(
          (material) => material.copyWith(
            facets: fieldFacets(material.text, material.facets),
          ),
        )
        .toList(),
  );
  return ProjectComposerSubmitArguments(
    text: submittedText,
    langs: List.unmodifiable(langs),
    project: projectWithFacets,
    reply: null,
    images: materializeImagesFromState
        ? imagesState.toCreatePostImages()
        : null,
    facets: limitedFacets,
  );
}

Future<ProjectPattern> _patternWithFacets(
  ProjectPattern pattern,
  ProjectFacetGenerator generateFacets,
  ProjectPattern? existingPattern,
) async {
  final nameFacets =
      pattern.nameFacets ??
      (pattern.name == existingPattern?.name
          ? existingPattern?.nameFacets
          : await _fieldFacets(
              pattern.name,
              generateFacets,
              includeMentions: false,
              includeLinks: false,
              includeTags: true,
            ));
  final designerFacets =
      pattern.designerFacets ??
      (pattern.designer == existingPattern?.designer
          ? existingPattern?.designerFacets
          : await _fieldFacets(
              pattern.designer,
              generateFacets,
              includeMentions: true,
              includeLinks: false,
              includeTags: true,
            ));
  final publisherFacets =
      pattern.publisherFacets ??
      (pattern.publisher == existingPattern?.publisher
          ? existingPattern?.publisherFacets
          : await _fieldFacets(
              pattern.publisher,
              generateFacets,
              includeMentions: true,
              includeLinks: false,
              includeTags: true,
            ));

  return pattern.copyWith(
    extraFields: {...?existingPattern?.extraFields, ...pattern.extraFields},
    nameFacets: nameFacets,
    designerFacets: designerFacets,
    publisherFacets: publisherFacets,
  );
}

List<ProjectMaterial>? _preserveUnchangedMaterialFacets(
  List<ProjectMaterial>? materials,
  List<ProjectMaterial>? existing,
) {
  if (materials == null) return null;
  return [
    for (var index = 0; index < materials.length; index++)
      ProjectMaterial(
        extraFields: {
          if (index < (existing?.length ?? 0) &&
              existing![index].text == materials[index].text)
            ...existing[index].extraFields,
          ...materials[index].extraFields,
        },
        text: materials[index].text,
        facets:
            index < (existing?.length ?? 0) &&
                existing![index].text == materials[index].text
            ? existing[index].facets
            : materials[index].facets,
      ),
  ];
}

Future<List<Map<String, dynamic>>?> _fieldFacets(
  String? text,
  ProjectFacetGenerator generateFacets, {
  required bool includeMentions,
  required bool includeLinks,
  required bool includeTags,
}) async {
  if (text == null || text.trim().isEmpty) return null;
  final facets = await generateFacets(
    text,
    includeMentions: includeMentions,
    includeLinks: includeLinks,
    includeTags: includeTags,
  );
  return facets.isEmpty ? null : facets;
}

// Preserve extension data when a scheduled project is rebuilt from form fields.
// Known values are taken exclusively from the form, so clearing remains valid.
Project _preserveProjectExtensions(Project project, Project? existing) {
  if (existing == null) return project;
  final common = project.common;
  final original = existing.common;
  final sameCraft = common.craftType == original.craftType;
  var details = project.details;
  final rawDetails = details == null
      ? null
      : const ProjectDetailsMapper().encode(details) as Map<String, dynamic>;
  final oldDetails = existing.details == null
      ? null
      : const ProjectDetailsMapper().encode(existing.details!)
            as Map<String, dynamic>;
  if (sameCraft && oldDetails != null) {
    final oldExtra = {
      ...switch (existing.details) {
        KnittingProjectDetails(:final extraFields) => extraFields,
        CrochetProjectDetails(:final extraFields) => extraFields,
        SewingProjectDetails(:final extraFields) => extraFields,
        QuiltingProjectDetails(:final extraFields) => extraFields,
        UnknownProjectDetails(:final type, :final raw)
            when type == ProjectDetailFields.typeForCraft(common.craftType) =>
          raw,
        _ => <String, dynamic>{},
      },
    };
    for (final field in ProjectDetailFields.forCraft(common.craftType)) {
      oldExtra.remove(field.key);
    }
    oldExtra
      ..remove('projectType')
      ..remove('projectSubtype');
    if (rawDetails != null && rawDetails[r'$type'] == oldDetails[r'$type']) {
      // Retain unfamiliar detail keys while replacing the known form values.
      final gauge = rawDetails['gauge'];
      final oldGauge = oldDetails['gauge'];
      if (gauge is Map<String, dynamic> && oldGauge is Map) {
        for (final entry in oldGauge.entries) {
          if (!const [
            'stitches',
            'rows',
            'measurement',
            'unit',
          ].contains(entry.key)) {
            gauge.putIfAbsent(entry.key as String, () => entry.value);
          }
        }
      }
      details = const ProjectDetailsMapper().decode({
        ...oldExtra,
        ...rawDetails,
      });
    } else if (details == null &&
        existing.details is UnknownProjectDetails &&
        oldDetails[r'$type'] !=
            ProjectDetailFields.typeForCraft(common.craftType)) {
      details = existing.details;
    } else if (details == null && oldExtra.isNotEmpty) {
      details = const ProjectDetailsMapper().decode({
        r'$type': oldDetails[r'$type'],
        ...oldExtra,
      });
    }
  }
  final pattern = common.pattern;
  final oldPatternExtra =
      original.pattern?.extraFields ?? const <String, dynamic>{};
  final preservedPattern =
      pattern?.copyWith(
        extraFields: {...oldPatternExtra, ...pattern.extraFields},
      ) ??
      (oldPatternExtra.isEmpty
          ? null
          : ProjectPattern(extraFields: oldPatternExtra));
  return project.copyWith(
    extraFields: {...existing.extraFields, ...project.extraFields},
    common: common.copyWith(
      extraFields: {...original.extraFields, ...common.extraFields},
      additionalCrafts: common.additionalCrafts == null
          ? null
          : [
              for (final craft in common.additionalCrafts!)
                ProjectCraft(
                  craftType: craft.craftType,
                  extraFields: {
                    for (final old
                        in original.additionalCrafts ?? const <ProjectCraft>[])
                      if (old.craftType == craft.craftType) ...old.extraFields,
                    ...craft.extraFields,
                  },
                ),
            ],
      pattern: common.pattern == null ? null : preservedPattern,
      tags: common.tags ?? original.tags,
    ),
    details: details,
  );
}
