import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';

enum DraftRepositoryFailureReason {
  invalidRequest,
  conflict,
  unavailable,
  storageUnavailable,
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

final class DraftWriteRequest {
  DraftWriteRequest({
    required this.id,
    required this.owner,
    required this.kind,
    required this.content,
    required this.schedule,
    required List<DraftMediaWrite> orderedMedia,
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
