import 'dart:typed_data';

import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:crypto/crypto.dart';

final class PlannedDraftMediaWrite {
  const PlannedDraftMediaWrite({required this.descriptor, required this.bytes});

  final DraftMediaDescriptor descriptor;
  final Uint8List bytes;

  String get mediaId => descriptor.mediaId;

  @override
  String toString() => 'PlannedDraftMediaWrite(<redacted>)';
}

/// Pure plan for the immutable-media portion of one draft save.
final class DraftUpdatePlan {
  DraftUpdatePlan._({
    required List<DraftMediaDescriptor> nextMedia,
    required List<PlannedDraftMediaWrite> mediaWrites,
    required List<String> cleanupFileNames,
  }) : nextMedia = List.unmodifiable(nextMedia),
       mediaWrites = List.unmodifiable(mediaWrites),
       cleanupFileNames = List.unmodifiable(cleanupFileNames);

  factory DraftUpdatePlan.build({
    required List<DraftMediaDescriptor> currentMedia,
    required List<DraftMediaWrite> orderedMedia,
    required String Function() nextStorageRevision,
  }) {
    if (orderedMedia.length > 4) {
      throw const DraftRepositoryException(
        DraftRepositoryFailureReason.invalidRequest,
      );
    }
    final currentById = {
      for (final media in currentMedia) media.mediaId: media,
    };
    final next = <DraftMediaDescriptor>[];
    final writes = <PlannedDraftMediaWrite>[];
    final retainedFileNames = <String>{};
    final nextIds = <String>{};

    for (var order = 0; order < orderedMedia.length; order++) {
      final media = orderedMedia[order];
      if (!nextIds.add(media.mediaId)) {
        throw const DraftRepositoryException(
          DraftRepositoryFailureReason.invalidRequest,
        );
      }
      switch (media) {
        case ExistingStoredMedia():
          final stored = currentById[media.mediaId];
          final matches =
              stored != null &&
              stored.storageRevision == media.storageRevision &&
              stored.sha256 == media.expectedSha256 &&
              stored.byteLength == media.byteLength &&
              stored.mimeType == media.mimeType &&
              stored.width == media.width &&
              stored.height == media.height;
          if (!matches) {
            throw const DraftRepositoryException(
              DraftRepositoryFailureReason.invalidRequest,
            );
          }
          retainedFileNames.add(stored.storageFileName);
          next.add(
            DraftMediaDescriptor(
              mediaId: stored.mediaId,
              storageRevision: stored.storageRevision,
              storageFileName: stored.storageFileName,
              displayFileName: media.displayFileName,
              mimeType: stored.mimeType,
              byteLength: stored.byteLength,
              sha256: stored.sha256,
              width: stored.width,
              height: stored.height,
              altText: media.altText,
              order: order,
            ),
          );
        case PreparedDraftMedia():
          final revision = nextStorageRevision();
          final extension = switch (media.mimeType) {
            'image/jpeg' => 'jpg',
            'image/png' => 'png',
            _ => throw const DraftRepositoryException(
              DraftRepositoryFailureReason.invalidRequest,
            ),
          };
          final descriptor = DraftMediaDescriptor(
            mediaId: media.mediaId,
            storageRevision: revision,
            storageFileName: '${media.mediaId}-$revision.$extension',
            displayFileName: media.displayFileName,
            mimeType: media.mimeType,
            byteLength: media.bytes.length,
            sha256: sha256.convert(media.bytes).toString(),
            width: media.width,
            height: media.height,
            altText: media.altText,
            order: order,
          );
          descriptor.validate();
          retainedFileNames.add(descriptor.storageFileName);
          next.add(descriptor);
          writes.add(
            PlannedDraftMediaWrite(descriptor: descriptor, bytes: media.bytes),
          );
      }
    }

    final cleanup = [
      for (final media in currentMedia)
        if (!retainedFileNames.contains(media.storageFileName))
          media.storageFileName,
    ];
    return DraftUpdatePlan._(
      nextMedia: next,
      mediaWrites: writes,
      cleanupFileNames: cleanup,
    );
  }

  final List<DraftMediaDescriptor> nextMedia;
  final List<PlannedDraftMediaWrite> mediaWrites;
  final List<String> cleanupFileNames;

  @override
  String toString() => 'DraftUpdatePlan(<redacted>)';
}
