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
    required this.project,
    required this.reply,
    required this.images,
    required this.facets,
  });

  final String text;
  final Project project;
  final PostReply? reply;
  final List<CreatePostImage>? images;
  final List<Map<String, dynamic>>? facets;
}

Future<ProjectComposerSubmitArguments> buildProjectComposerSubmitArguments({
  required String text,
  required Project project,
  required ComposerImagesState imagesState,
  required ProjectFacetGenerator generateFacets,
}) async {
  final trimmedText = text.trim();
  final facets = await generateFacets(trimmedText);
  final projectWithPatternFacets = await _projectWithPatternFacets(
    project,
    generateFacets,
  );
  return ProjectComposerSubmitArguments(
    text: trimmedText,
    project: projectWithPatternFacets,
    reply: null,
    images: imagesState.toCreatePostImages(),
    facets: facets.isEmpty ? null : facets,
  );
}

Future<Project> _projectWithPatternFacets(
  Project project,
  ProjectFacetGenerator generateFacets,
) async {
  final pattern = project.common.pattern;
  if (pattern == null) return project;

  final nameFacets = await _fieldFacets(
    pattern.name,
    generateFacets,
    includeMentions: false,
    includeLinks: false,
    includeTags: true,
  );
  final designerFacets = await _fieldFacets(
    pattern.designer,
    generateFacets,
    includeMentions: true,
    includeLinks: false,
    includeTags: false,
  );
  final publisherFacets = await _fieldFacets(
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
  return Project(
    common: ProjectCommon(
      craftType: project.common.craftType,
      status: project.common.status,
      title: project.common.title,
      duration: project.common.duration,
      pattern: nextPattern,
      materials: project.common.materials,
      colors: project.common.colors,
      designTags: project.common.designTags,
      tags: project.common.tags,
    ),
    details: project.details,
  );
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
