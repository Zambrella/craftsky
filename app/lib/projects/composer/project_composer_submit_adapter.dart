import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/models/project.dart';

typedef ProjectFacetGenerator =
    Future<List<Map<String, dynamic>>> Function(
      String text, {
      bool includeMentions,
      bool includeLinks,
      bool includeTags,
    });

class ProjectComposerSubmitArguments {
  const ProjectComposerSubmitArguments({
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
  final projectWithPatternFacets = await _projectWithPatternFacets(
    project,
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

  final nextPattern = ProjectPattern(
    url: pattern.url,
    name: pattern.name,
    nameFacets: nameFacets,
    difficulty: pattern.difficulty,
    designer: pattern.designer,
    designerFacets: designerFacets,
    publisher: pattern.publisher,
    publisherFacets: publisherFacets,
  );
  return _projectWith(project, pattern: nextPattern, materials: materials);
}

Project _projectWith(
  Project project, {
  ProjectPattern? pattern,
  List<ProjectMaterial>? materials,
}) => Project(
  common: ProjectCommon(
    craftType: project.common.craftType,
    status: project.common.status,
    title: project.common.title,
    duration: project.common.duration,
    pattern: pattern,
    materials: materials,
    colors: project.common.colors,
    designTags: project.common.designTags,
    tags: project.common.tags,
  ),
  details: project.details,
);

List<ProjectMaterial>? _preserveUnchangedMaterialFacets(
  List<ProjectMaterial>? materials,
  List<ProjectMaterial>? existing,
) {
  if (materials == null) return null;
  return [
    for (var index = 0; index < materials.length; index++)
      ProjectMaterial(
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
