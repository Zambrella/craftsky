import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/projects/models/project.dart';

/// Frozen local presentation, prepared before sending the edit. The mutation
/// publishes it only after confirmation; it is never public indexing evidence.
Post postEditPresentation(PostEditSeed seed, PostEditChanges changes) {
  final patch = changes.toMap();
  final record = _merge(seed.editable, patch)! as Map<String, Object?>;
  final original = seed.post;
  final images = record['images'] as List<Object?>?;
  final embed = record['embed'] as Map<String, Object?>?;
  final external = embed?['external'] as Map<String, Object?>?;
  final video = embed?['video'] as Map<String, Object?>?;
  final videoCid = video == null ? null : _cid(video);
  final retainedVideo = videoCid == original.video?.cid.value;
  final owner = original.author.did.value;
  final videoBase = videoCid == null
      ? null
      : 'https://video.bsky.app/watch/${Uri.encodeComponent(owner)}/$videoCid';
  return original.copyWith(
    text: record['text']! as String,
    facets: patch.containsKey('text') && !patch.containsKey('facets')
        ? null
        : (record['facets'] as List<Object?>?)?.cast<Map<String, dynamic>>(),
    langs:
        (record['langs'] as List<Object?>?)?.cast<String>() ?? const <String>[],
    sponsored: record['sponsored']! as bool,
    project: record['project'] == null
        ? null
        : ProjectMapper.fromMap(
            Map<String, dynamic>.from(
              record['project']! as Map<String, Object?>,
            ),
          ),
    images: images == null
        ? null
        : [
            for (final raw in images.cast<Map<String, Object?>>())
              _postImage(owner, raw, original.images),
          ],
    external: external == null
        ? null
        : PostExternal(
            uri: external['uri']! as String,
            title: external['title']! as String,
            description: external['description'] as String? ?? '',
            thumb: external['thumb'] == null
                ? null
                : _externalThumb(
                    owner,
                    external['thumb']! as Map<String, Object?>,
                    original.external?.thumb,
                  ),
          ),
    video: video == null
        ? null
        : PostVideo(
            cid: videoCid!,
            mime: video['mimeType']! as String,
            size: video['size']! as int,
            alt: embed?['alt'] as String?,
            aspectRatio: _ratio(embed?['aspectRatio']),
            playlist: retainedVideo
                ? original.video?.playlist
                : '$videoBase/playlist.m3u8',
            thumbnail: retainedVideo
                ? original.video?.thumbnail
                : '$videoBase/thumbnail.jpg',
            captions: retainedVideo
                ? original.video!.captions
                : const <PostVideoCaption>[],
          ),
  );
}

// Apply only the supported frozen editor projection. Retained array entries
// carry source indices; omitted leaves survive, explicit null clears them.
Object? _merge(Object? base, Object? patch) {
  if (patch is Map<String, Object?>) {
    final source = base is Map<String, Object?>
        ? base
        : const <String, Object?>{};
    final replaceType =
        patch.containsKey(r'$type') && patch[r'$type'] != source[r'$type'];
    return <String, Object?>{
      if (!replaceType)
        for (final entry in source.entries)
          if (entry.key != 'existingIndex') entry.key: entry.value,
      for (final entry in patch.entries)
        if (entry.key != 'existingIndex')
          entry.key: _merge(
            replaceType ? null : source[entry.key],
            entry.value,
          ),
    };
  }
  if (patch is List<Object?>) {
    final source = base is List<Object?> ? base : const <Object?>[];
    return [
      for (final value in patch)
        _merge(
          value is Map<String, Object?> && value['existingIndex'] is int
              ? source[value['existingIndex']! as int]
              : null,
          value,
        ),
    ];
  }
  return patch;
}

String _cid(Map<String, Object?> blob) =>
    (blob['ref']! as Map<String, Object?>)[r'$link']! as String;

PostImageAspectRatio? _ratio(Object? value) => value is Map<String, Object?>
    ? PostImageAspectRatio(
        width: value['width']! as int,
        height: value['height']! as int,
      )
    : null;

String _imageUrl(String owner, Map<String, Object?> blob, String kind) {
  final extension = switch (blob['mimeType']) {
    'image/png' => 'png',
    'image/webp' => 'webp',
    _ => 'jpeg',
  };
  return 'https://cdn.bsky.app/img/$kind/plain/$owner/${_cid(blob)}@$extension';
}

PostExternalThumb _externalThumb(
  String owner,
  Map<String, Object?> blob,
  PostExternalThumb? retained,
) => PostExternalThumb(
  cid: _cid(blob),
  mime: blob['mimeType']! as String,
  size: blob['size']! as int,
  url: retained?.cid.value == _cid(blob)
      ? retained!.url
      : _imageUrl(owner, blob, 'feed_thumbnail'),
);

PostImage _postImage(
  String owner,
  Map<String, Object?> value,
  List<PostImage>? originals,
) {
  final blob = value['image']! as Map<String, Object?>;
  final cid = _cid(blob);
  final retained = originals
      ?.where((image) => image.cid.value == cid)
      .firstOrNull;
  return PostImage(
    cid: cid,
    mime: blob['mimeType']! as String,
    size: blob['size']! as int,
    alt: value['alt'] as String? ?? '',
    aspectRatio: _ratio(value['aspectRatio']),
    thumb: retained?.thumb ?? _imageUrl(owner, blob, 'feed_thumbnail'),
    fullsize: retained?.fullsize ?? _imageUrl(owner, blob, 'feed_fullsize'),
  );
}
