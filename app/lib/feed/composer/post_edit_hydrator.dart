import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/composer/project_composer_hydrator.dart';
import 'package:craftsky_app/projects/models/project.dart';

Map<String, dynamic> hydratePublishedProjectComposer(PostEditSeed seed) {
  final project = seed.editable['project'];
  if (project is! Map<String, Object?>) {
    throw const FormatException('Incomplete published project seed');
  }
  final values = hydrateScheduledProjectComposer(
    Map<String, dynamic>.from(project),
  );
  final common = project['common']! as Map<String, Object?>;
  final source = common['materials'] as List? ?? const [];
  final materials =
      values[ProjectComposerFields.materials] as List<ProjectMaterial>? ??
      const [];
  if (source.length != materials.length) {
    throw const FormatException('Incomplete published material seed');
  }
  final seen = <int>{};
  values[ProjectComposerFields.materials] = [
    for (var position = 0; position < materials.length; position++)
      _publishedMaterial(
        materials[position],
        source[position] as Map<String, Object?>,
        source.length,
        seen,
      ),
  ];
  return values;
}

ProjectMaterial _publishedMaterial(
  ProjectMaterial material,
  Map<String, Object?> source,
  int length,
  Set<int> seen,
) {
  final index = source['existingIndex']! as int;
  if (index < 0 || index >= length || !seen.add(index)) {
    throw const FormatException('Incomplete published material seed');
  }
  return ProjectMaterial(
    text: material.text,
    facets: material.facets,
    extraFields: {'existingIndex': index},
  );
}

/// Seeds published references, never local bytes or an ordinary saved draft.
List<ComposerImageDraft> hydratePublishedPostImages(PostEditSeed seed) {
  final images = seed.editable['images'] as List? ?? const [];
  final visible = seed.post.images ?? const [];
  if (images.length != visible.length) {
    throw const FormatException('Incomplete published image seed');
  }
  return [
    for (var position = 0; position < images.length; position++)
      _publishedImage(
        seed,
        images[position] as Map<String, Object?>,
        visible[position].fullsize ?? visible[position].thumb,
      ),
  ];
}

ComposerImageDraft _publishedImage(
  PostEditSeed seed,
  Map<String, Object?> value,
  String? url,
) {
  final index = value['existingIndex']! as int;
  if (index < 0 || url == null || url.isEmpty) {
    throw const FormatException('Incomplete published image seed');
  }
  final image = value['image']! as Map<String, Object?>;
  final ratio = value['aspectRatio'] as Map<String, Object?>?;
  return ComposerImageDraft(
    id: 'published:${seed.baseCid}:$index',
    fileName: 'Published image',
    mimeType: image['mimeType']! as String,
    altText: value['alt'] as String? ?? '',
    previewAspectRatio: ratio == null
        ? null
        : CreatePostImageAspectRatio(
            width: ratio['width']! as int,
            height: ratio['height']! as int,
          ),
    phase: PublishedImageReady(existingIndex: index, url: url),
  );
}
