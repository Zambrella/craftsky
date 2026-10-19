import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/models/project.dart';

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
}) async {
  final trimmedText = text.trim();
  final facets = existingText?.trim() == trimmedText
      ? existingFacets ?? const <Map<String, dynamic>>[]
      : await generateFacets(trimmedText);
  final preserved = _preserveProjectExtensions(project, existingProject);
  final projectWithPatternFacets = await _projectWithPatternFacets(
    preserved,
    generateFacets,
    existingProject: existingProject,
  );
  return ProjectComposerSubmitArguments(
    text: trimmedText,
    langs: List.unmodifiable(langs),
    project: projectWithPatternFacets,
    reply: null,
    images: materializeImagesFromState
        ? imagesState.toCreatePostImages()
        : null,
    facets: facets.isEmpty ? null : facets,
  );
}

Future<Project> _projectWithPatternFacets(
  Project project,
  ProjectFacetGenerator generateFacets, {
  Project? existingProject,
}) async {
  final pattern = project.common.pattern;
  final existingPattern = existingProject?.common.pattern;
  final materials = _preserveUnchangedMaterialFacets(
    project.common.materials,
    existingProject?.common.materials,
  );
  if (pattern == null) {
    if (identical(materials, project.common.materials)) return project;
    return _projectWith(project, materials: materials);
  }

  final nameFacets = pattern.name == existingPattern?.name
      ? existingPattern?.nameFacets
      : await _fieldFacets(
          pattern.name,
          generateFacets,
          includeMentions: false,
          includeLinks: false,
          includeTags: true,
        );
  final designerFacets = pattern.designer == existingPattern?.designer
      ? existingPattern?.designerFacets
      : await _fieldFacets(
          pattern.designer,
          generateFacets,
          includeMentions: true,
          includeLinks: false,
          includeTags: false,
        );
  final publisherFacets = pattern.publisher == existingPattern?.publisher
      ? existingPattern?.publisherFacets
      : await _fieldFacets(
          pattern.publisher,
          generateFacets,
          includeMentions: true,
          includeLinks: false,
          includeTags: false,
        );

  final nextPattern = pattern.copyWith(
    extraFields: {...?existingPattern?.extraFields, ...pattern.extraFields},
    nameFacets: nameFacets,
    designerFacets: designerFacets,
    publisherFacets: publisherFacets,
  );
  return _projectWith(project, pattern: nextPattern, materials: materials);
}

Project _projectWith(
  Project project, {
  ProjectPattern? pattern,
  List<ProjectMaterial>? materials,
}) => project.copyWith.common(pattern: pattern, materials: materials);

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
    final oldExtra = switch (existing.details) {
      KnittingProjectDetails(:final extraFields) => extraFields,
      CrochetProjectDetails(:final extraFields) => extraFields,
      SewingProjectDetails(:final extraFields) => extraFields,
      QuiltingProjectDetails(:final extraFields) => extraFields,
      _ => <String, dynamic>{},
    };
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
    } else if (details == null && existing.details is UnknownProjectDetails) {
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
      duration: common.duration ?? original.duration,
      pattern: preservedPattern,
      tags: common.tags ?? original.tags,
    ),
    details: details,
  );
}
