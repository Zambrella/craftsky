import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/models/project.dart';

typedef ProjectFacetGenerator =
    Future<List<Map<String, dynamic>>> Function(
      String text,
    );

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
  return ProjectComposerSubmitArguments(
    text: trimmedText,
    project: project,
    reply: null,
    images: imagesState.toCreatePostImages(),
    facets: facets.isEmpty ? null : facets,
  );
}
