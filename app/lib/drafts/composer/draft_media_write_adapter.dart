import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';

List<DraftMediaWrite> draftMediaWritesFromComposer(
  List<ComposerImageDraft> images,
) {
  return [
    for (final image in images) _draftMediaWrite(image),
  ];
}

DraftMediaWrite _draftMediaWrite(ComposerImageDraft image) {
  final phase = image.phase;
  if (phase is! ImageReady) {
    throw const DraftRepositoryException(
      DraftRepositoryFailureReason.invalidRequest,
    );
  }
  final origin = phase.storedOrigin;
  if (origin != null &&
      origin.mediaId == image.id &&
      origin.sha256 == phase.sha256 &&
      origin.byteLength == phase.bytes.length) {
    return ExistingStoredMedia(
      mediaId: image.id,
      storageRevision: origin.storageRevision,
      expectedSha256: origin.sha256,
      displayFileName: image.fileName,
      mimeType: phase.mimeType,
      byteLength: origin.byteLength,
      width: phase.width,
      height: phase.height,
      altText: image.altText,
    );
  }
  return PreparedDraftMedia(
    mediaId: image.id,
    displayFileName: image.fileName,
    mimeType: phase.mimeType,
    bytes: phase.bytes,
    width: phase.width,
    height: phase.height,
    altText: image.altText,
  );
}
