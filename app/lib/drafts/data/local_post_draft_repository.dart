import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';

enum DraftRepositoryFailureReason {
  invalidRequest,
  conflict,
  unavailable,
  storageUnavailable,
  quotaExceeded,
}

final class DraftRepositoryException implements Exception {
  const DraftRepositoryException(this.reason);

  final DraftRepositoryFailureReason reason;

  @override
  String toString() => 'DraftRepositoryException(${reason.name})';
}

sealed class DraftMediaWrite {
  const DraftMediaWrite({required this.mediaId, required this.altText});

  final String mediaId;
  final String altText;
}

final class ExistingStoredMedia extends DraftMediaWrite {
  const ExistingStoredMedia({
    required super.mediaId,
    required this.storageRevision,
    required this.expectedSha256,
    required this.displayFileName,
    required this.mimeType,
    required this.byteLength,
    required this.width,
    required this.height,
    required super.altText,
  });

  final String storageRevision;
  final String expectedSha256;
  final String displayFileName;
  final String mimeType;
  final int byteLength;
  final int width;
  final int height;

  @override
  String toString() => 'ExistingStoredMedia(<redacted>)';
}

final class PreparedDraftMedia extends DraftMediaWrite {
  const PreparedDraftMedia({
    required super.mediaId,
    required this.displayFileName,
    required this.mimeType,
    required this.bytes,
    required this.width,
    required this.height,
    required super.altText,
  });

  final String displayFileName;
  final String mimeType;
  final Uint8List bytes;
  final int width;
  final int height;

  @override
  String toString() => 'PreparedDraftMedia(<redacted>)';
}

sealed class DraftVideoWrite {
  const DraftVideoWrite();
}

final class ExistingStoredVideo extends DraftVideoWrite {
  const ExistingStoredVideo({
    required this.storageRevision,
    required this.expectedSourceSha256,
    required this.expectedPosterSha256,
    required this.altText,
  });

  final String storageRevision;
  final String expectedSourceSha256;
  final String expectedPosterSha256;
  final String altText;
}

final class PreparedDraftVideo extends DraftVideoWrite {
  const PreparedDraftVideo({
    required this.displayFileName,
    required this.mimeType,
    required this.byteLength,
    required this.openSource,
    required this.width,
    required this.height,
    required this.duration,
    required this.altText,
    required this.posterMimeType,
    required this.posterBytes,
  });

  final String displayFileName;
  final String mimeType;
  final int byteLength;
  final Stream<List<int>> Function() openSource;
  final int width;
  final int height;
  final Duration? duration;
  final String altText;
  final String posterMimeType;
  final Uint8List posterBytes;

  @override
  String toString() => 'PreparedDraftVideo(<redacted>)';
}

final class DraftWriteRequest {
  DraftWriteRequest({
    required this.id,
    required this.owner,
    required this.kind,
    required this.content,
    required this.schedule,
    required List<DraftMediaWrite> orderedMedia,
    this.video,
    this.createdAt,
    this.expectedRevision,
  }) : orderedMedia = List.unmodifiable(orderedMedia);

  final String id;
  final AccountKey owner;
  final LocalPostDraftKind kind;
  final DateTime? createdAt;
  final int? expectedRevision;
  final LocalDraftContent content;
  final DraftScheduleIntent schedule;
  final List<DraftMediaWrite> orderedMedia;
  final DraftVideoWrite? video;

  @override
  String toString() => 'DraftWriteRequest(<redacted>)';
}

abstract interface class LocalPostDraftRepository {
  Future<List<LocalPostDraft>> list();

  Future<LocalPostDraft> get(String draftId);

  Future<LocalPostDraft> save(DraftWriteRequest request);

  Future<Uint8List> readMedia(String draftId, String mediaId);

  Future<void> delete(String draftId);
}

abstract interface class LocalVideoDraftRepository {
  Stream<List<int>> openVideoSource(String draftId);

  Future<Uint8List> readVideoPoster(String draftId);
}
